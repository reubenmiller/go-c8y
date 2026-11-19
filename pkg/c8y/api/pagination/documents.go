package pagination

import (
	"iter"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// Documents returns the items of all pages as JSON documents (see
// op.DocumentSource). Items that are not jsondoc.Unwrapper values are skipped.
func (it *Iterator[T]) Documents() iter.Seq2[jsondoc.JSONDoc, error] {
	return func(yield func(jsondoc.JSONDoc, error) bool) {
		for item, err := range it.Items() {
			if err != nil {
				if !yield(jsondoc.Empty(), err) {
					return
				}
				continue
			}
			u, ok := any(item).(jsondoc.Unwrapper)
			if !ok {
				continue
			}
			if !yield(u.GetJSONDoc(), nil) {
				return
			}
		}
	}
}

// PageDocuments returns one document per page: the whole response envelope
// (items, statistics and links), e.g. for a CLI's --raw output.
func (it *Iterator[T]) PageDocuments() iter.Seq2[jsondoc.JSONDoc, error] {
	return it.Pages()
}
