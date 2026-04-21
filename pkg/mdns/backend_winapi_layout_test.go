//go:build windows && (amd64 || arm64)

package mdns

import (
	"testing"
	"unsafe"
)

// TestWinAPIStructLayout verifies that the Go mirror structs have the exact
// field offsets expected by the Windows SDK. A mismatch would cause silent
// data corruption when calling DnsServiceBrowse / DnsServiceResolve.
func TestWinAPIStructLayout(t *testing.T) {
	t.Run("dnsServiceBrowseRequest", func(t *testing.T) {
		var r dnsServiceBrowseRequest
		check := func(name string, got, want uintptr) {
			t.Helper()
			if got != want {
				t.Errorf("dnsServiceBrowseRequest.%s: offset %d, want %d", name, got, want)
			}
		}
		check("Version", unsafe.Offsetof(r.Version), 0)
		check("InterfaceIndex", unsafe.Offsetof(r.InterfaceIndex), 4)
		check("QueryName", unsafe.Offsetof(r.QueryName), 8)
		check("BrowseCallback", unsafe.Offsetof(r.BrowseCallback), 16)
		check("QueryContext", unsafe.Offsetof(r.QueryContext), 24)
		if sz := unsafe.Sizeof(r); sz != 32 {
			t.Errorf("dnsServiceBrowseRequest size: %d, want 32", sz)
		}
	})

	t.Run("dnsServiceResolveRequest", func(t *testing.T) {
		var r dnsServiceResolveRequest
		check := func(name string, got, want uintptr) {
			t.Helper()
			if got != want {
				t.Errorf("dnsServiceResolveRequest.%s: offset %d, want %d", name, got, want)
			}
		}
		check("Version", unsafe.Offsetof(r.Version), 0)
		check("InterfaceIndex", unsafe.Offsetof(r.InterfaceIndex), 4)
		check("QueryName", unsafe.Offsetof(r.QueryName), 8)
		check("ResolveCompletionCallback", unsafe.Offsetof(r.ResolveCompletionCallback), 16)
		check("QueryContext", unsafe.Offsetof(r.QueryContext), 24)
		if sz := unsafe.Sizeof(r); sz != 32 {
			t.Errorf("dnsServiceResolveRequest size: %d, want 32", sz)
		}
	})

	t.Run("dnsServiceInstance", func(t *testing.T) {
		var r dnsServiceInstance
		check := func(name string, got, want uintptr) {
			t.Helper()
			if got != want {
				t.Errorf("dnsServiceInstance.%s: offset %d, want %d", name, got, want)
			}
		}
		check("InstanceName", unsafe.Offsetof(r.InstanceName), 0)
		check("HostName", unsafe.Offsetof(r.HostName), 8)
		check("IPv4Address", unsafe.Offsetof(r.IPv4Address), 16)
		check("IPv6Address", unsafe.Offsetof(r.IPv6Address), 24)
		check("Port", unsafe.Offsetof(r.Port), 32)
		check("Priority", unsafe.Offsetof(r.Priority), 34)
		check("Weight", unsafe.Offsetof(r.Weight), 36)
		check("PropertyCount", unsafe.Offsetof(r.PropertyCount), 40)
		check("Keys", unsafe.Offsetof(r.Keys), 48)
		check("Values", unsafe.Offsetof(r.Values), 56)
		check("InterfaceIndex", unsafe.Offsetof(r.InterfaceIndex), 64)
		if sz := unsafe.Sizeof(r); sz != 72 {
			t.Errorf("dnsServiceInstance size: %d, want 72", sz)
		}
	})

	t.Run("dnsRecord", func(t *testing.T) {
		var r dnsRecord
		check := func(name string, got, want uintptr) {
			t.Helper()
			if got != want {
				t.Errorf("dnsRecord.%s: offset %d, want %d", name, got, want)
			}
		}
		check("Next", unsafe.Offsetof(r.Next), 0)
		check("Name", unsafe.Offsetof(r.Name), 8)
		check("Type", unsafe.Offsetof(r.Type), 16)
		check("DataLength", unsafe.Offsetof(r.DataLength), 18)
		check("Flags", unsafe.Offsetof(r.Flags), 20)
		check("TTL", unsafe.Offsetof(r.TTL), 24)
		check("Reserved", unsafe.Offsetof(r.Reserved), 28)
		check("DataPtrName", unsafe.Offsetof(r.DataPtrName), 32)
	})

	t.Run("dnsServiceCancel", func(t *testing.T) {
		var r dnsServiceCancel
		if sz := unsafe.Sizeof(r); sz != 8 {
			t.Errorf("dnsServiceCancel size: %d, want 8 (one pointer)", sz)
		}
	})
}
