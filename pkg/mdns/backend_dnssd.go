//go:build darwin

package mdns

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// lookupTimeout is the per-instance resolution deadline for dns-sd -L.
const lookupTimeout = 3 * time.Second

// isDNSSDAvailable reports whether the dns-sd command-line tool is present in PATH.
// dns-sd is installed by default on macOS and communicates with the mDNSResponder
// daemon, which means no root privilege is required.
func isDNSSDAvailable() bool {
	_, err := exec.LookPath("dns-sd")
	return err == nil
}

// browseWithDNSSD uses the macOS dns-sd CLI to discover services.
//
// `dns-sd -B` runs for up to the configured timeout. In full mode (default)
// each discovered instance is resolved concurrently with `dns-sd -L` the
// moment its name appears in the browse output. In quick mode only the bare
// instance names from `dns-sd -B` are returned without extra lookups.
//
// All child processes are terminated when ctx is cancelled or the timeout elapses.
func (s *Scanner) browseWithDNSSD(ctx context.Context) (<-chan ServiceInstance, error) {
	browseCtx, browseCancel := context.WithTimeout(ctx, s.opts.Timeout)

	cmd := exec.CommandContext(browseCtx, "dns-sd", "-B", s.opts.ServiceType, s.opts.Domain)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		browseCancel()
		return nil, fmt.Errorf("dns-sd browse pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		browseCancel()
		return nil, fmt.Errorf("dns-sd browse start: %w", err)
	}

	out := make(chan ServiceInstance)
	var wg sync.WaitGroup
	seen := make(map[string]bool)
	var seenMu sync.Mutex

	go func() {
		// Correct shutdown order:
		//   1. drain the scanner (stdout pipe) — must happen before cmd.Wait
		//   2. reap the child process
		//   3. wait for all in-flight lookup goroutines to finish
		//   4. close the output channel
		// Using explicit sequencing here instead of defer (which is LIFO and
		// would close the channel before lookups finish).
		defer browseCancel()

		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			name, ok := parseBrowseLine(sc.Text())
			if !ok {
				continue
			}

			seenMu.Lock()
			duplicate := seen[name]
			seen[name] = true
			seenMu.Unlock()

			if duplicate {
				continue
			}

			if s.opts.Quick {
				// Quick mode: resolve hostname and IPs via the system resolver
				// (mDNSResponder on macOS) without spawning a dns-sd -L subprocess.
				wg.Add(1)
				go func(instanceName string) {
					defer wg.Done()
					inst := quickResolve(ctx, instanceName, s.opts.Domain)
					select {
					case out <- inst:
					case <-ctx.Done():
					}
				}(name)
				continue
			}

			wg.Add(1)
			go func(instanceName string) {
				defer wg.Done()
				inst, err := dnssdLookup(ctx, instanceName, s.opts.ServiceType, s.opts.Domain)
				if err != nil {
					s.opts.Logger.Printf("mdns: dns-sd lookup %q: %v", instanceName, err)
					return
				}
				select {
				case out <- inst:
				case <-ctx.Done():
				}
			}(name)
		}

		// Drain complete — reap the process, then wait for lookups, then close.
		_ = cmd.Wait()
		wg.Wait()
		close(out)
	}()

	return out, nil
}

// quickResolve returns a ServiceInstance with host and IPs populated via the
// system DNS resolver (which on macOS goes through mDNSResponder, so .local
// names are resolved without root and without spawning a dns-sd -L process).
func quickResolve(ctx context.Context, name, domain string) ServiceInstance {
	host := name + "." + domain + "."
	inst := ServiceInstance{Name: name, Host: host}
	rctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	// net.DefaultResolver.LookupHost respects the context deadline.
	addrs, err := net.DefaultResolver.LookupHost(rctx, host)
	if err == nil {
		for _, a := range addrs {
			if ip := net.ParseIP(a); ip != nil {
				inst.IPs = append(inst.IPs, ip)
			}
		}
	}
	return inst
}

// dnssdLookup resolves a single service instance using `dns-sd -L`.
// It wraps ctx with a 3 s deadline so a single unresponsive lookup cannot
// block the whole scan; the parent ctx still wins if it expires sooner.
func dnssdLookup(ctx context.Context, name, serviceType, domain string) (ServiceInstance, error) {
	lctx, cancel := context.WithTimeout(ctx, lookupTimeout)
	defer cancel()
	cmd := exec.CommandContext(lctx, "dns-sd", "-L", name, serviceType, domain)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return ServiceInstance{}, err
	}
	if err := cmd.Start(); err != nil {
		return ServiceInstance{}, err
	}

	inst := ServiceInstance{Name: name}

	sc := bufio.NewScanner(stdout)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.Contains(line, "can be reached at"):
			host, port, err := parseLookupReachLine(line)
			if err == nil {
				inst.Host = host
				inst.Port = port
			}
		case strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t"):
			// Indented line contains space-separated TXT records.
			trimmed := strings.TrimSpace(line)
			if trimmed != "" {
				inst.TXT = parseTXTLine(trimmed)
				inst.RawTXT = []byte(trimmed)
			}
		}
	}

	_ = cmd.Wait()

	if inst.Host == "" {
		return ServiceInstance{}, fmt.Errorf("no address resolved for %q", name)
	}

	// Resolve hostname to IPs via the system resolver, which on macOS handles
	// .local mDNS names through mDNSResponder.
	addrs, err := net.LookupHost(inst.Host)
	if err == nil {
		for _, a := range addrs {
			if ip := net.ParseIP(a); ip != nil {
				inst.IPs = append(inst.IPs, ip)
			}
		}
	}

	return inst, nil
}
