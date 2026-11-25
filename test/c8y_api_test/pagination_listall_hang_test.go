package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/operations"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/require"
)

// Regression: ListAll hung (or repeated items) when many items created back to
// back shared, or nearly shared, a timestamp. The cursor compared nanosecond
// timestamps while the platform filters at millisecond resolution. Every
// walk here runs under a deadline so a regression fails fast instead of hanging.
func TestListAll_BackToBackCreatesTerminate(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	dev := client.ManagedObjects.Create(ctx, map[string]any{"name": "listall-dev"})
	require.NoError(t, dev.Err)
	src := managedobjects.DeviceRef(dev.Data.ID())
	page := pagination.PaginationOptions{PageSize: 2}

	// withTime: all items carry the identical explicit time (a pure tie);
	// otherwise no time is sent and the server stamps them back to back.
	for _, withTime := range []bool{false, true} {
		var at time.Time
		if withTime {
			at = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		}
		name := map[bool]string{false: "server_time", true: "identical_time"}[withTime]
		t.Run("events/"+name, func(t *testing.T) {
			var want []string
			for i := 0; i < 7; i++ {
				r := client.Events.Create(ctx, events.CreateOptions{Source: src, Type: "ll_ev_" + name, Text: "e", Time: at})
				require.NoError(t, r.Err)
				want = append(want, r.Data.ID())
			}
			got := collectIDs(t, client.Events.ListAll(ctx, events.ListOptions{Source: src, Type: "ll_ev_" + name, PaginationOptions: page}),
				func(e jsonmodels.Event) string { return e.ID() })
			require.NoError(t, checkComplete(got, want))
		})
		t.Run("alarms/"+name, func(t *testing.T) {
			var want []string
			for i := 0; i < 7; i++ {
				r := client.Alarms.Create(ctx, alarms.CreateOptions{Source: src, Type: "ll_al_" + name, Text: "a", Severity: model.AlarmSeverityMajor, Time: at})
				require.NoError(t, r.Err)
				want = append(want, r.Data.ID())
			}
			got := collectIDs(t, client.Alarms.ListAll(ctx, alarms.ListOptions{Source: src, Type: []string{"ll_al_" + name}, PaginationOptions: page}),
				func(a jsonmodels.Alarm) string { return a.ID() })
			require.NoError(t, checkComplete(got, want))
		})
		t.Run("measurements/"+name, func(t *testing.T) {
			var want []string
			for i := 0; i < 7; i++ {
				r := client.Measurements.Create(ctx, measurements.CreateOptions{Source: src, Type: "ll_me_" + name, Time: at,
					Fragments: []model.Fragment{model.Frag("c8y_T", map[string]any{"T": map[string]any{"value": 1}})}})
				require.NoError(t, r.Err)
				want = append(want, r.Data.ID())
			}
			got := collectIDs(t, client.Measurements.ListAll(ctx, measurements.ListOptions{Source: src, Type: "ll_me_" + name, PaginationOptions: page}),
				func(m jsonmodels.Measurement) string { return m.ID() })
			require.NoError(t, checkComplete(got, want))
		})
	}

	t.Run("operations", func(t *testing.T) {
		var want []string
		for i := 0; i < 7; i++ {
			r := client.Operations.Create(ctx, operations.CreateOptions{DeviceID: src, Description: "op",
				Fragments: []model.Fragment{model.Frag("c8y_Restart", map[string]any{})}})
			require.NoError(t, r.Err)
			want = append(want, r.Data.ID())
		}
		got := collectIDs(t, client.Operations.ListAll(ctx, operations.ListOptions{DeviceID: src, PaginationOptions: page}),
			func(o jsonmodels.Operation) string { return o.ID() })
		require.NoError(t, checkComplete(got, want))
	})
}
