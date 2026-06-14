package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/require"
)

// This file holds the reusable pagination-invariant harness and proves it
// against the EXISTING offset paths. The same harness is reused by the keyset
// strategy tests (phases 3 and 4): a strategy is correct iff, for every dataset
// and page size, the items it yields are a duplicate-free enumeration whose set
// equals the seeded set (no skips), honouring MaxItems.

// checkComplete reports whether got is a duplicate-free enumeration whose set
// equals want. It returns a descriptive error naming duplicates, skips
// (missing) and unexpected (extra) ids, or nil when the enumeration is exact.
func checkComplete(got, want []string) error {
	seen := make(map[string]int, len(got))
	for _, id := range got {
		seen[id]++
	}
	wantSet := make(map[string]struct{}, len(want))
	for _, id := range want {
		wantSet[id] = struct{}{}
	}

	var dups, missing, extra []string
	for id, n := range seen {
		if n > 1 {
			dups = append(dups, id)
		}
		if _, ok := wantSet[id]; !ok {
			extra = append(extra, id)
		}
	}
	for _, id := range want {
		if seen[id] == 0 {
			missing = append(missing, id)
		}
	}
	if len(dups) == 0 && len(missing) == 0 && len(extra) == 0 {
		return nil
	}
	sort.Strings(dups)
	sort.Strings(missing)
	sort.Strings(extra)
	return fmt.Errorf("incomplete pagination (got %d, want %d): %d duplicate %v, %d missing %v, %d unexpected %v",
		len(got), len(want), len(dups), trunc(dups), len(missing), trunc(missing), len(extra), trunc(extra))
}

func trunc(s []string) []string {
	if len(s) > 8 {
		return append(s[:8:8], "...")
	}
	return s
}

// collectIDs drains an iterator, failing on any mid-iteration error, and returns
// the id of every yielded item in order.
func collectIDs[T any](t *testing.T, it *pagination.Iterator[T], idOf func(T) string) []string {
	t.Helper()
	var got []string
	for item, err := range it.Items() {
		require.NoError(t, err)
		got = append(got, idOf(item))
	}
	require.NoError(t, it.Err())
	return got
}

func requireOffline(t *testing.T, srv *fakeserver.FakeServer) {
	t.Helper()
	if srv == nil {
		t.Skip("pagination invariant tests require the fake server (offline mode)")
	}
}

// seedManagedObjects inserts n managed objects of the given type with known,
// numeric, monotonically increasing ids and returns those ids.
func seedManagedObjects(t *testing.T, srv *fakeserver.FakeServer, typ string, n int) []string {
	t.Helper()
	out := make([]string, 0, n)
	for i := 1; i <= n; i++ {
		id := strconv.Itoa(200000 + i)
		b, err := json.Marshal(map[string]any{
			"id":   id,
			"self": srv.URL() + "/inventory/managedObjects/" + id,
			"name": fmt.Sprintf("po-%05d", i),
			"type": typ,
		})
		require.NoError(t, err)
		srv.ManagedObjects.CreateWithID(id, b)
		out = append(out, id)
	}
	return out
}

// seedMeasurements inserts one measurement per supplied timestamp (repeat a
// timestamp to build a duplicate-timestamp cluster) and returns the created ids.
func seedMeasurements(t *testing.T, srv *fakeserver.FakeServer, source string, times []time.Time) []string {
	t.Helper()
	out := make([]string, 0, len(times))
	for _, tm := range times {
		b, err := json.Marshal(map[string]any{
			"source":   map[string]any{"id": source},
			"type":     "ci_PagMeas",
			"time":     tm.UTC().Format(time.RFC3339Nano),
			"ci_Value": map[string]any{"x": map[string]any{"value": 1.0}},
		})
		require.NoError(t, err)
		id, _ := srv.Measurements.Create(b, srv.URL()+"/measurement/measurements")
		out = append(out, id)
	}
	return out
}

// TestPaginationInvariants_HarnessDetectsErrors is the harness self-test: it
// must flag skips, duplicates and unexpected ids, otherwise a buggy strategy
// could pass silently in later phases.
func TestPaginationInvariants_HarnessDetectsErrors(t *testing.T) {
	want := []string{"a", "b", "c"}
	require.NoError(t, checkComplete([]string{"c", "a", "b"}, want), "exact set (any order) is complete")
	require.Error(t, checkComplete([]string{"a", "b"}, want), "must detect a skip")
	require.Error(t, checkComplete([]string{"a", "b", "b", "c"}, want), "must detect a duplicate")
	require.Error(t, checkComplete([]string{"a", "b", "c", "x"}, want), "must detect an unexpected id")
}

// TestPaginationInvariants_OffsetManagedObjects proves the existing offset path
// enumerates the full set exactly across a range of page sizes.
func TestPaginationInvariants_OffsetManagedObjects(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	const typ = "ci_PagOffsetMO"
	want := seedManagedObjects(t, srv, typ, 250)

	for _, ps := range []int{1, 7, 100, 2000} {
		it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
			Type:              typ,
			PaginationOptions: pagination.PaginationOptions{PageSize: ps, Strategy: pagination.StrategyOffset},
		})
		got := collectIDs(t, it, func(m jsonmodels.ManagedObject) string { return m.ID() })
		require.NoErrorf(t, checkComplete(got, want), "offset managed objects, pageSize=%d", ps)
	}
}

// TestPaginationInvariants_OffsetMeasurementsDuplicateTimestamps proves the
// offset baseline is complete even when many measurements share a timestamp,
// including a cluster larger than the page size. This is the baseline the time
// keyset strategy must match in phase 4.
func TestPaginationInvariants_OffsetMeasurementsDuplicateTimestamps(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	_ = client
	const source = "po_device_meas"

	base := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	var times []time.Time
	for i := 0; i < 200; i++ {
		times = append(times, base.Add(time.Duration(i)*time.Second))
	}
	// A 30-wide cluster all sharing one timestamp (exceeds the small page sizes).
	cluster := base.Add(500 * time.Second)
	for i := 0; i < 30; i++ {
		times = append(times, cluster)
	}
	want := seedMeasurements(t, srv, source, times)

	for _, ps := range []int{1, 13, 100} {
		it := client.Measurements.ListAll(context.Background(), measurements.ListOptions{
			Source:            managedobjects.DeviceRef(source),
			PaginationOptions: pagination.PaginationOptions{PageSize: ps, Strategy: pagination.StrategyOffset},
		})
		got := collectIDs(t, it, func(m jsonmodels.Measurement) string { return m.ID() })
		require.NoErrorf(t, checkComplete(got, want), "offset measurements, pageSize=%d", ps)
	}
}

// TestPaginationInvariants_OffsetRespectsMaxItems proves MaxItems caps the
// stream to exactly that many distinct items drawn from the seeded set.
func TestPaginationInvariants_OffsetRespectsMaxItems(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	const typ = "ci_PagMaxItems"
	want := seedManagedObjects(t, srv, typ, 250)

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type:              typ,
		PaginationOptions: pagination.PaginationOptions{PageSize: 10, MaxItems: 55, Strategy: pagination.StrategyOffset},
	})
	got := collectIDs(t, it, func(m jsonmodels.ManagedObject) string { return m.ID() })
	require.Len(t, got, 55)
	require.NoError(t, checkComplete(got, got), "MaxItems result must be duplicate-free")
	require.Subset(t, want, got, "MaxItems result must be a subset of the seeded set")
}
