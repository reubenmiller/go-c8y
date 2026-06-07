package model

import (
	"encoding/json"
	"testing"
)

func TestRawMarshalsValueNotWrapper(t *testing.T) {
	b, err := json.Marshal(Frag("c8y_Custom", map[string]any{"foo": "bar"}))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != `{"foo":"bar"}` {
		t.Errorf("Raw should marshal its Value, got %s", b)
	}
}

func TestMergeFragments(t *testing.T) {
	body, err := MergeFragments([]byte(`{"type":"c8y_Test"}`), []Fragment{
		Frag("count", 3),
		Frag("c8y_Custom", map[string]any{"a": 1}),
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatal(err)
	}
	if got["type"] != "c8y_Test" {
		t.Errorf("base field lost: %v", got)
	}
	if got["count"].(float64) != 3 {
		t.Errorf("scalar fragment not merged: %v", got)
	}
	if c, ok := got["c8y_Custom"].(map[string]any); !ok || c["a"].(float64) != 1 {
		t.Errorf("object fragment not merged: %v", got)
	}
}

func TestMergeFragmentsLaterWins(t *testing.T) {
	body, err := MergeFragments([]byte(`{}`), []Fragment{
		Frag("k", "first"),
		Frag("k", "second"),
	})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(body, &got)
	if got["k"] != "second" {
		t.Errorf("later fragment should win, got %v", got["k"])
	}
}

func TestMergeFragmentsSkipsNil(t *testing.T) {
	body, err := MergeFragments([]byte(`{"a":1}`), []Fragment{nil, Frag("b", 2)})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(body, &got)
	if got["b"].(float64) != 2 {
		t.Errorf("nil fragment should be skipped, b should merge: %v", got)
	}
}
