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

	// quRetryInterval is how often the QU query is re-sent during the scan
	// window.  Some devices wake up late or miss the first probe.
	quRetryInterval = 2 * time.Second
)

// browseWithQU sends mDNS queries with the QU (Unicast-response) bit set.
// Crucially, one persistent UDP socket is opened per interface IP (plus a
// wildcard fallback) and kept open for the entire scan.  Because RFC 6762 §5.4
// requires responders to unicast their answer back to the source address:port
// of the query, those sockets must remain open; closing them immediately after
// sending (as a short-lived socket would) causes every unicast reply to be
// dropped before it can be read.
//
// This avoids conflicts with systemd-resolved, avahi, or the Windows DNS
// Client holding port 5353.
func (s *Scanner) browseWithQU(ctx context.Context) (<-chan ServiceInstance, error) {
	dest, err := net.ResolveUDPAddr("udp4", mdnsIPv4Addr)
	if err != nil {
		return nil, err
	}

	// Build PTR query with the QU bit set (high bit of the QCLASS field).
	fqdn := dns.Fqdn(s.opts.ServiceType + "." + s.opts.Domain)
	m := new(dns.Msg)
	m.SetQuestion(fqdn, dns.TypePTR)
	m.RecursionDesired = false
	for i := range m.Question {
		m.Question[i].Qclass |= 1 << 15 // QU bit
	}
	pkt, err := m.Pack()
	if err != nil {
		return nil, fmt.Errorf("mdns QU: pack query: %w", err)
	}

	// Open one persistent socket per interface IP.  These must stay open so
	// that unicast QU replies directed back to each socket's ephemeral source
	// port are actually received.
	type ifSocket struct {
		conn *net.UDPConn
		ip   net.IP
		name string // interface name for logging
	}
	var sockets []ifSocket

	ifaces, err := activeMulticastIfaces(s.opts.Ifaces)
	if err != nil {
		s.opts.Logger.Printf("mdns QU: interface enumeration: %v", err)
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ip, _, err := net.ParseCIDR(addr.String())
			if err != nil || ip.IsLoopback() || ip.To4() == nil {
				continue
			}
			sock, err := net.ListenUDP("udp4", &net.UDPAddr{IP: ip})
			if err != nil {
				s.opts.Logger.Printf("mdns QU: bind %s: %v", ip, err)
				continue
			}
			sockets = append(sockets, ifSocket{conn: sock, ip: ip, name: iface.Name})
			s.debugf("mdns QU: listening on %s (iface %s)", sock.LocalAddr(), iface.Name)
		}
	}

	// Wildcard fallback socket: catches any multicast replies the OS routes back
	// on the default interface and acts as a safety net when no per-interface
	// sockets could be opened.
	wildcard, err := net.ListenUDP("udp4", &net.UDPAddr{})
	if err != nil {
		for i := range sockets {
			sockets[i].conn.Close()
		}
		return nil, fmt.Errorf("mdns QU: listen wildcard: %w", err)
	}
	s.debugf("mdns QU: wildcard listener on %s", wildcard.LocalAddr())

	deadline := time.Now().Add(s.opts.Timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}

	type udpPacket struct {
		data []byte
		from *net.UDPAddr
	}
	rawCh := make(chan udpPacket, 64)

	// startReader drains conn until deadline or ctx is cancelled, forwarding
	// every received packet to rawCh.  It exits when the socket is closed
	// (by the defers in the main goroutine) or the deadline passes.
	startReader := func(conn *net.UDPConn) {
		buf := make([]byte, 65535)
		for {
			conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, from, err := conn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					if time.Now().After(deadline) || ctx.Err() != nil {
						return
					}
					continue
				}
				return // socket closed or fatal error
			}
			data := make([]byte, n)
			copy(data, buf[:n])
			select {
			case rawCh <- udpPacket{data: data, from: from}:
			case <-ctx.Done():
				return
			}
		}
	}

	sendQuery := func() {
		sent := 0
		for _, sock := range sockets {
			n, werr := sock.conn.WriteToUDP(pkt, dest)
			if werr != nil {
				s.opts.Logger.Printf("mdns QU: send from %s: %v", sock.ip, werr)
			} else {
				s.debugf("mdns QU: sent %d-byte query from %s \u2192 %s (iface %s)", n, sock.ip, dest, sock.name)
				sent++
			}
		}
		if n, werr := wildcard.WriteToUDP(pkt, dest); werr != nil {
			s.opts.Logger.Printf("mdns QU: wildcard send: %v", werr)
		} else {
			s.debugf("mdns QU: sent %d-byte query from wildcard \u2192 %s", n, dest)
			sent++
		}
		if sent == 0 {
			s.opts.Logger.Printf("mdns QU: WARNING: no queries were sent — check interface selection and firewall rules")
		}
	}

	out := make(chan ServiceInstance)

	go func() {
		defer close(out)
		defer wildcard.Close()
		for i := range sockets {
			defer sockets[i].conn.Close()
		}

		// Start one reader goroutine per socket (including wildcard).
		// Closing the sockets in the defers above will unblock any pending
		// ReadFromUDP, causing the readers to exit cleanly.
		go startReader(wildcard)
		for i := range sockets {
			go startReader(sockets[i].conn)
		}

		// Initial query burst.
		sendQuery()

		// Periodic re-send so devices that miss the first probe still respond.
		ticker := time.NewTicker(quRetryInterval)
		defer ticker.Stop()
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()

		seen := make(map[string]bool)
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				return
			case <-ticker.C:
				sendQuery()
			case pkt, ok := <-rawCh:
				if !ok {
					return
				}
				s.debugf("mdns QU: received %d bytes from %s", len(pkt.data), pkt.from)
				resp := new(dns.Msg)
				if err := resp.Unpack(pkt.data); err != nil {
					s.opts.Logger.Printf("mdns QU: unpack error from %s: %v", pkt.from, err)
					continue
				}
				instances := extractInstances(resp, s.opts.ServiceType, s.opts.Domain)
				s.debugf("mdns QU: parsed %d instance(s) from %s", len(instances), pkt.from)
				for _, inst := range instances {
					if seen[inst.Name] {
						continue
					}
					seen[inst.Name] = true
					if !s.opts.Quick && inst.Host != "" {
						addrs, err := net.DefaultResolver.LookupHost(ctx, inst.Host)
						if err != nil {
							if ctx.Err() == nil {
								s.opts.Logger.Printf("mdns QU: resolve %s: %v", inst.Host, err)
							}
						} else {
							for _, a := range addrs {
								if ip := net.ParseIP(a); ip != nil {
									inst.IPs = append(inst.IPs, ip)
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
