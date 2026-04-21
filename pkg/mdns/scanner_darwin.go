//go:build darwin

package mdns

import "context"

// browse selects the best available mDNS backend on macOS.
//
// The dns-sd CLI is preferred because it delegates to the mDNSResponder system
// daemon, requiring no special privileges. If dns-sd is not found in PATH, the
// pure-Go multicast backend is used as a fallback.
func (s *Scanner) browse(ctx context.Context) (<-chan ServiceInstance, error) {
	if isDNSSDAvailable() {
		s.opts.Logger.Printf("mdns: using dns-sd backend (delegates to mDNSResponder)")
		ch, err := s.browseWithDNSSD(ctx)
		if err == nil {
			return ch, nil
		}
		s.opts.Logger.Printf("mdns: dns-sd backend failed (%v), falling back to pure-Go backend", err)
	} else {
		s.opts.Logger.Printf("mdns: dns-sd not found in PATH, using pure-Go backend")
	}
	return s.browseWithPureGo(ctx)
}
