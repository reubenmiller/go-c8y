package applications

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// binaryServer is a minimal application binaries API for RotateBinaries tests.
type binaryServer struct {
	mu       sync.Mutex
	activeID string
	// binaries in listing order, each with a creation time
	binaries []struct{ id, created string }
	// ids whose delete returns 500
	failDelete map[string]bool
	// ids whose delete returns 404
	missing map[string]bool
	deleted []string
	appGets int
}

func (b *binaryServer) handler(w http.ResponseWriter, r *http.Request) {
	b.mu.Lock()
	defer b.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/application/applications/123":
		b.appGets++
		_, _ = fmt.Fprintf(w, `{"id":"123","activeVersionId":%q}`, b.activeID)
	case r.Method == http.MethodGet && r.URL.Path == "/application/applications/123/binaries":
		items := []string{}
		for _, bin := range b.binaries {
			items = append(items, fmt.Sprintf(`{"id":%q,"created":%q}`, bin.id, bin.created))
		}
		_, _ = fmt.Fprintf(w, `{"attachments":[%s]}`, strings.Join(items, ","))
	case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/application/applications/123/binaries/"):
		id := strings.TrimPrefix(r.URL.Path, "/application/applications/123/binaries/")
		if b.failDelete[id] {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"general/internalError","message":"boom"}`))
			return
		}
		if b.missing[id] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b.deleted = append(b.deleted, id)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// newBinaryServer returns binaries b1..bn, created a day apart with b1 the
// oldest, listed in a shuffled order so the test relies on the creation time.
func newBinaryServer(n int, activeID string) *binaryServer {
	b := &binaryServer{activeID: activeID, failDelete: map[string]bool{}}
	for i := n; i >= 1; i-- {
		b.binaries = append(b.binaries, struct{ id, created string }{
			id:      fmt.Sprintf("b%d", i),
			created: fmt.Sprintf("2026-01-%02dT00:00:00.000Z", i),
		})
	}
	// Listing order: bn, b1, b2, ..., b(n-1), so it differs from creation order
	slices.Reverse(b.binaries[1:])
	return b
}

func metaIDs(t *testing.T, r op.Result[any], key string) []string {
	t.Helper()
	ids, ok := r.Meta[key].([]string)
	if !ok {
		t.Fatalf("Meta[%q] = %#v, want []string", key, r.Meta[key])
	}
	return ids
}

func rotate(t *testing.T, b *binaryServer, ctx context.Context, opt RotateBinariesOptions) op.Result[any] {
	t.Helper()
	svc, closeFn := testService(b.handler)
	defer closeFn()
	res := svc.RotateBinaries(ctx, "123", opt)
	return op.Result[any]{Status: res.Status, Err: res.Err, Meta: res.Meta}
}

func TestRotateBinariesKeepsNewest(t *testing.T) {
	b := newBinaryServer(5, "b5")
	res := rotate(t, b, context.Background(), RotateBinariesOptions{Keep: 2})
	if res.Err != nil {
		t.Fatalf("RotateBinaries: %v", res.Err)
	}
	if res.Status != op.StatusUpdated {
		t.Errorf("status = %s, want %s", res.Status, op.StatusUpdated)
	}
	want := []string{"b1", "b2", "b3"}
	if !slices.Equal(b.deleted, want) {
		t.Errorf("deleted = %v, want %v", b.deleted, want)
	}
	if got := metaIDs(t, res, "deletedBinaries"); !slices.Equal(got, want) {
		t.Errorf("Meta deletedBinaries = %v, want %v", got, want)
	}
	if res.Meta["applicationId"] != "123" {
		t.Errorf("Meta applicationId = %v, want 123", res.Meta["applicationId"])
	}
	if b.appGets != 1 {
		t.Errorf("application gets = %d, want 1 (active version lookup)", b.appGets)
	}
}

func TestRotateBinariesProtectsOldestActive(t *testing.T) {
	b := newBinaryServer(4, "b1")
	res := rotate(t, b, context.Background(), RotateBinariesOptions{Keep: 2})
	if res.Err != nil {
		t.Fatalf("RotateBinaries: %v", res.Err)
	}
	// b1 is active and counts towards keep, so only b4 is kept with it
	want := []string{"b2", "b3"}
	if !slices.Equal(b.deleted, want) {
		t.Errorf("deleted = %v, want %v", b.deleted, want)
	}
}

func TestRotateBinariesExplicitActiveAndProtect(t *testing.T) {
	b := newBinaryServer(5, "b5")
	res := rotate(t, b, context.Background(), RotateBinariesOptions{
		Keep:            1,
		ActiveVersionID: "b2",
		Protect:         []string{"b1", ""},
	})
	if res.Err != nil {
		t.Fatalf("RotateBinaries: %v", res.Err)
	}
	if b.appGets != 0 {
		t.Errorf("application gets = %d, want 0 when ActiveVersionID is set", b.appGets)
	}
	// b1 and b2 are protected; they already fill keep, so all others go
	want := []string{"b3", "b4", "b5"}
	if !slices.Equal(b.deleted, want) {
		t.Errorf("deleted = %v, want %v", b.deleted, want)
	}
}

func TestRotateBinariesNothingToDelete(t *testing.T) {
	b := newBinaryServer(2, "b2")
	res := rotate(t, b, context.Background(), RotateBinariesOptions{Keep: 2})
	if res.Err != nil {
		t.Fatalf("RotateBinaries: %v", res.Err)
	}
	if res.Status != op.StatusOK {
		t.Errorf("status = %s, want %s", res.Status, op.StatusOK)
	}
	if len(b.deleted) != 0 || len(metaIDs(t, res, "deletedBinaries")) != 0 {
		t.Errorf("deleted = %v, want none", b.deleted)
	}
}

func TestRotateBinariesKeepNotSet(t *testing.T) {
	for _, keep := range []int{0, -1} {
		b := newBinaryServer(3, "b3")
		res := rotate(t, b, context.Background(), RotateBinariesOptions{Keep: keep})
		if res.Err != nil {
			t.Fatalf("keep=%d: RotateBinaries: %v", keep, res.Err)
		}
		if res.Status != op.StatusSkipped {
			t.Errorf("keep=%d: status = %s, want %s", keep, res.Status, op.StatusSkipped)
		}
		if len(b.deleted) != 0 || b.appGets != 0 {
			t.Errorf("keep=%d: deleted = %v, app gets = %d, want no requests", keep, b.deleted, b.appGets)
		}
		if got := metaIDs(t, res, "deletedBinaries"); len(got) != 0 {
			t.Errorf("keep=%d: Meta deletedBinaries = %v, want empty", keep, got)
		}
	}
}

func TestRotateBinariesDryRun(t *testing.T) {
	b := newBinaryServer(4, "b4")
	ctx := ctxhelpers.WithDryRun(context.Background(), true)
	res := rotate(t, b, ctx, RotateBinariesOptions{Keep: 2})
	if res.Err != nil {
		t.Fatalf("RotateBinaries: %v", res.Err)
	}
	if len(b.deleted) != 0 {
		t.Errorf("deleted = %v, want none in dry run", b.deleted)
	}
	if got := metaIDs(t, res, "deletedBinaries"); len(got) != 0 {
		t.Errorf("Meta deletedBinaries = %v, want empty in dry run", got)
	}
	if got, want := metaIDs(t, res, "binariesToDelete"), []string{"b1", "b2"}; !slices.Equal(got, want) {
		t.Errorf("Meta binariesToDelete = %v, want %v", got, want)
	}
}

func TestRotateBinariesFailedDeleteContinues(t *testing.T) {
	b := newBinaryServer(5, "b5")
	b.failDelete["b2"] = true
	res := rotate(t, b, context.Background(), RotateBinariesOptions{Keep: 2})
	if res.Err == nil || !strings.Contains(res.Err.Error(), "binaryId=b2") {
		t.Fatalf("err = %v, want an error naming b2", res.Err)
	}
	if res.Status != op.StatusPartial {
		t.Errorf("status = %s, want %s", res.Status, op.StatusPartial)
	}
	want := []string{"b1", "b3"}
	if !slices.Equal(b.deleted, want) {
		t.Errorf("deleted = %v, want %v", b.deleted, want)
	}
	if got := metaIDs(t, res, "deletedBinaries"); !slices.Equal(got, want) {
		t.Errorf("Meta deletedBinaries = %v, want %v", got, want)
	}
	if got, want := metaIDs(t, res, "binariesToDelete"), []string{"b1", "b2", "b3"}; !slices.Equal(got, want) {
		t.Errorf("Meta binariesToDelete = %v, want %v", got, want)
	}
}

func TestRotateBinariesDeferred(t *testing.T) {
	b := newBinaryServer(3, "b3")
	svc, closeFn := testService(b.handler)
	defer closeFn()

	ctx := ctxhelpers.WithDeferredExecution(context.Background(), true)
	res := svc.RotateBinaries(ctx, "123", RotateBinariesOptions{Keep: 1})
	if !res.IsDeferred() {
		t.Fatalf("result is not deferred")
	}
	if len(b.deleted) != 0 {
		t.Fatalf("deleted = %v before Execute, want none", b.deleted)
	}
	res = res.Execute(context.Background())
	if res.Err != nil {
		t.Fatalf("Execute: %v", res.Err)
	}
	if want := []string{"b1", "b2"}; !slices.Equal(b.deleted, want) {
		t.Errorf("deleted = %v, want %v", b.deleted, want)
	}
}
