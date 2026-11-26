package op

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// BodyKind tells the shape of a collection body held by a result (see [Body]).
type BodyKind int

const (
	// BodyNone means the result holds no collection body: a single object, an
	// error, a 204 No Content, a dry run or a deferred result.
	BodyNone BodyKind = iota
	// BodyEnvelope means the body is an object (items, statistics, paging
	// links) from which the items were plucked into Data.
	BodyEnvelope
	// BodyArray means the body is a top-level JSON array, a collection without
	// an envelope (e.g. the tenant usage statistics summary).
	BodyArray
)

// String returns the name of the kind: "none", "envelope" or "array".
func (k BodyKind) String() string {
	switch k {
	case BodyEnvelope:
		return "envelope"
	case BodyArray:
		return "array"
	}
	return "none"
}

// Body is the whole response body of a collection result, as returned by
// [Result.RawBody]. It is a plain value which does not depend on the type
// parameter of the result, so a consumer which only holds a [RawBodySource]
// can read it without reflection.
type Body struct {
	// Raw is the body exactly as received (for BodyEnvelope), or the JSON array
	// (for BodyArray). Nil when Kind is BodyNone. Do not modify it: it shares
	// memory with the result.
	Raw []byte
	// Kind is the shape of the body.
	Kind BodyKind
	// Empty reports that the collection holds no items: an empty JSON array, or
	// an envelope whose items array is empty. Always false for BodyNone.
	Empty bool
}

// Has reports whether there is a body, i.e. Kind is not BodyNone and Raw is not
// empty. It is safe to call on the zero value.
func (b Body) Has() bool {
	return b.Kind != BodyNone && len(b.Raw) > 0
}

// Doc returns the body as a JSON document, or jsondoc.Empty() when there is no
// body.
func (b Body) Doc() jsondoc.JSONDoc {
	if !b.Has() {
		return jsondoc.Empty()
	}
	return jsondoc.New(b.Raw)
}

// RawBodySource is implemented by every op.Result. It lets a consumer, such as
// a CLI printing the whole response (--raw), reach the response body of a
// result without knowing its type parameter.
type RawBodySource interface {
	RawBody() Body
}

// RawBody returns the whole response body of a collection result, which
// [Result.Documents] otherwise splits into one document per item.
//
//   - A collection whose items were plucked from an envelope: the response as
//     received, with Kind BodyEnvelope (the retained Response field). If the
//     response itself is a JSON array, the Kind is BodyArray.
//   - A result whose Data is a JSON array (jsondoc.Unwrapper): that array, with
//     Kind BodyArray.
//   - Anything else (a single object, an error, 204 No Content, a dry run or a
//     deferred result): the zero Body (Kind BodyNone, Raw nil).
//
// The result is safe to use on the zero Result. Raw is not copied.
func (r Result[T]) RawBody() Body {
	// the logic is in a function which is not generic, so that it is compiled
	// once and not for every instantiation of Result (see the package doc)
	return rawBody(r.Err != nil || r.Request != nil || r.IsDeferred(), any(r.Data), r.Response)
}

func rawBody(skip bool, data any, response []byte) Body {
	if skip {
		return Body{}
	}
	var doc jsondoc.JSONDoc
	isArray, empty := false, false
	if u, ok := data.(jsondoc.Unwrapper); ok {
		doc = u.GetJSONDoc()
		if len(doc.Raw()) > 0 {
			if value := doc.Get(); value.IsArray() {
				isArray = true
				empty = len(value.Array()) == 0
			}
		}
	}
	if len(response) > 0 {
		kind := BodyEnvelope
		if jsondoc.New(response).Get().IsArray() {
			kind = BodyArray
		}
		return Body{Raw: response, Kind: kind, Empty: isArray && empty}
	}
	if !isArray {
		return Body{}
	}
	return Body{Raw: doc.Raw(), Kind: BodyArray, Empty: empty}
}
