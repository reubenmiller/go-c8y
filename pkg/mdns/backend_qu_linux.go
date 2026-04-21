//go:build linux

package mdns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

const (
	mdnsIPv4Addr = "224.0.0.251:5353"
	mdnsIPv6Addr = "[ff02::fb]:5353"
)

// browseWithQU sends mDNS queries with the QU (Unicast-response) bit set from
// an ephemeral UDP port so that mDNS responders send their answers directly
// back as unicast UDP. This avoids any conflict with systemd-resolved or avahi
// holding port 5353.
//
// Reference: RFC 6762 §5.4 — when the source port is not 5353, or the QU bit
// is set, responders SHOULD reply via unicast to the source address:port.
func (s *Scanner) browseWithQU(ctx context.Context) (<-chan ServiceInstance, error) {
	// Bind to an ephemeral UDP port on the wildcard address.
	conn, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		return nil, fmt.Errorf("mdns QU: listen: %w", err)
	}

	dest, err := net.ResolveUDPAddr("udp4", mdnsIPv4Addr)
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Build PTR query with the QU bit set (high bit of the QCLASS field).
	fqdn := dns.Fqdn(s.opts.ServiceType + "." + s.opts.Domain)
	m := new(dns.Msg)
	m.SetQuestion(fqdn, dns.TypePTR)
	m.RecursionDesired = false
	// Set QU bit on the question.
	for i := range m.Question {
		m.Question[i].Qclass |= 1 << 15
	}
	pkt, err := m.Pack()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("mdns QU: pack query: %w", err)
	}

	out := make(chan ServiceInstance)
	seen := make(map[string]bool)

	go func() {
		defer close(out)
		defer conn.Close()

		deadline := time.Now().Add(s.opts.Timeout)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		conn.SetDeadline(deadline)

		// Send the query on each suitable interface.
		ifaces, _ := activeMulticastIfaces(s.opts.Ifaces)
		for _, iface := range ifaces {
			addrs, _ := iface.Addrs()
			for _, addr := range addrs {
				ip, _, err := net.ParseCIDR(addr.String())
				if err != nil || ip.IsLoopback() || ip.To4() == nil {
					continue
				}
				src := &net.UDPAddr{IP: ip, Port: 0}
				sock, err := net.ListenUDP("udp4", src)
				if err != nil {
					continue
				}
				sock.WriteToUDP(pkt, dest)
				sock.Close()
			}
		}
		// Also send once from the wildcard so devices respond regardless of
		// which interface they see the query arrive on.
		conn.WriteToUDP(pkt, dest)

		buf := make([]byte, 65535)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			n, _, err := conn.ReadFromUDP(buf)
			if err != nil {
				// deadline exceeded or ctx cancelled
				return
			}

			resp := new(dns.Msg)
			if err := resp.Unpack(buf[:n]); err != nil {
				continue
			}

			instances := extractInstances(resp, s.opts.ServiceType, s.opts.Domain)
			for _, inst := range instances {
				if seen[inst.Name] {
					continue
				}
				seen[inst.Name] = true

				if !s.opts.Quick {
					// Resolve host → IPs via system resolver (goes through
					// systemd-resolved which handles .local names).
					if inst.Host != "" {
						addrs, err := net.DefaultResolver.LookupHost(ctx, inst.Host)
						if err == nil {
							for _, a := range addrs {
								if ip := net.ParseIP(a); ip != nil {
									inst.IPs = append(inst.IPs, ip)
								}
							}
						}
					}
				}

				select {
				case out <- inst:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out, nil
}

// extractInstances parses a DNS message and extracts ServiceInstance data from
// PTR, SRV, and TXT records. Multiple records are correlated by owner name.
func extractInstances(msg *dns.Msg, serviceType, domain string) []ServiceInstance {
	type srvInfo struct {
		host string
		port int
	}
	srvMap := make(map[string]srvInfo)
	txtMap := make(map[string]map[string]string)
	rawTXTMap := make(map[string][]byte)
	var names []string

	fqdn := dns.Fqdn(serviceType + "." + domain)

	all := append(append(msg.Answer, msg.Ns...), msg.Extra...)
	for _, rr := range all {
		switch r := rr.(type) {
		case *dns.PTR:
			if strings.EqualFold(r.Hdr.Name, fqdn) {
				// r.Ptr is the instance FQDN: "name._service._tcp.domain."
				names = append(names, r.Ptr)
			}
		case *dns.SRV:
			srvMap[r.Hdr.Name] = srvInfo{host: r.Target, port: int(r.Port)}
		case *dns.TXT:
			txtMap[r.Hdr.Name] = parseTXTSlice(r.Txt)
			rawTXTMap[r.Hdr.Name] = []byte(strings.Join(r.Txt, "\x00"))
		}
	}

	var out []ServiceInstance
	for _, fqdnName := range names {
		// Strip the service+domain suffix to get the bare instance name.
		suffix := "." + dns.Fqdn(serviceType+"."+domain)
		bare := strings.TrimSuffix(fqdnName, suffix)
		bare = strings.TrimSuffix(bare, ".") // safety

		inst := ServiceInstance{
			Name:   bare,
			TXT:    txtMap[fqdnName],
			RawTXT: rawTXTMap[fqdnName],
		}
		if srv, ok := srvMap[fqdnName]; ok {
			inst.Host = srv.host
			inst.Port = srv.port
		}
		// Fallback: if SRV was absent, derive the hostname from the instance
		// name by appending the search domain (e.g. "mydevice.local.").
		if inst.Host == "" {
			inst.Host = dns.Fqdn(bare + "." + domain)
		}
		out = append(out, inst)
	}
	return out
}
