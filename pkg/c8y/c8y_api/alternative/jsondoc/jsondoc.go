package jsondoc

import (
	"encoding/json"
	"iter"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type JSONDoc struct {
	raw []byte
}

func New(raw []byte) JSONDoc {
	return JSONDoc{raw: raw}
	// return JSONDoc{raw: append([]byte(nil), raw...)}
}

func Empty() JSONDoc {
	return JSONDoc{raw: []byte(`{}`)}
}

func (d JSONDoc) Bytes() []byte {
	return append([]byte(nil), d.raw...)
}

// MarshalJSON implements json.Marshaler to allow JSONDoc to be
// marshaled by returning the raw bytes directly
func (d JSONDoc) MarshalJSON() ([]byte, error) {
	return d.raw, nil
}

// UnmarshalJSON implements json.Unmarshaler to allow JSONDoc to be
// unmarshaled by simply assigning the raw bytes rather than parsing
func (d *JSONDoc) UnmarshalJSON(data []byte) error {
	d.raw = append([]byte(nil), data...)
	return nil
}

func (d JSONDoc) Get(path ...string) gjson.Result {
	if len(path) == 0 || path[0] == "" {
		return gjson.ParseBytes(d.raw)
	}
	return gjson.GetBytes(d.raw, path[0])
}

func (d JSONDoc) Exists(path string) bool {
	return d.Get(path).Exists()
}

func (d JSONDoc) GetJSONDoc() JSONDoc {
	return d
}

func (d JSONDoc) Length() int {
	if len(d.raw) == 0 {
		return 0
	}
	res := gjson.ParseBytes(d.raw)
	if res.IsArray() {
		return len(res.Array())
	}
	return 1
}

func (d JSONDoc) Set(path string, value any) (JSONDoc, error) {
	out, err := sjson.SetBytes(d.raw, path, value)
	if err != nil {
		return JSONDoc{}, err
	}
	return JSONDoc{raw: out}, nil
}

func (d JSONDoc) Delete(path string) (JSONDoc, error) {
	out, err := sjson.DeleteBytes(d.raw, path)
	if err != nil {
		return JSONDoc{}, err
	}
	return JSONDoc{raw: out}, nil
}

func (d JSONDoc) Iter() iter.Seq[JSONDoc] {
	root := gjson.ParseBytes(d.raw)

	if root.IsObject() {
		return func(yield func(JSONDoc) bool) {
			yield(JSONDoc{raw: []byte(root.Raw)})
		}
	}

	if root.IsArray() {
		return func(yield func(JSONDoc) bool) {
			root.ForEach(func(_, v gjson.Result) bool {
				return yield(JSONDoc{raw: []byte(v.Raw)})
			})
		}
	}

	// Return empty iterator for invalid JSON
	return func(yield func(JSONDoc) bool) {}
}

// IterBytes returns an iterator that yields items as ByteProvider interface.
// This satisfies the types.CollectionIterator interface.
func (d JSONDoc) IterBytes() iter.Seq[types.ByteProvider] {
	return func(yield func(types.ByteProvider) bool) {
		for doc := range d.Iter() {
			if !yield(doc) {
				return
			}
		}
	}
}

func Decode[T any](d JSONDoc) (*T, error) {
	var out T
	if err := json.Unmarshal(d.raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DecodeIter transforms an iterator of types that embed JSONDoc by JSON unmarshaling to type T.
// Items that fail to unmarshal are skipped.
// Works with both iter.Seq[JSONDoc] and iter.Seq[jsonmodels.X] (any type embedding JSONDoc).
// The input type is inferred from the iterator, you only need to specify the output type.
// Example: jsondoc.DecodeIter[CustomModel](iterator.Items())
// Example: jsondoc.DecodeIter[CustomModel](result.Data.Iter())
func DecodeIter[T any, F Unwrapper](seq iter.Seq[F]) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for doc := range seq {
			v, err := Decode[T](doc.GetJSONDoc())
			if err != nil {
				continue
			}
			if !yield(v) {
				return
			}
		}
	}
}

// DecodeIterErr transforms an iterator of types that embed JSONDoc by JSON unmarshaling to type T.
// Unlike DecodeIter, this yields both the value and any decoding error, allowing callers to handle errors.
// Works with both iter.Seq[JSONDoc] and iter.Seq[jsonmodels.X] (any type embedding JSONDoc).
// The input type is inferred from the iterator, you only need to specify the output type.
// Example: for item, err := range jsondoc.DecodeIterErr[CustomModel](iterator.Items()) { }
func DecodeIterErr[T any, F Unwrapper](seq iter.Seq[F]) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for doc := range seq {
			v, err := Decode[T](doc.GetJSONDoc())
			if !yield(v, err) {
				return
			}
		}
	}
}

