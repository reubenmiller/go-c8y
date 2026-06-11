package output

import (
	"encoding/json"
	"fmt"
	"io"
	"iter"
	"unsafe"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// FromIterator adapts an error-aware iterator of jsondoc-backed items (e.g.
// pagination.Iterator[T].Items()) into a pipeline source.
func FromIterator[T jsondoc.Unwrapper](items iter.Seq2[T, error]) Seq {
	return func(yield func(jsondoc.JSONDoc, error) bool) {
		for item, err := range items {
			if err != nil {
				if !yield(jsondoc.Empty(), err) {
					return
				}
				continue
			}
			if !yield(item.GetJSONDoc(), nil) {
				return
			}
		}
	}
}

// FromBytes yields the elements of the array at the given path within an
// already-buffered response body (e.g. "managedObjects"). An empty path
// treats body itself as the collection. If the value at path is not an
// array, the single value is yielded as one document.
//
// Elements are sliced out of body without copying where possible, so body
// must not be modified while the sequence is in use.
func FromBytes(body []byte, path string) Seq {
	return func(yield func(jsondoc.JSONDoc, error) bool) {
		// View the body as a string without copying: gjson.GetBytes/ParseBytes
		// defensively clone the matched raw value, which for a collection is
		// nearly the whole body.
		doc := unsafe.String(unsafe.SliceData(body), len(body))
		var res gjson.Result
		if path == "" {
			res = gjson.Parse(doc)
		} else {
			res = gjson.Get(doc, path)
		}
		if !res.Exists() {
			yield(jsondoc.Empty(), fmt.Errorf("output: path %q not found in response body", path))
			return
		}
		if !res.IsArray() {
			yield(jsondoc.New(rawBytes(body, res)), nil)
			return
		}
		res.ForEach(func(_, v gjson.Result) bool {
			return yield(jsondoc.New(rawBytes(body, v)), nil)
		})
	}
}

// rawBytes returns the raw bytes of a gjson result, slicing into the original
// document without copying when the result's offset is known.
func rawBytes(body []byte, v gjson.Result) []byte {
	if v.Index > 0 && v.Index+len(v.Raw) <= len(body) {
		return body[v.Index : v.Index+len(v.Raw)]
	}
	return []byte(v.Raw)
}

// FromReader incrementally scans a JSON response body, yielding the elements
// of the array at the given top-level key (e.g. "managedObjects") as they are
// read from the stream. Only one element is held in memory at a time, and
// because the pipeline is pull-based the reader is only consumed as fast as
// the sink accepts items — backpressure propagates to the network connection
// when r is an HTTP response body.
//
// An empty key expects the body itself to be a JSON array. Keys before the
// target array are skipped; once the array has been consumed the remainder of
// the stream is left unread (callers should still close the body).
func FromReader(r io.Reader, key string) Seq {
	return func(yield func(jsondoc.JSONDoc, error) bool) {
		scanReader(json.NewDecoder(r), key, yield)
	}
}

func scanReader(dec *json.Decoder, key string, yield func(jsondoc.JSONDoc, error) bool) {
	if key == "" {
		if err := expectDelim(dec, '['); err != nil {
			yield(jsondoc.Empty(), err)
			return
		}
		yieldArrayElements(dec, yield)
		return
	}

	if err := seekArrayKey(dec, key); err != nil {
		yield(jsondoc.Empty(), err)
		return
	}
	yieldArrayElements(dec, yield)
}

// seekArrayKey advances the decoder past the opening of the array at the
// given top-level key, skipping any preceding keys.
func seekArrayKey(dec *json.Decoder, key string) error {
	if err := expectDelim(dec, '{'); err != nil {
		return err
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		name, ok := tok.(string)
		if !ok {
			return fmt.Errorf("output: expected object key, got %v", tok)
		}
		if name != key {
			if err := skipValue(dec); err != nil {
				return err
			}
			continue
		}
		return expectDelim(dec, '[')
	}
	return fmt.Errorf("output: key %q not found in response body", key)
}

func yieldArrayElements(dec *json.Decoder, yield func(jsondoc.JSONDoc, error) bool) {
	for dec.More() {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			yield(jsondoc.Empty(), err)
			return
		}
		if !yield(jsondoc.New(raw), nil) {
			return
		}
	}
}

func expectDelim(dec *json.Decoder, want json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if d, ok := tok.(json.Delim); !ok || d != want {
		return fmt.Errorf("output: expected %q, got %v", want, tok)
	}
	return nil
}

// skipValue consumes and discards the next JSON value (scalar, object or
// array) from the decoder.
func skipValue(dec *json.Decoder) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	d, ok := tok.(json.Delim)
	if !ok || (d != '{' && d != '[') {
		return nil
	}
	depth := 1
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{', '[':
				depth++
			case '}', ']':
				depth--
			}
		}
	}
	return nil
}
