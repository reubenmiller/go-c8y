package mdns

import (
	"fmt"
	"strconv"
	"strings"
)

// parseBrowseLine parses a single output line from `dns-sd -B` and returns the
// service instance name. Only "Add" events are returned; "Rmv" lines and header
// lines return ("", false).
//
// Expected format (whitespace-separated columns after initialisation lines):
//
//	10:30:01.234  Add        3   7 local.               _tedge._tcp.         my-device
//	              [1]       [2] [3] [4]                  [5]                  [6...]
func parseBrowseLine(line string) (name string, ok bool) {
	fields := strings.Fields(line)
	// Need at least: timestamp A/R flags if domain serviceType instanceName
	if len(fields) < 7 {
		return "", false
	}
	if fields[1] != "Add" {
		return "", false
	}
	// Instance name is everything from field 6 onward (handles spaces in names).
	name = strings.Join(fields[6:], " ")
	return name, name != ""
}

// parseLookupReachLine extracts the host and port from a `dns-sd -L` output
// line of the form:
//
//	10:30:01.234  my-device._tedge._tcp.local. can be reached at my-device.local.:1883 (interface 7)
func parseLookupReachLine(line string) (host string, port int, err error) {
	const marker = "can be reached at "
	idx := strings.Index(line, marker)
	if idx < 0 {
		return "", 0, fmt.Errorf("marker %q not found in line", marker)
	}
	rest := line[idx+len(marker):]
	// Take only the first token: "my-device.local.:1883"
	addr := strings.Fields(rest)[0]
	// Split on the last colon to handle IPv6 literals and dotted hostnames.
	lastColon := strings.LastIndex(addr, ":")
	if lastColon < 0 {
		return "", 0, fmt.Errorf("no port separator found in %q", addr)
	}
	host = addr[:lastColon]
	port, err = strconv.Atoi(addr[lastColon+1:])
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in %q: %w", addr, err)
	}
	return host, port, nil
}

// parseTXTSlice converts a slice of "key=value" or "key" strings into a map.
// Keys without "=" are stored with an empty string value.
func parseTXTSlice(records []string) map[string]string {
	m := make(map[string]string, len(records))
	for _, r := range records {
		if k, v, ok := strings.Cut(r, "="); ok {
			m[k] = v
		} else if r != "" {
			m[r] = ""
		}
	}
	return m
}

// parseTXTLine splits a space-separated TXT record line into a map.
// Suitable for parsing the indented TXT line produced by `dns-sd -L`.
func parseTXTLine(line string) map[string]string {
	return parseTXTSlice(strings.Fields(line))
}
