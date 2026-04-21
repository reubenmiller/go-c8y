// Package mdns provides cross-platform mDNS/Bonjour/Zeroconf service discovery
// without requiring root or administrator privileges.
//
// On macOS the system dns-sd CLI is preferred (it delegates to mDNSResponder
// which runs as a daemon). If dns-sd is unavailable a pure-Go multicast backend
// is used instead.
//
// On Linux and Windows a pure-Go multicast UDP implementation is used.
// WSL2 environments are detected and the user is warned that multicast traffic
// may not traverse the WSL2 virtual network adapter.
package mdns

import (
	"context"
	"log"
	"net"
	"regexp"
	"strings"
	"time"
)

// DefaultServiceType is the default mDNS service type for thin-edge.io devices.
const DefaultServiceType = "_tedge._tcp"

// DefaultDomain is the default mDNS search domain.
const DefaultDomain = "local"

// DefaultTimeout is how long Browse scans before closing the results channel.
const DefaultTimeout = 5 * time.Second

// ServiceInstance holds the details of a discovered mDNS service instance.
type ServiceInstance struct {
	// Name is the human-readable service instance name (e.g. "my-device").
	Name string
	// Host is the hostname advertised by the service (e.g. "my-device.local.").
	Host string
	// IPs are the resolved IP addresses for the host. May be empty when the
	// host could not be resolved.
	IPs []net.IP
	// Port is the TCP/UDP port the service listens on.
	Port int
	// TXT contains parsed key=value TXT record pairs.
	TXT map[string]string
	// RawTXT is the raw concatenated TXT record bytes (NUL-separated entries).
	RawTXT []byte
}

// Options controls the behaviour of the Scanner.
type Options struct {
	// ServiceType is the mDNS service type to look for (default: DefaultServiceType).
	ServiceType string
	// Domain is the mDNS search domain (default: DefaultDomain).
	Domain string
	// Timeout controls how long Browse scans before stopping (default: DefaultTimeout).
	// A context deadline shorter than Timeout takes precedence.
	Timeout time.Duration
	// Quick skips per-instance resolution (dns-sd -L on macOS) and returns only
	// the instance names discovered during browsing. Faster when host/port/TXT
	// details are not needed.
	Quick bool
	// Ifaces restricts scanning to specific network interfaces. When nil all
	// suitable interfaces are used (UP, multicast, non-loopback, with carrier).
	Ifaces []net.Interface
	// Logger receives warning and error messages. Defaults to log.Default().
	Logger *log.Logger
	// DebugLogger, when non-nil, receives verbose diagnostic messages such as
	// backend selection, per-interface socket details, and per-instance discovery
	// events. Nil by default (debug output suppressed).
	DebugLogger *log.Logger
	// Filter, when non-nil, restricts results to instances whose Name matches
	// the compiled regular expression.
	Filter *regexp.Regexp
	// MaxResults, when > 0, stops the scan as soon as that many matching
	// instances have been returned.
	MaxResults int
}

// Option is a functional option for NewScanner.
type Option func(*Options)

// WithServiceType overrides the mDNS service type (default: "_tedge._tcp").
func WithServiceType(s string) Option {
	return func(o *Options) { o.ServiceType = s }
}

// WithDomain overrides the mDNS search domain (default: "local").
func WithDomain(d string) Option {
	return func(o *Options) { o.Domain = d }
}

// WithTimeout sets how long Browse scans before closing the channel (default: 5s).
// A context deadline shorter than this value takes precedence.
func WithTimeout(t time.Duration) Option {
	return func(o *Options) { o.Timeout = t }
}

// WithLogger sets a custom logger for warning and error messages.
func WithLogger(l *log.Logger) Option {
	return func(o *Options) { o.Logger = l }
}

// WithDebugLogger sets a logger for verbose diagnostic messages (backend
// selection, per-interface details, per-instance discovery events). Pass nil
// to disable debug output (the default).
func WithDebugLogger(l *log.Logger) Option {
	return func(o *Options) { o.DebugLogger = l }
}

// WithQuick enables quick mode: only instance names are returned without
// per-instance resolution (host, port, TXT). Useful when the caller only needs
// to know which service instances are present on the network.
func WithQuick() Option {
	return func(o *Options) { o.Quick = true }
}

// WithIfaces restricts scanning to the specified network interfaces.
// By default all suitable interfaces are used automatically.
// Use this to target a specific interface (e.g. "wlan0") when automatic
// selection picks up unwanted interfaces.
func WithIfaces(ifaces []net.Interface) Option {
	return func(o *Options) { o.Ifaces = ifaces }
}

// WithFilter restricts results to instances whose Name matches the given
// regular expression. Non-matching instances are silently dropped.
func WithFilter(pattern *regexp.Regexp) Option {
	return func(o *Options) { o.Filter = pattern }
}

// WithMaxResults stops scanning as soon as n matching instances have been
// returned. Values <= 0 are ignored (no limit).
func WithMaxResults(n int) Option {
	return func(o *Options) { o.MaxResults = n }
}

// Scanner discovers mDNS service instances on the local network.
type Scanner struct {
	opts Options
}

// debugf writes to DebugLogger when it is non-nil. Call for verbose
// diagnostic messages that should not appear in normal (non-verbose) output.
func (s *Scanner) debugf(format string, args ...any) {
	if s.opts.DebugLogger != nil {
		s.opts.DebugLogger.Printf(format, args...)
	}
}

// NewScanner creates a Scanner with the given options.
//
//	sc := mdns.NewScanner(
//	    mdns.WithServiceType("_http._tcp"),
//	    mdns.WithTimeout(10*time.Second),
//	)
func NewScanner(opts ...Option) *Scanner {
	o := Options{
		ServiceType: DefaultServiceType,
		Domain:      DefaultDomain,
		Timeout:     DefaultTimeout,
		Logger:      log.Default(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &Scanner{opts: o}
}

// Browse scans for mDNS service instances and streams results on the returned
// channel. The channel is closed when the scan completes, times out, or ctx is
// cancelled. Results may arrive at any time while the channel is open.
//
// Results are filtered by Filter (if set) and capped at MaxResults (if > 0).
// When MaxResults is reached the scan is cancelled early.
//
// The platform-specific backend is selected automatically:
//   - macOS:   dns-sd CLI (falls back to pure-Go if unavailable)
//   - Linux:   pure-Go multicast UDP (warns in WSL2 environments)
//   - Windows: pure-Go multicast UDP (hints about Windows Firewall on empty results)
func (s *Scanner) Browse(ctx context.Context) (<-chan ServiceInstance, error) {
	ctx, cancel := context.WithCancel(ctx)

	raw, err := s.browse(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	out := make(chan ServiceInstance)
	go func() {
		defer close(out)
		defer cancel()
		count := 0
		for inst := range raw {
			if s.opts.Filter != nil && !s.opts.Filter.MatchString(inst.Name) {
				continue
			}
			// Normalize the hostname: strip any trailing dot that DNS backends
			// include in fully-qualified names (e.g. "my-device.local." → "my-device.local").
			inst.Host = strings.TrimSuffix(inst.Host, ".")
			select {
			case out <- inst:
			case <-ctx.Done():
				return
			}
			count++
			if s.opts.MaxResults > 0 && count >= s.opts.MaxResults {
				return
			}
		}
	}()

	return out, nil
}
