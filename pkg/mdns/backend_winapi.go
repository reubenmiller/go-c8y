//go:build windows && (amd64 || arm64)

package mdns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows DNS Service API procs loaded lazily from dnsapi.dll.
// DnsServiceBrowse / DnsServiceResolve are available on Windows 10 build 1703+.
// If the proc cannot be found (older Windows) the load will fail gracefully.
var (
	modDnsapi = windows.NewLazySystemDLL("dnsapi.dll")

	procDnsServiceBrowse        = modDnsapi.NewProc("DnsServiceBrowse")
	procDnsServiceBrowseCancel  = modDnsapi.NewProc("DnsServiceBrowseCancel")
	procDnsServiceResolve       = modDnsapi.NewProc("DnsServiceResolve")
	procDnsServiceResolveCancel = modDnsapi.NewProc("DnsServiceResolveCancel")
	procDnsServiceFreeInstance  = modDnsapi.NewProc("DnsServiceFreeInstance")
	procDnsRecordListFree       = modDnsapi.NewProc("DnsRecordListFree")
)

const (
	dnsQueryRequestVersion1 = 1
	dnsRequestPending       = 9506 // DNS_REQUEST_PENDING — returned by async APIs on success
	dnsRecordTypePTR        = 0x000C
	dnsFreeRecordList       = 1 // DnsFreeRecordList flag for DnsRecordListFree
)

// dnsServiceCancel mirrors DNS_SERVICE_CANCEL.
// The struct holds an opaque PVOID handle written by DnsServiceBrowse /
// DnsServiceResolve. It MUST remain pinned in memory (not stack-local in a
// function that returns before cancel) until DnsServiceBrowseCancel /
// DnsServiceResolveCancel returns.
type dnsServiceCancel struct {
	reserved uintptr
}

// dnsServiceBrowseRequest mirrors DNS_SERVICE_BROWSE_REQUEST (Version 1).
//
// amd64 layout (verified by backend_winapi_layout_test.go):
//
//	offset  0: Version        uint32  (4 bytes)
//	offset  4: InterfaceIndex uint32  (4 bytes)
//	offset  8: QueryName      *uint16 (8 bytes)
//	offset 16: BrowseCallback uintptr (8 bytes)  ← union field, Version-1 callback
//	offset 24: QueryContext   uintptr (8 bytes)
//	total: 32 bytes
type dnsServiceBrowseRequest struct {
	Version        uint32
	InterfaceIndex uint32
	QueryName      *uint16
	BrowseCallback uintptr
	QueryContext   uintptr
}

// dnsServiceResolveRequest mirrors DNS_SERVICE_RESOLVE_REQUEST (Version 1).
// Same field layout as dnsServiceBrowseRequest; only the callback semantics differ.
type dnsServiceResolveRequest struct {
	Version                   uint32
	InterfaceIndex            uint32
	QueryName                 *uint16
	ResolveCompletionCallback uintptr
	QueryContext              uintptr
}

// dnsServiceInstance mirrors DNS_SERVICE_INSTANCE.
//
// amd64 layout:
//
//	offset  0: InstanceName  *uint16     (8)
//	offset  8: HostName      *uint16     (8)
//	offset 16: IPv4Address   *uint32     (8)  — pointer to 4-byte IPv4, may be nil
//	offset 24: IPv6Address   *[16]byte   (8)  — pointer to IP6_ADDRESS, may be nil
//	offset 32: Port          uint16      (2)
//	offset 34: Priority      uint16      (2)
//	offset 36: Weight        uint16      (2)
//	offset 38: _pad0         [2]byte     (2)
//	offset 40: PropertyCount uint32      (4)
//	offset 44: _pad1         [4]byte     (4)
//	offset 48: Keys          **uint16    (8)
//	offset 56: Values        **uint16    (8)
//	offset 64: InterfaceIndex uint32     (4)
//	offset 68: _pad2         [4]byte     (4)
//	total: 72 bytes
type dnsServiceInstance struct {
	InstanceName   *uint16
	HostName       *uint16
	IPv4Address    *uint32
	IPv6Address    *[16]byte
	Port           uint16
	Priority       uint16
	Weight         uint16
	_pad0          [2]byte
	PropertyCount  uint32
	_pad1          [4]byte
	Keys           **uint16
	Values         **uint16
	InterfaceIndex uint32
	_pad2          [4]byte
}

