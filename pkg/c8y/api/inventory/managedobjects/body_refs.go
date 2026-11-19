package managedobjects

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// ResolveBodyRefs resolves the device references held at the given JSON paths
// of a pre-built request body (e.g. "source.id" of an event), so raw-body
// create calls accept the same references as the typed options: a value that
// is not an id ("dev01", "name:dev01", "ext:c8y_Serial:ABC", "query:...") is
// looked up and replaced with the device id.
//
// Ids, missing paths and non-string values are left untouched, and a body
// that needs no lookup is returned as given (no re-encoding). Otherwise the
// result is a json.RawMessage. Lookups use the resolution context, so they
// run even under dry run or deferred execution.
func (r *DeviceResolver) ResolveBodyRefs(ctx context.Context, body any, paths ...string) (any, error) {
	if r == nil || body == nil || len(paths) == 0 {
		return body, nil
	}
	raw, err := bodyBytes(body)
	if err != nil {
		return body, err
	}
	changed := false
	for _, path := range paths {
		v := gjson.GetBytes(raw, path)
		if v.Type != gjson.String || v.Str == "" || source.IsID(v.Str) {
			continue
		}
		id, err := r.ResolveID(ctxhelpers.ResolutionContext(ctx), DeviceRef(source.ParseRef(v.Str)), nil)
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
