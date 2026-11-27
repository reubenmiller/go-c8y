package realtime

import (
	"encoding/json"
	"testing"
)

// A /meta message has no data: it must still marshal, as the "ws (recv)" log of
// the client marshals every /meta message it receives.
func TestMessageMarshalWithoutData(t *testing.T) {
	msgs := []Message{}
	in := `[{"channel":"/meta/handshake","data":{},"id":"1","clientId":"c1","successful":true}]`
	if err := json.Unmarshal([]byte(in), &msgs); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(msgs[0])
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	want := `{"channel":"/meta/handshake","data":{},"id":"1","clientId":"c1","successful":true}`
	if string(out) != want {
		t.Errorf("got %s, want %s", out, want)
	}
}

func TestMessageMarshalWithData(t *testing.T) {
	msgs := []Message{}
	in := `[{"channel":"/alarms/1","data":{"realtimeAction":"CREATE","data":{"id":"5"}}}]`
	if err := json.Unmarshal([]byte(in), &msgs); err != nil {
		t.Fatal(err)
	}
	out, err := json.Marshal(msgs[0])
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if string(out) != in[1:len(in)-1] {
		t.Errorf("got %s, want %s", out, in[1:len(in)-1])
	}
}
