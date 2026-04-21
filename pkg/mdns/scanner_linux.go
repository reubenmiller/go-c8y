//go:build linux

package mdns

import (
	"context"
	"os"
	"strings"
)

// browse selects the best available mDNS backend on Linux.
//
// If port 5353 is held by another process (systemd-resolved, avahi), the
// pure-Go multicast backend cannot reliably receive mDNS responses because the
// kernel delivers SO_REUSEPORT multicast packets to only one socket. In that
// case the QU (Unicast-response) backend is used: it sends queries from an
// ephemeral port and mDNS responders unicast their answers back, fully
// bypassing port 5353.
//
// If running inside WSL2 a warning is logged: the WSL2 virtual network adapter
// does not forward multicast traffic to the host network.
func (s *Scanner) browse(ctx context.Context) (<-chan ServiceInstance, error) {
	if isWSL2() {
		s.opts.Logger.Printf(
			"mdns: WARNING: running inside WSL2. The WSL2 virtual network adapter " +
				"does not forward multicast traffic (224.0.0.251:5353), so mDNS " +
				"discovery may return no results. Consider running the native " +
				"Windows build of this tool instead.",
		)
	}

	if udp5353Busy() {
		s.opts.Logger.Printf("mdns: port 5353 is in use (systemd-resolved/avahi detected); " +
			"using QU unicast-response backend")
		return s.browseWithQU(ctx)
	}

	return s.browseWithPureGo(ctx)
}

// udp5353Busy reports whether something is listening on UDP port 5353 by
// reading /proc/net/udp. Port 5353 in big-endian hex is 14E9.
// No root is required to read /proc/net/udp.
func udp5353Busy() bool {
	data, err := os.ReadFile("/proc/net/udp")
	if err != nil {
		return false
	}
	// Each line has "local_address" in the form IPADDR:PORT (hex, big-endian).
	// Port 5353 = 0x14E9. We look for ":14E9 " (space terminates the port field).
	return strings.Contains(strings.ToUpper(string(data)), ":14E9 ")
}