// dnsRecord is the minimal head of a DNS_RECORD linked list.
// We only need Next, Type, and the PTR data field.
//
// amd64 layout (relevant fields):
//
//	offset  0: Next        *dnsRecord (8)
//	offset  8: Name        *uint16    (8)
//	offset 16: Type        uint16     (2)
//	offset 18: DataLength  uint16     (2)
//	offset 20: Flags       uint32     (4)
//	offset 24: TTL         uint32     (4)
//	offset 28: Reserved    uint32     (4)
//	offset 32: DataPtrName *uint16    (8)  ← first field of Data union (PTR record)
type dnsRecord struct {
	Next        *dnsRecord
	Name        *uint16
	Type        uint16
	DataLength  uint16
	Flags       uint32
	TTL         uint32
	Reserved    uint32
	DataPtrName *uint16 // valid when Type == dnsRecordTypePTR
}

// winAPIAvailable returns true if the DnsServiceBrowse proc can be loaded.
// This fails on Windows versions prior to build 1703.
func winAPIAvailable() bool {
	return procDnsServiceBrowse.Find() == nil
}

// browseWithWinAPI discovers mDNS services using the Windows DNS Service API
// (DnsServiceBrowse → DnsServiceResolve). Both APIs communicate with the
// Windows DNS Client service via ALPC, so no inbound socket is opened in this
// process — Windows Firewall never prompts and no admin privileges are required.
//
// Requires Windows 10 build 1703 (April Creators Update) or later.
func (s *Scanner) browseWithWinAPI(ctx context.Context) (<-chan ServiceInstance, error) {
	if err := procDnsServiceBrowse.Find(); err != nil {
		return nil, fmt.Errorf("mdns WinAPI: DnsServiceBrowse not available: %w", err)
	}

	// Query name: "_tedge._tcp.local" — no trailing dot for these APIs.
	queryName := s.opts.ServiceType + "." + s.opts.Domain
	queryNameW, err := windows.UTF16PtrFromString(queryName)
	if err != nil {
		return nil, fmt.Errorf("mdns WinAPI: encode query name: %w", err)
	}

	// Internal channel for PTR FQDNs received in the browse callback.
	ptrCh := make(chan string, 32)

	// The browse callback fires on a Windows thread-pool thread.
	// It must not block; only channel sends are permitted.
	browseCallback := windows.NewCallback(func(status uint32, _ uintptr, records uintptr) uintptr {
		if status != 0 || records == 0 {
			return 0
		}
		for rec := (*dnsRecord)(unsafe.Pointer(records)); rec != nil; rec = rec.Next {
			if rec.Type == dnsRecordTypePTR && rec.DataPtrName != nil {
				fqdn := windows.UTF16PtrToString(rec.DataPtrName)
				select {
				case ptrCh <- fqdn:
				default: // drop if buffer full; next browse callback will retry
				}
			}
		}
		procDnsRecordListFree.Call(records, dnsFreeRecordList)
		return 0
	})

	// browseState keeps the cancel struct alive on the heap for the lifetime
	// of the browse operation (DNS_SERVICE_CANCEL must not be stack-local).
	type browseState struct {
		cancel dnsServiceCancel
	}
	state := new(browseState)

	req := dnsServiceBrowseRequest{
		Version:        dnsQueryRequestVersion1,
		InterfaceIndex: 0, // all interfaces
		QueryName:      queryNameW,
		BrowseCallback: browseCallback,
		QueryContext:   0,
	}

	r1, _, _ := procDnsServiceBrowse.Call(
		uintptr(unsafe.Pointer(&req)),
		uintptr(unsafe.Pointer(&state.cancel)),
	)
	if r1 != dnsRequestPending {
		return nil, fmt.Errorf("mdns WinAPI: DnsServiceBrowse returned %d (expected DNS_REQUEST_PENDING=%d)", r1, dnsRequestPending)
	}

	s.debugf("mdns WinAPI: browse started for %q", queryName)

	out := make(chan ServiceInstance)
	seen := make(map[string]bool)
	var seenMu sync.Mutex

	go func() {
		defer close(out)
		defer func() {
			procDnsServiceBrowseCancel.Call(uintptr(unsafe.Pointer(&state.cancel)))
			close(ptrCh)
		}()

		// Drain the timeout context.
		browseCtx, browseCancel := context.WithTimeout(ctx, s.opts.Timeout)
		defer browseCancel()

		for {
			select {
			case <-browseCtx.Done():
				return
			case fqdn, ok := <-ptrCh:
				if !ok {
					return
				}
				seenMu.Lock()
				dup := seen[fqdn]
				seen[fqdn] = true
				seenMu.Unlock()
				if dup {
					continue
				}

				s.debugf("mdns WinAPI: discovered %q", fqdn)

				if s.opts.Quick {
					inst := ServiceInstance{
						Name: stripServiceSuffix(fqdn, queryName),
					}
					select {
					case out <- inst:
					case <-browseCtx.Done():
						return
					}
					continue
				}

				// Resolve in a goroutine so slow hosts don't block the browse loop.
				go func(instanceFQDN string) {
					inst, err := s.winAPIResolve(browseCtx, instanceFQDN, queryName)
					if err != nil {
						if browseCtx.Err() == nil {
							s.opts.Logger.Printf("mdns WinAPI: resolve %q: %v", instanceFQDN, err)
						}
						return
					}
					select {
					case out <- inst:
					case <-browseCtx.Done():
					}
				}(fqdn)
			}
		}
	}()

	return out, nil
}

