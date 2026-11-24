package source

import (
	"context"
	"encoding/json"
	"testing"
)

func TestResolveBodyRefs(t *testing.T) {
	var asked []string
	resolve := func(_ context.Context, ref string) (string, error) {
		asked = append(asked, ref)
		return "42", nil
	}
	in := json.RawMessage(`{"a":{"id":"dev01"},"b":"7","c":1}`)
	out, err := ResolveBodyRefs(context.Background(), in, resolve, "a.id", "b", "c", "missing")
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out.(json.RawMessage)); got != `{"a":{"id":"42"},"b":"7","c":1}` {
		t.Errorf("body = %s", got)
	}
	if len(asked) != 1 || asked[0] != "name:dev01" {
		t.Errorf("resolver asked %v", asked)
	}
	if string(in) != `{"a":{"id":"dev01"},"b":"7","c":1}` {
		t.Errorf("input body was modified: %s", in)
	}

	ids := map[string]any{"a": map[string]any{"id": "1"}}
	same, err := ResolveBodyRefs(context.Background(), ids, resolve, "a.id")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := same.(map[string]any); !ok {
		t.Errorf("a body without lookups must be returned as given, got %T", same)
	}
}
