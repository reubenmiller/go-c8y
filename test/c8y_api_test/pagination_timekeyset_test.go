package api_test

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/require"
)

func measIDs(t *testing.T, it *pagination.Iterator[jsonmodels.Measurement]) []string {
	return collectIDs(t, it, func(m jsonmodels.Measurement) string { return m.ID() })
}

// timesDistinct builds n timestamps, one second apart (descending in age).
func timesDistinct(base time.Time, n int) []time.Time {
	out := make([]time.Time, n)
	for i := range out {
		out[i] = base.Add(time.Duration(i) * time.Second)
	}
	return out
}

// TestTimeKeyset_MeasurementsMatchOffset is the core correctness test: across a
// range of adversarial timestamp layouts and page sizes, the default time keyset
// must enumerate exactly the same items, in the same order, as offset paging —
// no skips, no duplicates, even across duplicate-timestamp page boundaries and
// clusters larger than a page.
func TestTimeKeyset_MeasurementsMatchOffset(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// Helper to build a timestamp slice from "runs": each run is (count, sameTime?).
	cluster := func(at time.Time, n int) []time.Time {
		out := make([]time.Time, n)
		for i := range out {
			out[i] = at
		}
		return out
	}

	datasets := map[string][]time.Time{
		"distinct": timesDistinct(base, 150),
		"pairs": append(append(append(
			timesDistinct(base, 40),
			cluster(base.Add(40*time.Second), 2)...),
			timesDistinct(base.Add(50*time.Second), 40)...),
			cluster(base.Add(100*time.Second), 2)...),
		"cluster_middle": append(append(
			timesDistinct(base, 50),
			cluster(base.Add(200*time.Second), 30)...),
			timesDistinct(base.Add(300*time.Second), 50)...),
		"cluster_newest": append(
			cluster(base.Add(400*time.Second), 30),
			timesDistinct(base, 100)...),
		"cluster_oldest": append(
			timesDistinct(base.Add(10*time.Second), 100),
			cluster(base, 30)...), // base is the oldest
		"all_same": cluster(base, 50),
		// A cluster on a sub-second timestamp: exercises the millisecond cursor
		// (a second-precision dateTo would truncate and loop forever here).
		"cluster_subsecond": append(append(
			timesDistinct(base, 40),
			cluster(base.Add(20*time.Second).Add(250*time.Millisecond), 30)...),
			timesDistinct(base.Add(60*time.Second), 40)...),
	}

	for name, times := range datasets {
		t.Run(name, func(t *testing.T) {
			source := "tk_" + name
			want := seedMeasurements(t, srv, source, times)

			offset := measIDs(t, client.Measurements.ListAll(context.Background(), measurements.ListOptions{
				Source:            managedobjects.DeviceRef(source),
				PaginationOptions: pagination.PaginationOptions{PageSize: 2000, Strategy: pagination.StrategyOffset},
			}))
			require.NoError(t, checkComplete(offset, want), "offset baseline incomplete")

			for _, ps := range []int{1, 7, 13, 100} {
				keyset := measIDs(t, client.Measurements.ListAll(context.Background(), measurements.ListOptions{
					Source:            managedobjects.DeviceRef(source),
					PaginationOptions: pagination.PaginationOptions{PageSize: ps}, // Auto -> time keyset
				}))
				require.NoErrorf(t, checkComplete(keyset, want), "time keyset incomplete, ps=%d", ps)
				require.Equalf(t, offset, keyset, "time keyset order must match offset, ps=%d", ps)
			}
		})
	}
}

// TestTimeKeyset_RequestPattern proves the optimisation is engaged: the cursor
// (dateTo) advances and is present on the follow-up requests.
func TestTimeKeyset_RequestPattern(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	const source = "tk_reqpattern"
	seedMeasurements(t, srv, source, timesDistinct(base, 45)) // 5 pages at pageSize 10

	srv.ResetRequests()
	it := client.Measurements.ListAll(context.Background(), measurements.ListOptions{
		Source:            managedobjects.DeviceRef(source),
		PaginationOptions: pagination.PaginationOptions{PageSize: 10},
	})
	_ = measIDs(t, it)

	withDateTo := 0
	for _, r := range srv.Requests() {
		if r.Method != "GET" || r.Path != "/measurement/measurements" {
			continue
		}
		vals, _ := url.ParseQuery(r.RawQuery)
		if vals.Get("dateTo") != "" {
			withDateTo++
		}
	}
	require.GreaterOrEqual(t, withDateTo, 3, "follow-up pages must carry an advancing dateTo cursor")
}