// winAPIResolve calls DnsServiceResolve for a single service instance FQDN and
// returns the populated ServiceInstance.
func (s *Scanner) winAPIResolve(ctx context.Context, instanceFQDN, serviceQueryName string) (ServiceInstance, error) {
	queryNameW, err := windows.UTF16PtrFromString(instanceFQDN)
	if err != nil {
		return ServiceInstance{}, err
	}

	type resolveResult struct {
		inst ServiceInstance
		err  error
	}
	done := make(chan resolveResult, 1)

	// The resolve callback fires on a Windows thread-pool thread.
	resolveCallback := windows.NewCallback(func(status uint32, _ uintptr, instPtr uintptr) uintptr {
		if status != 0 || instPtr == 0 {
			done <- resolveResult{err: fmt.Errorf("DnsServiceResolve status %d", status)}
			return 0
		}
		si := (*dnsServiceInstance)(unsafe.Pointer(instPtr))
		inst := ServiceInstance{
			Name: stripServiceSuffix(instanceFQDN, serviceQueryName),
			Port: int(si.Port),
		}
		if si.HostName != nil {
			inst.Host = windows.UTF16PtrToString(si.HostName)
		}
		if si.IPv4Address != nil {
			b := (*[4]byte)(unsafe.Pointer(si.IPv4Address))
			inst.IPs = append(inst.IPs, net.IP{b[0], b[1], b[2], b[3]})
		}
		if si.IPv6Address != nil {
			ip6 := make(net.IP, 16)
			copy(ip6, si.IPv6Address[:])
			inst.IPs = append(inst.IPs, ip6)
		}
		n := int(si.PropertyCount)
		if n > 0 && si.Keys != nil && si.Values != nil {
			keys := unsafe.Slice(si.Keys, n)
			vals := unsafe.Slice(si.Values, n)
			txtPairs := make([]string, 0, n)
			for i := 0; i < n; i++ {
				k := windows.UTF16PtrToString(keys[i])
				v := windows.UTF16PtrToString(vals[i])
				if k != "" {
					txtPairs = append(txtPairs, k+"="+v)
				}
			}
			inst.TXT = parseTXTSlice(txtPairs)
			inst.RawTXT = []byte(strings.Join(txtPairs, "\x00"))
		}
		procDnsServiceFreeInstance.Call(instPtr)
		done <- resolveResult{inst: inst}
		return 0
	})

	type resolveState struct {
		cancel dnsServiceCancel
	}
	rstate := new(resolveState)

	req := dnsServiceResolveRequest{
		Version:                   dnsQueryRequestVersion1,
		InterfaceIndex:            0,
		QueryName:                 queryNameW,
		ResolveCompletionCallback: resolveCallback,
		QueryContext:              0,
	}

	r1, _, _ := procDnsServiceResolve.Call(
		uintptr(unsafe.Pointer(&req)),
		uintptr(unsafe.Pointer(&rstate.cancel)),
	)
	if r1 != dnsRequestPending {
		return ServiceInstance{}, fmt.Errorf("DnsServiceResolve returned %d", r1)
	}

	select {
	case res := <-done:
		return res.inst, res.err
	case <-ctx.Done():
		procDnsServiceResolveCancel.Call(uintptr(unsafe.Pointer(&rstate.cancel)))
		return ServiceInstance{}, ctx.Err()
	}
}

// stripServiceSuffix removes the "._service._tcp.domain" suffix from a PTR
// target FQDN to produce the bare instance name.
// e.g. "MyDevice._tedge._tcp.local" → "MyDevice"
func stripServiceSuffix(fqdn, serviceQueryName string) string {
	// serviceQueryName is e.g. "_tedge._tcp.local" (no trailing dot)
	suffix := "." + serviceQueryName
	bare := strings.TrimSuffix(fqdn, ".")          // remove trailing dot if present
	bare = strings.TrimSuffix(bare, suffix)
	bare = strings.TrimSuffix(bare, "."+serviceQueryName) // also without trailing dot
	return bare
}
