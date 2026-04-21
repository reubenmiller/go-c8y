package mdns

import (
	"strings"
	"testing"
)

// --- parseBrowseLine ---

func TestParseBrowseLine_Add(t *testing.T) {
	line := "10:30:01.234  Add        3   7 local.               _tedge._tcp.         my-device"
	name, ok := parseBrowseLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if name != "my-device" {
		t.Errorf("name=%q, want %q", name, "my-device")
	}
}

func TestParseBrowseLine_AddWithSpacesInName(t *testing.T) {
	line := "10:30:01.234  Add        3   7 local.               _tedge._tcp.         device with spaces"
	name, ok := parseBrowseLine(line)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if name != "device with spaces" {
		t.Errorf("name=%q, want %q", name, "device with spaces")
	}
}

func TestParseBrowseLine_Remove(t *testing.T) {
	line := "10:30:01.234  Rmv        3   7 local.               _tedge._tcp.         my-device"
	_, ok := parseBrowseLine(line)
	if ok {
		t.Error("expected ok=false for Rmv line")
	}
}

func TestParseBrowseLine_Header(t *testing.T) {
	line := "Timestamp     A/R    Flags  if Domain               Service Type         Instance Name"
	_, ok := parseBrowseLine(line)
	if ok {
		t.Error("expected ok=false for header line")
	}
}

func TestParseBrowseLine_Empty(t *testing.T) {
	_, ok := parseBrowseLine("")
	if ok {
		t.Error("expected ok=false for empty line")
	}
}

func TestParseBrowseLine_Starting(t *testing.T) {
	line := "10:30:00.000  ...STARTING..."
	_, ok := parseBrowseLine(line)
	if ok {
		t.Error("expected ok=false for STARTING line")
	}
}

// --- parseLookupReachLine ---

func TestParseLookupReachLine(t *testing.T) {
	line := "10:30:01.234  my-device._tedge._tcp.local. can be reached at my-device.local.:1883 (interface 7)"
	host, port, err := parseLookupReachLine(line)
	if err != nil {
		t.Fatal(err)
	}
	if host != "my-device.local." {
		t.Errorf("host=%q, want %q", host, "my-device.local.")
	}
	if port != 1883 {
		t.Errorf("port=%d, want 1883", port)
	}
}

func TestParseLookupReachLine_NoMarker(t *testing.T) {
	_, _, err := parseLookupReachLine("some unrelated line")
	if err == nil {
		t.Error("expected error for line without marker")
	}
}

func TestParseLookupReachLine_InvalidPort(t *testing.T) {
	line := "10:30:01.234  x._tedge._tcp.local. can be reached at x.local.:notaport (interface 4)"
	_, _, err := parseLookupReachLine(line)
	if err == nil {
		t.Error("expected error for invalid port")
	}
}

// --- parseTXTSlice ---

func TestParseTXTSlice_KeyValue(t *testing.T) {
	m := parseTXTSlice([]string{"mapper=c8y-mapper", "type=thin-edge", "flag"})
	if m["mapper"] != "c8y-mapper" {
		t.Errorf("mapper=%q", m["mapper"])
	}
	if m["type"] != "thin-edge" {
		t.Errorf("type=%q", m["type"])
	}
	if _, ok := m["flag"]; !ok {
		t.Error("expected 'flag' key to be present")
	}
}

func TestParseTXTSlice_Empty(t *testing.T) {
	m := parseTXTSlice(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestParseTXTSlice_ValueWithEquals(t *testing.T) {
	// Values may themselves contain "=".
	m := parseTXTSlice([]string{"url=http://host:8080/path?a=b"})
	if m["url"] != "http://host:8080/path?a=b" {
		t.Errorf("url=%q", m["url"])
	}
}

// --- WSL2 detection logic (no OS calls, tests the string matching) ---

func TestWSL2Detection_Match(t *testing.T) {
	kernelStrings := []string{
		"Linux version 5.15.90.1-microsoft-standard-WSL2 (oe-user@oe-host)",
		"Linux version 4.4.0-19041-Microsoft (Microsoft@Microsoft.com)",
	}
	for _, s := range kernelStrings {
		lower := strings.ToLower(s)
		if !(strings.Contains(lower, "microsoft") && strings.Contains(lower, "wsl")) {
			// WSL1 may not contain "wsl" explicitly; only assert WSL2 pattern.
			if !strings.Contains(lower, "microsoft") {
				t.Errorf("expected 'microsoft' in %q", s)
			}
		}
	}
}

func TestWSL2Detection_NoMatch(t *testing.T) {
	kernelStrings := []string{
		"Linux version 6.1.0-18-amd64 (debian-kernel@lists.debian.org)",
		"Linux version 5.15.0-91-generic (buildd@lcy02-amd64-059)",
	}
	for _, s := range kernelStrings {
		lower := strings.ToLower(s)
		if strings.Contains(lower, "microsoft") {
			t.Errorf("expected no 'microsoft' in %q", s)
		}
	}
}

// --- Scanner construction ---

func TestNewScanner_Defaults(t *testing.T) {
	sc := NewScanner()
	if sc.opts.ServiceType != DefaultServiceType {
		t.Errorf("ServiceType=%q, want %q", sc.opts.ServiceType, DefaultServiceType)
	}
	if sc.opts.Domain != DefaultDomain {
		t.Errorf("Domain=%q, want %q", sc.opts.Domain, DefaultDomain)
	}
	if sc.opts.Timeout != DefaultTimeout {
		t.Errorf("Timeout=%v, want %v", sc.opts.Timeout, DefaultTimeout)
	}
}

func TestNewScanner_Options(t *testing.T) {
	sc := NewScanner(
		WithServiceType("_http._tcp"),
		WithDomain("local"),
	)
	if sc.opts.ServiceType != "_http._tcp" {
		t.Errorf("ServiceType=%q", sc.opts.ServiceType)
	}
}
