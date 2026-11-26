package op

import (
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

func TestResultRawBody(t *testing.T) {
	arr := docModel{jsondoc.New([]byte(`[{"id":"1"},{"id":"2"}]`))}
	items := docModel{jsondoc.New([]byte(`[{"id":"1"}]`))}
	emptyArr := docModel{jsondoc.New([]byte(`[]`))}
	obj := docModel{jsondoc.New([]byte(`{"id":"1"}`))}
	envelope := []byte(`{"next":null,"managedObjects":[{"id":"1"}]}`)
	emptyEnvelope := []byte(`{"managedObjects":[]}`)

	withResp := func(d docModel, resp []byte) Result[docModel] { return WithResponse(OK(d), resp) }
	dry, _ := http.NewRequest(http.MethodGet, "http://x", nil)

	cases := []struct {
		name  string
		got   Body
		raw   string
		kind  BodyKind
		empty bool
	}{
		{"top-level array", OK(arr).RawBody(), `[{"id":"1"},{"id":"2"}]`, BodyArray, false},
		{"top-level array retained", withResp(arr, []byte(`[{"id":"1"},{"id":"2"}]`)).RawBody(), `[{"id":"1"},{"id":"2"}]`, BodyArray, false},
		{"envelope with items", withResp(items, envelope).RawBody(), string(envelope), BodyEnvelope, false},
		{"empty envelope", withResp(emptyArr, emptyEnvelope).RawBody(), string(emptyEnvelope), BodyEnvelope, true},
		{"empty array", OK(emptyArr).RawBody(), `[]`, BodyArray, true},
		{"object", OK(obj).RawBody(), "", BodyNone, false},
		{"error", Failed[docModel](http.ErrAbortHandler, false).RawBody(), "", BodyNone, false},
		{"no content", NoContent(docModel{}).RawBody(), "", BodyNone, false},
		{"zero result", Result[docModel]{}.RawBody(), "", BodyNone, false},
		{"dry run", WithRequest(withResp(items, envelope), dry).RawBody(), "", BodyNone, false},
		{"non-unwrapper data", OK(map[string]string{"a": "b"}).RawBody(), "", BodyNone, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if string(c.got.Raw) != c.raw || c.got.Kind != c.kind || c.got.Empty != c.empty {
				t.Fatalf("RawBody() = {%q %v %v}, want {%q %v %v}", c.got.Raw, c.got.Kind, c.got.Empty, c.raw, c.kind, c.empty)
			}
			if c.got.Has() != (c.kind != BodyNone) {
				t.Fatalf("Has() = %v for kind %v", c.got.Has(), c.kind)
			}
			if c.kind != BodyNone {
				if got := string(c.got.Doc().Raw()); got != c.raw {
					t.Fatalf("Doc() = %q, want %q", got, c.raw)
				}
			} else if len(c.got.Doc().Get().Map()) != 0 {
				t.Fatalf("Doc() of no body should be empty, got %q", c.got.Doc().Raw())
			}
		})
	}
}

func TestRawBodySource(t *testing.T) {
	var src RawBodySource = WithResponse(OK(docModel{jsondoc.New([]byte(`[]`))}), []byte(`{"a":[]}`))
	if b := src.RawBody(); b.Kind != BodyEnvelope || !b.Empty {
		t.Fatalf("got %+v", b)
	}
	if (Body{}).Has() || len((Body{}).Doc().Get().Map()) != 0 {
		t.Fatal("zero Body should be empty")
	}
	if BodyArray.String() != "array" || BodyEnvelope.String() != "envelope" || BodyNone.String() != "none" {
		t.Fatal("String()")
	}
}
