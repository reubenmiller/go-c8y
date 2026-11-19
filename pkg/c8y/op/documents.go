package op

import (
	"encoding/json"
	"iter"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// DocumentSource is implemented by every value that can be rendered as a
// stream of JSON documents: an op.Result and a pagination iterator. It lets a
// consumer such as a CLI render the outcome of any service call through one
// code path, without choosing an adapter per result type.
type DocumentSource interface {
	Documents() iter.Seq2[jsondoc.JSONDoc, error]
}

// Documents returns the result as a stream of JSON documents:
//
//   - a failed result yields its error once;
//   - a result with nothing to show yields nothing: dry runs and deferred
//     results (the request was not sent), 204 No Content and skipped results;
//   - Data implementing jsondoc.Unwrapper yields its document (if non-empty);
//   - json.RawMessage / []byte Data yields it as is;
//   - any other Data is JSON-encoded (e.g. the map of a tenant-options call).
func (r Result[T]) Documents() iter.Seq2[jsondoc.JSONDoc, error] {
	return func(yield func(jsondoc.JSONDoc, error) bool) {
		if r.Err != nil {
			yield(jsondoc.Empty(), r.Err)
			return
		}
		if r.Request != nil || r.IsDeferred() {
			return // dry run or deferred: nothing was received
		}
		switch r.Status {
		case StatusNoContent, StatusSkipped:
			return
		}
		switch data := any(r.Data).(type) {
		case nil:
			return
		case DocumentSource:
			// e.g. a realtime stream: the result holds the source of documents
			for doc, err := range data.Documents() {
				if !yield(doc, err) {
					return
				}
			}
		case jsondoc.Unwrapper:
			doc := data.GetJSONDoc()
			if len(doc.Raw()) == 0 {
				return
			}
			yield(doc, nil)
		case json.RawMessage:
			if len(data) > 0 {
				yield(jsondoc.New(data), nil)
			}
		case []byte:
			if len(data) > 0 {
				yield(jsondoc.New(data), nil)
			}
		default:
			raw, err := json.Marshal(data)
			if err != nil {
				yield(jsondoc.Empty(), err)
				return
			}
			if string(raw) == "null" {
				return
			}
			yield(jsondoc.New(raw), nil)
		}
	}
}
