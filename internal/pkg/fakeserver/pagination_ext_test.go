package fakeserver

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// These tests cover the pagination/keyset primitives added so the fake server
// can exercise the SDK pagination strategies offline: inventory comparison
// operators (_id gt) and $orderby execution, deterministic time ordering with
// an id tiebreak, precision-adaptive date filtering, and the empty-collection
// envelope shape.

func docs(t *testing.T, objs ...map[string]any) []json.RawMessage {
	t.Helper()
	out := make([]json.RawMessage, 0, len(objs))
	for _, o := range objs {
		b, err := json.Marshal(o)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		out = append(out, b)
	}
	return out
}

func ids(items []json.RawMessage) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, getJSONString(it, "id"))
	}
	return out
}

func TestApplyCQLFilter_IDGreaterThanNumeric(t *testing.T) {
	items := docs(t,
		map[string]any{"id": "2"},
		map[string]any{"id": "3"},
		map[string]any{"id": "10"},
		map[string]any{"id": "20"},
		map[string]any{"id": "100"},
	)
	// Numeric comparison: 10, 20, 100 are all > 3 (lexical string compare would
	// wrongly drop "10" and "100" because "1" < "3").
	got := ids(applyCQLFilter(items, "$filter=(_id gt '3') $orderby=_id asc"))
	want := []string{"10", "20", "100"}
	if !equalStrings(got, want) {
		t.Fatalf("_id gt '3' = %v, want %v", got, want)
	}
}

func TestApplyCQLFilter_Operators(t *testing.T) {
	items := docs(t,
		map[string]any{"id": "5"},
		map[string]any{"id": "10"},
		map[string]any{"id": "15"},
	)
	cases := map[string][]string{
		"_id gt '10'": {"15"},
		"_id lt '10'": {"5"},
		"_id ge '10'": {"10", "15"},
		"_id le '10'": {"5", "10"},
		"_id ne '10'": {"5", "15"},
	}
	for expr, want := range cases {
		got := ids(applyCQLFilter(items, expr))
		if !equalStrings(got, want) {
			t.Errorf("%q = %v, want %v", expr, got, want)
		}
	}
}

func TestApplyCQLFilter_CursorWithFilterAndOrder(t *testing.T) {
	items := docs(t,
		map[string]any{"id": "1", "type": "ci_Test"},
		map[string]any{"id": "2", "type": "other"},
		map[string]any{"id": "30", "type": "ci_Test"},
		map[string]any{"id": "4", "type": "ci_Test"},
	)
	// Mirrors the inventory keyset query: cursor + original filter, ordered by id.
	got := ids(applyCQLFilter(items, "$filter=(_id gt '1' and (type eq 'ci_Test')) $orderby=_id asc"))
	got = ids(applyCQLOrderBy(toDocs(t, got), "$orderby=_id asc"))
	want := []string{"4", "30"} // numeric order, type-filtered, id>1
	if !equalStrings(got, want) {
		t.Fatalf("cursor query = %v, want %v", got, want)
	}
}

func TestApplyCQLOrderBy_NumericIDAscDesc(t *testing.T) {
	items := docs(t,
		map[string]any{"id": "20"},
		map[string]any{"id": "3"},
		map[string]any{"id": "100"},
		map[string]any{"id": "1"},
	)
	asc := ids(applyCQLOrderBy(items, "$filter=(_id gt '0') $orderby=_id asc"))
	if want := []string{"1", "3", "20", "100"}; !equalStrings(asc, want) {
		t.Errorf("asc = %v, want %v", asc, want)
	}
	desc := ids(applyCQLOrderBy(items, "$orderby=_id desc"))
	if want := []string{"100", "20", "3", "1"}; !equalStrings(desc, want) {
		t.Errorf("desc = %v, want %v", desc, want)
	}
}

func TestOrderTimeItems_DefaultNewestFirstWithIDTiebreak(t *testing.T) {
	// Two items share a timestamp; the tiebreak must be deterministic (numeric
	// id), descending by default.
	items := docs(t,
		map[string]any{"id": "1", "time": "2024-01-01T10:00:00Z"},
		map[string]any{"id": "2", "time": "2024-01-01T10:00:02Z"},
		map[string]any{"id": "3", "time": "2024-01-01T10:00:01Z"},
		map[string]any{"id": "4", "time": "2024-01-01T10:00:01Z"}, // dup of id 3
	)
	r := httptest.NewRequest("GET", "/x", nil)
	got := ids(OrderTimeItems(r, items))
	// time desc: 10:00:02 (id2), then 10:00:01 cluster newest-id-first (id4, id3), then 10:00:00 (id1)
	want := []string{"2", "4", "3", "1"}
	if !equalStrings(got, want) {
		t.Fatalf("default order = %v, want %v", got, want)
	}
}

func TestOrderTimeItems_RevertOldestFirst(t *testing.T) {
	items := docs(t,
		map[string]any{"id": "1", "time": "2024-01-01T10:00:00Z"},
		map[string]any{"id": "4", "time": "2024-01-01T10:00:01Z"},
		map[string]any{"id": "3", "time": "2024-01-01T10:00:01Z"}, // dup
		map[string]any{"id": "2", "time": "2024-01-01T10:00:02Z"},
	)
	r := httptest.NewRequest("GET", "/x?revert=true", nil)
	got := ids(OrderTimeItems(r, items))
	// ascending: 10:00:00 (id1), 10:00:01 cluster oldest-id-first (id3, id4), 10:00:02 (id2)
	want := []string{"1", "3", "4", "2"}
	if !equalStrings(got, want) {
		t.Fatalf("revert order = %v, want %v", got, want)
	}
}

func TestDateFilter_PrecisionAdaptive(t *testing.T) {
	// An item half a second past the boundary second.
	item := docs(t, map[string]any{"id": "1", "time": "2024-01-01T10:00:00.500Z"})[0]

	// Second-precision dateTo "10:00:00" -> compared at second granularity, so
	// the .500 item is treated as within the boundary second (included).
	if !dateBeforeOrEqual("2024-01-01T10:00:00Z")(item) {
		t.Error("second-precision dateTo should include same-second item")
	}
	// Millisecond-precision dateTo "10:00:00.000" -> compared at full precision,
	// so the .500 item is after the cursor (excluded). This is what lets a
	// millisecond keyset cursor position exactly between same-second items.
	if dateBeforeOrEqual("2024-01-01T10:00:00.000Z")(item) {
		t.Error("ms-precision dateTo .000 should exclude a .500 item")
	}
	// ms-precision dateTo at exactly .500 includes it (inclusive boundary).
	if !dateBeforeOrEqual("2024-01-01T10:00:00.500Z")(item) {
		t.Error("ms-precision dateTo .500 should include the .500 item (inclusive)")
	}
}

func TestBuildCollectionResponse_EmptyIsArrayNotNull(t *testing.T) {
	r := httptest.NewRequest("GET", "/alarm/alarms", nil)
	out := BuildCollectionResponse(r, "http://x", "alarms", Paginate(r, nil))
	var env map[string]json.RawMessage
	if err := json.Unmarshal(out, &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(env["alarms"]) != "[]" {
		t.Fatalf("empty collection = %s, want []", env["alarms"])
	}
}

// --- helpers ---

func toDocs(t *testing.T, idList []string) []json.RawMessage {
	t.Helper()
	objs := make([]map[string]any, 0, len(idList))
	for _, id := range idList {
		objs = append(objs, map[string]any{"id": id})
	}
	return docs(t, objs...)
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
