//go:build !linux

package mdns

import "net"

// activeMulticastIfaces returns network interfaces that are UP, support
// multicast, and are not loopback. The ifaces parameter, if non-nil, is
// filtered; otherwise all system interfaces are enumerated.
func activeMulticastIfaces(ifaces []net.Interface) ([]net.Interface, error) {
	all := ifaces
	var err error
	if all == nil {
		all, err = net.Interfaces()
		if err != nil {
			return nil, err
		}
	}
	var out []net.Interface
	for _, iface := range all {
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			continue
		}
		out = append(out, iface)
	}
	return out, nil
}
