package source

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ResolveBodyRefs resolves the references held at the given JSON paths of a
// pre-built request body, using resolve for each value that is not an id
// (e.g. "dev01", "name:dev01", "ext:c8y_Serial:ABC"). The value is passed to
// resolve after ParseRef, so a bare name arrives as "name:dev01".
//
// Ids, missing paths and non-string values are left untouched, and a body
// that needs no lookup is returned as given (no re-encoding). Otherwise the
// result is a json.RawMessage. resolve is responsible for using a lookup
// context that works under dry run and deferred execution.
func ResolveBodyRefs(ctx context.Context, body any, resolve func(ctx context.Context, ref string) (string, error), paths ...string) (any, error) {
	if body == nil || resolve == nil || len(paths) == 0 {
		return body, nil
	}
	raw, err := bodyBytes(body)
	if err != nil {
		return body, err
	}
	changed := false
	for _, path := range paths {
		v := gjson.GetBytes(raw, path)
		if v.Type != gjson.String || v.Str == "" || IsID(v.Str) {
			continue
		}
		id, err := resolve(ctx, ParseRef(v.Str))
		if err != nil {
			return body, fmt.Errorf("resolve %s %q: %w", path, v.Str, err)
		}
		raw, err = sjson.SetBytes(raw, path, id)
		if err != nil {
			return body, err
		}
		changed = true
	}
	if !changed {
		return body, nil
	}
	return json.RawMessage(raw), nil
}

func bodyBytes(body any) ([]byte, error) {
	switch b := body.(type) {
	case json.RawMessage:
		return append([]byte(nil), b...), nil
	case []byte:
		return append([]byte(nil), b...), nil
	case string:
		return []byte(b), nil
	default:
		return json.Marshal(body)
	}
}