// TestTimeKeyset_MaxItems caps the time keyset to the newest N items, matching
// the offset prefix.
func TestTimeKeyset_MaxItems(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	const source = "tk_maxitems"
	seedMeasurements(t, srv, source, timesDistinct(base, 200))

	offset := measIDs(t, client.Measurements.ListAll(context.Background(), measurements.ListOptions{
		Source:            managedobjects.DeviceRef(source),
		PaginationOptions: pagination.PaginationOptions{PageSize: 2000, Strategy: pagination.StrategyOffset},
	}))
	keyset := measIDs(t, client.Measurements.ListAll(context.Background(), measurements.ListOptions{
		Source:            managedobjects.DeviceRef(source),
		PaginationOptions: pagination.PaginationOptions{PageSize: 10, MaxItems: 35},
	}))
	require.Len(t, keyset, 35)
	require.Equal(t, offset[:35], keyset, "MaxItems must return the newest 35 in order")
}

// TestTimeKeyset_InapplicableErrors rejects the id keyset on a time-series list.
func TestTimeKeyset_InapplicableErrors(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	const source = "tk_err"
	seedMeasurements(t, srv, source, timesDistinct(time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), 5))

	it := client.Measurements.ListAll(context.Background(), measurements.ListOptions{
		Source:            managedobjects.DeviceRef(source),
		PaginationOptions: pagination.PaginationOptions{Strategy: pagination.StrategyIDKeyset},
	})
	n := 0
	for _, err := range it.Items() {
		if err != nil {
			break
		}
		n++
	}
	require.Error(t, it.Err(), "id keyset must be rejected for measurements")
	require.Zero(t, n)
}

// TestTimeKeyset_OtherEntitiesComplete is a cross-entity completeness smoke test
// for events and alarms (same TimeKeysetStrategy) including a duplicate-timestamp
// cluster larger than the page size.
func TestTimeKeyset_OtherEntitiesComplete(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	base := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	// 80 distinct + a 25-wide cluster.
	times := timesDistinct(base, 80)
	clusterAt := base.Add(40 * time.Second).Add(500 * time.Millisecond)
	for i := 0; i < 25; i++ {
		times = append(times, clusterAt)
	}

	const source = "tk_other"
	wantEvents := seedEvents(t, srv, source, times)
	wantAlarms := seedAlarms(t, srv, source, times)

	for _, ps := range []int{7, 13} {
		ev := collectIDs(t, client.Events.ListAll(context.Background(), events.ListOptions{
			Source:            managedobjects.DeviceRef(source),
			PaginationOptions: pagination.PaginationOptions{PageSize: ps},
		}), func(e jsonmodels.Event) string { return e.ID() })
		require.NoErrorf(t, checkComplete(ev, wantEvents), "events time keyset, ps=%d", ps)

		al := collectIDs(t, client.Alarms.ListAll(context.Background(), alarms.ListOptions{
			Source:            managedobjects.DeviceRef(source),
			PaginationOptions: pagination.PaginationOptions{PageSize: ps},
		}), func(a jsonmodels.Alarm) string { return a.ID() })
		require.NoErrorf(t, checkComplete(al, wantAlarms), "alarms time keyset, ps=%d", ps)
	}
}

// seedTimed inserts one document per timestamp into the given store and returns
// the created ids. Shared by the event/alarm seeders.
func seedTimed(t *testing.T, store *fakeserver.Store, source, typ string, times []time.Time, selfURL string) []string {
	t.Helper()
	out := make([]string, 0, len(times))
	for _, tm := range times {
		b, err := json.Marshal(map[string]any{
			"source": map[string]any{"id": source},
			"type":   typ,
			"text":   "x",
			"time":   tm.UTC().Format(time.RFC3339Nano),
		})
		require.NoError(t, err)
		id, _ := store.Create(b, selfURL)
		out = append(out, id)
	}
	return out
}

func seedEvents(t *testing.T, srv *fakeserver.FakeServer, source string, times []time.Time) []string {
	return seedTimed(t, srv.Events, source, "ci_PagEvent", times, srv.URL()+"/event/events")
}

func seedAlarms(t *testing.T, srv *fakeserver.FakeServer, source string, times []time.Time) []string {
	return seedTimed(t, srv.Alarms, source, "ci_PagAlarm", times, srv.URL()+"/alarm/alarms")
}
