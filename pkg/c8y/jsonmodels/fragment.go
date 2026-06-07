package jsonmodels

import (
	"encoding/json"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// GetAs decodes the named top-level fragment of a document into T. It is the read-side
// counterpart to writing fragments via model.Fragment.
//
//	pos, err := jsonmodels.GetAs[model.Position](result.Data, "c8y_Position")
//
// It returns model.ErrFragmentNotFound (wrapped) when the key is absent, so callers can
// distinguish "missing" from a decode error.
func GetAs[T any](u jsondoc.Unwrapper, key string) (T, error) {
	var out T
	res := u.GetJSONDoc().Get(key)
	if !res.Exists() {
		return out, model.ErrFragmentNotFound
	}
	if err := json.Unmarshal([]byte(res.Raw), &out); err != nil {
		return out, err
	}
	return out, nil
}

// GetFragment decodes a fragment using the fragment type's own key, making it fully
// symmetric with the write side. T must be a value type whose zero value reports the
// correct FragmentKey() (a constant-returning method); pointer types are not supported.
//
//	pos, err := jsonmodels.GetFragment[model.Position](result.Data)
func GetFragment[T model.Fragment](u jsondoc.Unwrapper) (T, error) {
	var zero T
	return GetAs[T](u, zero.FragmentKey())
}
