//go:build windows && 386

package mdns

import "context"

// browse uses the multicast+QU fallback path on 32-bit Windows.
// The Windows DNS Service API (DnsServiceBrowse) is only supported on
// 64-bit builds (amd64/arm64) where the struct layouts are verified.
func (s *Scanner) browse(ctx context.Context) (<-chan ServiceInstance, error) {
	s.opts.Logger.Printf("mdns: 32-bit Windows — using multicast+QU backend (WinAPI backend requires 64-bit build)")
	return s.browseMulticastAndQU(ctx)
}