// DecodeSeq2 transforms an error-aware iterator (Seq2) of types that embed JSONDoc by JSON unmarshaling to type T.
// This propagates both input sequence errors and decoding errors.
// Use this with iterator.Items() which returns Seq2[T, error].
// Example: for item, err := range jsondoc.DecodeSeq2[CustomModel](iterator.Items()) { }
func DecodeSeq2[T any, F Unwrapper](seq iter.Seq2[F, error]) iter.Seq2[*T, error] {
	return func(yield func(*T, error) bool) {
		for doc, err := range seq {
			if err != nil {
				// Propagate input errors
				if !yield(nil, err) {
					return
				}
				continue
			}
			// Decode and yield (may have decoding error)
			v, err := Decode[T](doc.GetJSONDoc())
			if !yield(v, err) {
				return
			}
		}
	}
}

// Unwrapper is a constraint for types that can provide access to their embedded JSONDoc.
type Unwrapper interface {
	GetJSONDoc() JSONDoc
}

// IterWith transforms an iterator of types that embed JSONDoc using a constructor function.
// Works with both iter.Seq[JSONDoc] and iter.Seq[jsonmodels.X] (any type embedding JSONDoc).
// Example: jsondoc.IterWith(result.Data.Iter(), jsonmodels.NewMicroservice)
// Example: jsondoc.IterWith(iterator.Items(), customConstructor)
func IterWith[F Unwrapper, T any](seq iter.Seq[F], constructor func([]byte) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		for doc := range seq {
			item := constructor(doc.GetJSONDoc().Bytes())
			if !yield(item) {
				return
			}
		}
	}
}

// IterWithErr transforms an iterator of types that embed JSONDoc using a fallible constructor.
// Items that fail to construct are skipped (similar to DecodeIter behavior).
// Works with both iter.Seq[JSONDoc] and iter.Seq[jsonmodels.X] (any type embedding JSONDoc).
// Example: jsondoc.IterWithErr(result.Data.Iter(), parseCustomModel)
// Example: jsondoc.IterWithErr(iterator.Items(), parseCustomModel)
func IterWithErr[F Unwrapper, T any](seq iter.Seq[F], constructor func([]byte) (T, error)) iter.Seq[T] {
	return func(yield func(T) bool) {
		for doc := range seq {
			item, err := constructor(doc.GetJSONDoc().Bytes())
			if err != nil {
				continue // Skip items that fail to construct
			}
			if !yield(item) {
				return
			}
		}
	}
}

func MapIter(
	seq iter.Seq[JSONDoc],
	fn func(JSONDoc) (JSONDoc, bool),
) iter.Seq[JSONDoc] {

	return func(yield func(JSONDoc) bool) {
		for doc := range seq {
			if out, ok := fn(doc); ok {
				if !yield(out) {
					return
				}
			}
		}
	}
}

func IterToChan[T any](seq iter.Seq[T]) <-chan T {
	ch := make(chan T)

	go func() {
		defer close(ch)
		for v := range seq {
			ch <- v
		}
	}()

	return ch
}

type Facade struct {
	JSONDoc
}

// GetJSONDoc returns the embedded JSONDoc. This allows Facade types to satisfy the Unwrapper interface.
func (f Facade) GetJSONDoc() JSONDoc {
	return f.JSONDoc
}
