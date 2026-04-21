package mdns

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/grandcat/zeroconf"
)

// browseWithPureGo uses the grandcat/zeroconf pure-Go mDNS implementation.
// It joins the mDNS multicast group (224.0.0.251:5353) directly without
// requiring root privileges (port 5353 is above the privileged port boundary).
//
// Only UP, multicast-capable, non-loopback interfaces with a carrier signal
// are used. On Linux this excludes bridge interfaces like docker0 that are UP
// but have no attached containers, which can cause the multicast socket to fail.
func (s *Scanner) browseWithPureGo(ctx context.Context) (<-chan ServiceInstance, error) {
	ifaces, err := activeMulticastIfaces(s.opts.Ifaces)
	if err != nil {
		return nil, err
	}
	if len(ifaces) == 0 {
		return nil, fmt.Errorf("no active multicast-capable network interfaces found")
	}
	s.opts.Logger.Printf("mdns: using interfaces: %v", ifaceNames(ifaces))

	resolver, err := zeroconf.NewResolver(zeroconf.SelectIfaces(ifaces))
	if err != nil {
		return nil, err
	}

	// Derive a timeout context so Browse stops after the configured duration
	// (or when the parent ctx expires, whichever is sooner).
	browseCtx, cancel := context.WithTimeout(ctx, s.opts.Timeout)

	entries := make(chan *zeroconf.ServiceEntry)

	// Browse returns immediately; it populates entries in a background goroutine
	// and closes the channel when browseCtx is done.
	if err := resolver.Browse(browseCtx, s.opts.ServiceType, s.opts.Domain+".", entries); err != nil {
		cancel()
		return nil, err
	}

	out := make(chan ServiceInstance)

	go func() {
		defer close(out)
		defer cancel() // ensure the timeout context is always released

		for entry := range entries {
			var inst ServiceInstance
			if s.opts.Quick {
				// zeroconf provides host, port and IPs from the browse response
				// for free — include them even in quick mode.
				inst = ServiceInstance{
					Name: entry.Instance,
					Host: entry.HostName,
					Port: entry.Port,
				}
				for _, a := range entry.AddrIPv4 {
					inst.IPs = append(inst.IPs, a)
				}
				for _, a := range entry.AddrIPv6 {
					inst.IPs = append(inst.IPs, a)
				}
			} else {
				inst = entryToServiceInstance(entry)
			}
			select {
			case out <- inst:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// entryToServiceInstance converts a zeroconf.ServiceEntry to a ServiceInstance.
func entryToServiceInstance(e *zeroconf.ServiceEntry) ServiceInstance {
	inst := ServiceInstance{
		Name: e.Instance,
		Host: e.HostName,
		Port: e.Port,
		TXT:  parseTXTSlice(e.Text),
	}

	// Build RawTXT as NUL-separated concatenation of TXT records.
	inst.RawTXT = []byte(strings.Join(e.Text, "\x00"))

	for _, a := range e.AddrIPv4 {
		inst.IPs = append(inst.IPs, a)
	}
	for _, a := range e.AddrIPv6 {
		inst.IPs = append(inst.IPs, a)
	}

	return inst
}

// ifaceNames returns a slice of interface names for logging.
func ifaceNames(ifaces []net.Interface) []string {
	names := make([]string, len(ifaces))
	for i, iface := range ifaces {
		names[i] = iface.Name
	}
	return names
}
