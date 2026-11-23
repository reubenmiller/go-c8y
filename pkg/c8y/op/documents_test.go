package op

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type docModel struct{ jsondoc.JSONDoc }

func collect[T any](r Result[T]) (docs []string, errs []error) {
	for d, err := range r.Documents() {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		docs = append(docs, string(d.Raw()))
	}
	return
}

func TestResultDocuments(t *testing.T) {
	boom := errors.New("boom")
	cases := []struct {
		name string
		docs []string
		errs int
		got  func() ([]string, []error)
	}{
		{"unwrapper", []string{`{"id":"1"}`}, 0, func() ([]string, []error) {
			return collect(OK(docModel{jsondoc.New([]byte(`{"id":"1"}`))}))
		}},
		{"empty unwrapper", nil, 0, func() ([]string, []error) { return collect(OK(docModel{})) }},
		{"error", nil, 1, func() ([]string, []error) { return collect(Failed[docModel](boom, false)) }},
		{"no content", nil, 0, func() ([]string, []error) { return collect(NoContent(struct{}{})) }},
		{"dry run", nil, 0, func() ([]string, []error) {
			req, _ := http.NewRequest(http.MethodPost, "http://x", nil)
			return collect(OK(map[string]string{}).WithRequest(req))
		}},
		{"deferred", nil, 0, func() ([]string, []error) {
			return collect(Result[docModel]{}.WithExecutor(func(context.Context) Result[docModel] { return Result[docModel]{} }))
		}},
		{"plain value", []string{`{"a":"b"}`}, 0, func() ([]string, []error) { return collect(OK(map[string]string{"a": "b"})) }},
		{"raw message", []string{`[1]`}, 0, func() ([]string, []error) { return collect(OK(json.RawMessage(`[1]`))) }},
		{"nil map", nil, 0, func() ([]string, []error) { return collect(OK(map[string]string(nil))) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			docs, errs := c.got()
			if len(errs) != c.errs {
				t.Fatalf("errors = %v, want %d", errs, c.errs)
			}
			if len(docs) != len(c.docs) {
				t.Fatalf("docs = %v, want %v", docs, c.docs)
			}
			for i := range docs {
				if docs[i] != c.docs[i] {
					t.Errorf("doc %d = %s, want %s", i, docs[i], c.docs[i])
				}
			}
		})
	}
}
