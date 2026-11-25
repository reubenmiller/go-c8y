package pagination

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// stuckStrategy never advances: it keeps asking for the same request.
type stuckStrategy struct{ OffsetStrategy }

func (stuckStrategy) Advance(prev PageRequest, _ PageView) (PageRequest, bool) { return prev, true }

// A strategy that re-requests the same page must produce an error, not loop.
func TestExecutor_NoProgressGuard(t *testing.T) {
	for name, ra := range map[string]int{"sequential": 0, "read-ahead": 2} {
		t.Run(name, func(t *testing.T) {
			srv := &mockServer{total: 10}
			it := PaginateWith(context.Background(),
				PageRequest{PaginationOptions: PaginationOptions{PageSize: 2, ReadAhead: ra}},
				stuckStrategy{}, srv.fetch, idCtor)

			done := make(chan error, 1)
			go func() { _, err := collect(it); done <- err }()
			select {
			case err := <-done:
				if err == nil || !strings.Contains(err.Error(), "no progress") {
					t.Fatalf("want a no-progress error, got %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("walk did not terminate")
			}
			if n := srv.calls.Load(); n > 4 {
				t.Fatalf("too many requests before stopping: %d", n)
			}
		})
	}
}

// Sub-millisecond timestamps must collapse to the platform's millisecond
// resolution so items in the same millisecond are ties.
func TestDocTimestamp_MillisecondResolution(t *testing.T) {
	a := docTimestamp(jsondoc.New([]byte(`{"time":"2026-01-01T10:00:00.123456Z"}`)))
	b := docTimestamp(jsondoc.New([]byte(`{"creationTime":"2026-01-01T10:00:00.123999Z"}`)))
	if !a.Equal(b) {
		t.Fatalf("same-millisecond timestamps differ: %v vs %v", a, b)
	}
}
