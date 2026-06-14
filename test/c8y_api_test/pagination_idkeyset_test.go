package api_test

import (
	"context"
	"net/url"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/require"
)

// inventoryQueries returns the decoded inventory query string (the "query" or
// "q" param) of every GET /inventory/managedObjects collection request.
func inventoryQueries(reqs []fakeserver.CapturedRequest) []string {
	var out []string
	for _, r := range reqs {
		if r.Method != "GET" || r.Path != "/inventory/managedObjects" {
			continue
		}
		vals, _ := url.ParseQuery(r.RawQuery)
		q := vals.Get("query")
		if q == "" {
			q = vals.Get("q")
		}
		// Include offset requests too (they carry no query, only filters).
		out = append(out, "currentPage="+vals.Get("currentPage")+" | "+q)
	}
	return out
}

// TestIDKeyset_CompleteAndMatchesOffset proves the default (id keyset) walk
// enumerates the full set, in the same order as offset, across page sizes — for
// both managed objects and devices.
func TestIDKeyset_CompleteAndMatchesOffset(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	const typ = "ci_IDKeyset"
	want := seedManagedObjects(t, srv, typ, 250)

	for _, ps := range []int{1, 7, 100, 2000} {
		// Default (Auto -> id keyset)
		itKeyset := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
			Type:              typ,
			PaginationOptions: pagination.PaginationOptions{PageSize: ps},
		})
		keyset := collectIDs(t, itKeyset, func(m jsonmodels.ManagedObject) string { return m.ID() })
		require.NoErrorf(t, checkComplete(keyset, want), "id keyset, pageSize=%d", ps)

		// Explicit offset for comparison
		itOffset := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
			Type:              typ,
			PaginationOptions: pagination.PaginationOptions{PageSize: ps, Strategy: pagination.StrategyOffset},
		})
		offset := collectIDs(t, itOffset, func(m jsonmodels.ManagedObject) string { return m.ID() })
		require.Equalf(t, offset, keyset, "id keyset must match offset order, pageSize=%d", ps)
	}

	// Devices share the inventory backend and the id keyset default.
	itDev := client.Devices.ListAll(context.Background(), devices.ListOptions{
		Query:             "$filter=(type eq '" + typ + "')",
		PaginationOptions: pagination.PaginationOptions{PageSize: 13},
	})
	dev := collectIDs(t, itDev, func(m jsonmodels.ManagedObject) string { return m.ID() })
	require.NoError(t, checkComplete(dev, want), "devices id keyset")
}

// TestIDKeyset_RequestPattern proves the optimisation is actually engaged: every
// keyset request re-asks for page 1 with an advancing "_id gt" cursor, while
// offset increments currentPage and never sends a cursor.
func TestIDKeyset_RequestPattern(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	const typ = "ci_IDKeysetReq"
	seedManagedObjects(t, srv, typ, 45) // 5 pages at pageSize 10

	// id keyset
	srv.ResetRequests()
	itKeyset := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type:              typ,
		PaginationOptions: pagination.PaginationOptions{PageSize: 10},
	})
	_ = collectIDs(t, itKeyset, func(m jsonmodels.ManagedObject) string { return m.ID() })
	keysetReqs := inventoryQueries(srv.Requests())
	require.GreaterOrEqual(t, len(keysetReqs), 4, "expected multiple keyset pages")
	for _, q := range keysetReqs {
		require.Contains(t, q, "_id gt ", "keyset request must carry an _id cursor: %s", q)
		require.Contains(t, q, "currentPage=1", "keyset must always re-request page 1: %s", q)
		require.Contains(t, q, "$orderby=_id asc", "keyset must order by _id asc: %s", q)
	}

	// offset
	srv.ResetRequests()
	itOffset := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type:              typ,
		PaginationOptions: pagination.PaginationOptions{PageSize: 10, Strategy: pagination.StrategyOffset},
	})
	_ = collectIDs(t, itOffset, func(m jsonmodels.ManagedObject) string { return m.ID() })
	offsetReqs := inventoryQueries(srv.Requests())
	require.GreaterOrEqual(t, len(offsetReqs), 4)
	sawPage2 := false
	for _, q := range offsetReqs {
		require.NotContains(t, q, "_id gt ", "offset must not inject an _id cursor: %s", q)
		if strings.Contains(q, "currentPage=2") {
			sawPage2 = true
		}
	}
	require.True(t, sawPage2, "offset must increment currentPage")
}

// TestIDKeyset_ConflictingOrderByFallsBackToOffset proves Auto preserves a
// caller-supplied non-_id $orderby by using offset (no _id cursor injected), and
// the result is still complete.
func TestIDKeyset_ConflictingOrderByFallsBackToOffset(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	const typ = "ci_IDKeysetOrder"
	want := seedManagedObjects(t, srv, typ, 60)

	srv.ResetRequests()
	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Query:             "$filter=(type eq '" + typ + "') $orderby=name",
		PaginationOptions: pagination.PaginationOptions{PageSize: 10},
	})
	got := collectIDs(t, it, func(m jsonmodels.ManagedObject) string { return m.ID() })
	require.NoError(t, checkComplete(got, want))
	for _, q := range inventoryQueries(srv.Requests()) {
		require.NotContains(t, q, "_id gt ", "Auto must fall back to offset when a non-_id $orderby is present: %s", q)
	}
}

// TestIDKeyset_ExplicitInapplicableErrors proves an explicit, inapplicable
// strategy is a hard error surfaced via Iterator.Err() with no items yielded.
func TestIDKeyset_ExplicitInapplicableErrors(t *testing.T) {
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	requireOffline(t, srv)
	const typ = "ci_IDKeysetErr"
	seedManagedObjects(t, srv, typ, 10)

	cases := map[string]managedobjects.ListOptions{
		"time keyset on inventory": {
			Type:              typ,
			PaginationOptions: pagination.PaginationOptions{Strategy: pagination.StrategyTimeKeyset},
		},
		"id keyset with conflicting orderby": {
			Query:             "$filter=(type eq '" + typ + "') $orderby=name",
			PaginationOptions: pagination.PaginationOptions{Strategy: pagination.StrategyIDKeyset},
		},
	}
	for name, opts := range cases {
		it := client.ManagedObjects.ListAll(context.Background(), opts)
		n := 0
		for _, err := range it.Items() {
			if err != nil {
				break
			}
			n++
		}
		require.Errorf(t, it.Err(), "%s: expected a hard error", name)
		require.Zerof(t, n, "%s: expected no items", name)
	}
}
