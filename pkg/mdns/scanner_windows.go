//go:build windows

package mdns

import (
	"context"
	"net"
)

// browse selects the best available mDNS backend on Windows.
//
// Windows 10 (build 1703+) and Windows 11 include an mDNS responder in the
// DNS Client service (dnscache) that binds exclusively to UDP port 5353. When
// that service is running, the pure-Go multicast backend receives no packets
// because the Windows DNS Client holds the socket without SO_REUSEPORT sharing.
//
// In that case the QU (Unicast-response) backend is used: it sends queries
// from an ephemeral port so that mDNS responders unicast their answers back,
// bypassing port 5353 entirely. No administrator privileges are required.
//
// If port 5353 is free (DNS Client disabled or not present), the pure-Go
// multicast backend is used and a Windows Firewall hint is logged if no
// results are found.
func (s *Scanner) browse(ctx context.Context) (<-chan ServiceInstance, error) {
	if udp5353BusyWindows() {
		s.opts.Logger.Printf("mdns: port 5353 is in use (Windows DNS Client detected); " +
			"using QU unicast-response backend")
		return s.browseWithQU(ctx)
	}

	upstream, err := s.browseWithPureGo(ctx)
	if err != nil {
		return nil, err
	}

	out := make(chan ServiceInstance)
	go func() {
		defer close(out)
		count := 0
		for inst := range upstream {
			count++
			out <- inst
		}
		if count == 0 {
			s.opts.Logger.Printf(
				"mdns: no services found. If you expected results, Windows Firewall " +
					"may be blocking inbound UDP on port 5353 (multicast DNS). " +
					"To allow mDNS, run the following command as Administrator:\n" +
					`  netsh advfirewall firewall add rule name="mDNS" dir=in ` +
					`action=allow protocol=UDP localport=5353`,
			)
		}
	}()
	return out, nil
}

// udp5353BusyWindows reports whether UDP port 5353 is already held by another
// process (typically the Windows DNS Client service). Unlike Linux, Windows
// does not use SO_REUSEPORT for this socket, so a bind attempt will fail
// immediately with WSAEADDRINUSE if the port is taken.
func udp5353BusyWindows() bool {
	c, err := net.ListenUDP("udp4", &net.UDPAddr{Port: 5353})
	if err != nil {
		return true // port is taken
	}
	c.Close()
	return false
}
