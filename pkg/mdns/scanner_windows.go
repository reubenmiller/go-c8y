//go:build windows && (amd64 || arm64)

package mdns

import "context"

// browse selects the best available mDNS backend on Windows.
//
// Priority:
//  1. Windows DNS Service API (DnsServiceBrowse / DnsServiceResolve in dnsapi.dll,
//     Windows 10 build 1703+): communicates with the Windows DNS Client service
//     via ALPC/IPC — no inbound socket opened in this process, no Windows Firewall
//     prompt, no administrator privileges required.
//  2. Pure-Go multicast + QU unicast-response in parallel (fallback for older
//     Windows or when the WinAPI procs are unavailable).
func (s *Scanner) browse(ctx context.Context) (<-chan ServiceInstance, error) {
	if winAPIAvailable() {
		s.debugf("mdns: using Windows DNS Service API (DnsServiceBrowse) — no firewall rules required")
		ch, err := s.browseWithWinAPI(ctx)
		if err == nil {
			return ch, nil
		}
		s.opts.Logger.Printf("mdns: Windows DNS Service API failed (%v); falling back to multicast+QU", err)
	} else {
		s.opts.Logger.Printf("mdns: Windows DNS Service API not available (requires Windows 10 build 1703+); using multicast+QU")
	}

	return s.browseMulticastAndQU(ctx)
}

// browseMulticastAndQU is defined in scanner_windows_fallback.go so it is
// shared between all Windows architectures (amd64, arm64, and 386).
