// mdns-scanner is a command-line tool for discovering mDNS/Bonjour/Zeroconf
// service instances on the local network.
//
// Usage:
//
//	mdns-scanner [flags]
//
// Examples:
//
//	# Scan for the default thin-edge.io service type
//	mdns-scanner
//
//	# Scan for HTTP services with a 10-second timeout and JSON output
//	mdns-scanner --service _http._tcp --timeout 10s --output json
//
//	# Scan and print only hostnames and ports
//	mdns-scanner --service _mqtt._tcp --output text
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/mdns"
	"github.com/spf13/cobra"
)

type outputFormat string

const (
	formatText outputFormat = "text"
	formatJSON outputFormat = "json"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		serviceType string
		domain      string
		timeout     time.Duration
		output      string
		verbose     bool
		quick       bool
		ifaces      []string
		pattern     string
		maxResults  int
	)

	cmd := &cobra.Command{
		Use:   "mdns-scanner",
		Short: "Discover mDNS/Bonjour/Zeroconf services on the local network",
		Long: `mdns-scanner discovers mDNS (Bonjour/Zeroconf) service instances on the
local network without requiring root or administrator privileges.

Platform notes:
  macOS   — uses dns-sd (delegates to mDNSResponder daemon); falls back to
            pure-Go multicast if dns-sd is unavailable.
  Linux   — uses pure-Go multicast UDP.
  WSL2    — multicast may not traverse the WSL2 vNIC; try the native Windows binary.
  Windows — uses pure-Go multicast UDP. The Windows Firewall may block UDP 5353;
            a remediation hint is printed when no services are found.`,
		Example: `  # Scan for the default thin-edge.io service type
  mdns-scanner

  # Quick scan — return instance names only, no per-host resolution
  mdns-scanner --quick

  # Scan for HTTP services with a 10-second timeout
  mdns-scanner --service _http._tcp --timeout 10s

  # Machine-readable JSON output
  mdns-scanner --output json

  # Verbose diagnostic logging
  mdns-scanner --verbose`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(serviceType, domain, timeout, outputFormat(output), verbose, quick, ifaces, pattern, maxResults)
		},
	}

	cmd.Flags().StringVarP(&serviceType, "service", "s", mdns.DefaultServiceType,
		"mDNS service type to scan for (e.g. _http._tcp, _mqtt._tcp)")
	cmd.Flags().StringVar(&domain, "domain", mdns.DefaultDomain,
		"mDNS search domain")
	cmd.Flags().DurationVarP(&timeout, "timeout", "t", mdns.DefaultTimeout,
		"how long to scan before stopping (e.g. 5s, 10s, 1m)")
	cmd.Flags().StringVarP(&output, "output", "o", string(formatText),
		`output format: "text" or "json"`)
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"enable verbose diagnostic logging")
	cmd.Flags().BoolVarP(&quick, "quick", "q", false,
		"quick mode: return instance names only without per-host resolution (faster)")
	cmd.Flags().StringSliceVarP(&ifaces, "iface", "i", nil,
		"network interface(s) to scan on (e.g. wlan0,eth0); default: all suitable interfaces")
	cmd.Flags().StringVarP(&pattern, "pattern", "p", "",
		"regular expression to filter results by instance name (e.g. \"^rpi5\"")
	cmd.Flags().IntVarP(&maxResults, "max", "n", 0,
		"stop after finding this many results (0 = no limit)")

	return cmd
}

func run(serviceType, domain string, timeout time.Duration, output outputFormat, verbose bool, quick bool, ifaceNames []string, pattern string, maxResults int) error {
	logger := log.New(os.Stderr, "[mdns] ", log.LstdFlags)
	if !verbose {
		// Suppress all diagnostic messages in non-verbose mode except warnings.
		// We do this by directing the logger to a discard writer and emitting
		// warnings ourselves through scanner callbacks.
		logger = log.New(os.Stderr, "[mdns] WARNING: ", 0)
	}

	opts := []mdns.Option{
		mdns.WithServiceType(serviceType),
		mdns.WithDomain(domain),
		mdns.WithTimeout(timeout),
		mdns.WithLogger(logger),
	}
	if quick {
		opts = append(opts, mdns.WithQuick())
	}
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid --filter pattern: %w", err)
		}
		opts = append(opts, mdns.WithFilter(re))
	}
	if maxResults > 0 {
		opts = append(opts, mdns.WithMaxResults(maxResults))
	}
	if len(ifaceNames) > 0 {
		var netIfaces []net.Interface
		for _, name := range ifaceNames {
			iface, err := net.InterfaceByName(name)
			if err != nil {
				return fmt.Errorf("interface %q: %w", name, err)
			}
			netIfaces = append(netIfaces, *iface)
		}
		opts = append(opts, mdns.WithIfaces(netIfaces))
	}
	sc := mdns.NewScanner(opts...)

	// Respect Ctrl-C so the scan can be interrupted cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if verbose {
		fmt.Fprintf(os.Stderr, "Scanning for %q on %q (timeout: %s)...\n",
			serviceType, domain, timeout)
	}

	start := time.Now()
	ch, err := sc.Browse(ctx)
	if err != nil {
		return fmt.Errorf("browse: %w", err)
	}

	count := 0
	for inst := range ch {
		count++
		switch output {
		case formatJSON:
			printJSON(inst)
		default:
			printText(inst)
		}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "\nFound %d instance(s) in %s.\n", count, time.Since(start).Round(time.Millisecond))
	} else if count == 0 {
		fmt.Fprintf(os.Stderr, "No %q services found on %q within %s.\n", serviceType, domain, timeout)
	}

	return nil
}

// printText writes a human-readable block for one service instance.
func printText(inst mdns.ServiceInstance) {
	fmt.Printf("Name: %s\n", inst.Name)
	if inst.Host != "" {
		fmt.Printf("Host: %s\n", inst.Host)
		if inst.Port != 0 {
			fmt.Printf("Port: %d\n", inst.Port)
		}
	}
	for _, ip := range inst.IPs {
		fmt.Printf("IP:   %s\n", ip)
	}
	if len(inst.TXT) > 0 {
		fmt.Printf("TXT:\n")
		for k, v := range inst.TXT {
			if v == "" {
				fmt.Printf("  %s\n", k)
			} else {
				fmt.Printf("  %s=%s\n", k, v)
			}
		}
	}
	fmt.Println()
}

// jsonInstance is the JSON representation of a ServiceInstance.
type jsonInstance struct {
	Name string            `json:"name"`
	Host string            `json:"host"`
	IPs  []string          `json:"ips"`
	Port int               `json:"port"`
	TXT  map[string]string `json:"txt,omitempty"`
}

// printJSON writes a single NDJSON line for one service instance.
func printJSON(inst mdns.ServiceInstance) {
	out := jsonInstance{
		Name: inst.Name,
		Host: inst.Host,
		Port: inst.Port,
		TXT:  inst.TXT,
	}
	for _, ip := range inst.IPs {
		out.IPs = append(out.IPs, ip.String())
	}
	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}
