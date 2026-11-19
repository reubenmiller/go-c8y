package pagination

import (
	"errors"
	"iter"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

func docs(raws ...string) iter.Seq2[jsondoc.JSONDoc, error] {
	return func(yield func(jsondoc.JSONDoc, error) bool) {
		for _, r := range raws {
			if !yield(jsondoc.New([]byte(r)), nil) {
				return
			}
		}
	}
}

func TestIteratorMapAndAs(t *testing.T) {
	it := NewIterator(docs(`{"name":"a"}`, `{"name":"b"}`))
	var names []string
	for n, err := range it.Map(func(d jsondoc.JSONDoc) string { return d.Get("name").String() }) {
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, n)
	}
	if len(names) != 2 || names[1] != "b" {
		t.Fatalf("names = %v", names)
	}

	type item struct {
		Name string `json:"name"`
	}
	var items []item
	for v, err := range NewIterator(docs(`{"name":"x"}`)).As[item]() {
		if err != nil {
			t.Fatal(err)
		}
		items = append(items, v)
	}
	if len(items) != 1 || items[0].Name != "x" {
		t.Fatalf("items = %v", items)
	}
}

func TestIteratorMapStopsAtError(t *testing.T) {
	failing := func(yield func(jsondoc.JSONDoc, error) bool) {
		yield(jsondoc.JSONDoc{}, errors.New("page failed"))
	}
	for _, err := range NewIterator(iter.Seq2[jsondoc.JSONDoc, error](failing)).Map(func(d jsondoc.JSONDoc) string { return "" }) {
		if err == nil || err.Error() != "page failed" {
			t.Fatalf("err = %v", err)
		}
	}
}
