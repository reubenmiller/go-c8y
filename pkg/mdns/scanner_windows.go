//go:build windows

package mdns

import "context"

// browse uses the pure-Go mDNS multicast backend on Windows.
//
// The implementation joins the mDNS multicast group (224.0.0.251:5353) using
// standard UDP sockets and requires no administrator privileges.
//
// If no services are found, a Windows Firewall hint is logged: the default
// Windows Firewall policy may block inbound UDP on port 5353.
func (s *Scanner) browse(ctx context.Context) (<-chan ServiceInstance, error) {
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
