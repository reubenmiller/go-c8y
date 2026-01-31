package jsondoc

import (
	"encoding/json"
	"iter"

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

func (d JSONDoc) Get(path string) gjson.Result {
	return gjson.GetBytes(d.raw, path)
}

func (d JSONDoc) Exists(path string) bool {
	return d.Get(path).Exists()
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

func Decode[T any](d JSONDoc) (*T, error) {
	var out T
	if err := json.Unmarshal(d.raw, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func DecodeIter[T any](seq iter.Seq[JSONDoc]) iter.Seq[*T] {
	return func(yield func(*T) bool) {
		for doc := range seq {
			v, err := Decode[T](doc)
			if err != nil {
				continue
			}
			if !yield(v) {
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
