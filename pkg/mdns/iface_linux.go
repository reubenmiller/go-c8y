//go:build linux

package mdns

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// hasCarrier reports whether a network interface has a physical carrier signal
// by reading /sys/class/net/<name>/carrier. Returns true if the file cannot be
// read (e.g. on non-Linux or virtual interfaces that don't expose the file).
func hasCarrier(name string) bool {
	data, err := os.ReadFile(fmt.Sprintf("/sys/class/net/%s/carrier", name))
	if err != nil {
		// Cannot determine — assume it's usable.
		return true
	}
	return strings.TrimSpace(string(data)) == "1"
}

// activeMulticastIfaces returns network interfaces that are UP, support
// multicast, are not loopback, and have a physical carrier. On Linux the
// carrier check prevents interfaces like docker0 (bridge with no containers)
// from being selected, which can cause the zeroconf multicast socket to fail.
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
		if !hasCarrier(iface.Name) {
			continue
		}
		out = append(out, iface)
	}
	return out, nil
}
