package fakeserver

import (
	"encoding/json"
	"testing"
	"time"
)

// The platform stores timestamps at millisecond resolution; the fake must not
// hand out finer-grained creationTime values.
func TestStoreCreate_MillisecondTimestamps(t *testing.T) {
	s := NewStore()
	for i := 0; i < 50; i++ {
		_, doc := s.Create(json.RawMessage(`{}`), "http://x")
		var m map[string]string
		if err := json.Unmarshal(doc, &m); err != nil {
			t.Fatal(err)
		}
		ts, err := time.Parse(time.RFC3339Nano, m["creationTime"])
		if err != nil {
			t.Fatal(err)
		}
		if !ts.Equal(ts.Truncate(time.Millisecond)) {
			t.Fatalf("creationTime %q has sub-millisecond precision", m["creationTime"])
		}
	}
}
