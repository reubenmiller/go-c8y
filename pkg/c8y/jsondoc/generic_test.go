package jsondoc

import (
	"errors"
	"testing"
)

type position struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (position) FragmentKey() string { return "c8y_Position" }

func TestJSONDocGenericMethods(t *testing.T) {
	d := New([]byte(`{"name":"dev1","c8y_Battery":{"level":87},"c8y_Position":{"lat":1.5,"lng":2.5}}`))

	level, err := d.GetAs[int]("c8y_Battery.level")
	if err != nil || level != 87 {
		t.Fatalf("GetAs = %d, %v", level, err)
	}
	if _, err := d.GetAs[int]("missing"); !errors.Is(err, ErrKeyNotFound) {
		t.Fatalf("missing key err = %v", err)
	}
	pos, err := d.Fragment[position]()
	if err != nil || pos.Lat != 1.5 {
		t.Fatalf("Fragment = %+v, %v", pos, err)
	}
	whole, err := d.Decode[map[string]any]()
	if err != nil || whole["name"] != "dev1" {
		t.Fatalf("Decode = %v, %v", whole, err)
	}
}
