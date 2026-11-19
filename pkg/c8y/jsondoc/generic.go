package jsondoc

import (
	"encoding/json"
	"errors"
)

// ErrKeyNotFound is returned by GetAs and Fragment when the key is absent.
var ErrKeyNotFound = errors.New("jsondoc: key not found")

// Decode decodes the whole document into U, for example a user-defined struct.
// It is available on every jsonmodels type through embedding:
//
//	device, err := mo.Decode[MyDevice]()
func (d JSONDoc) Decode[U any]() (U, error) {
	var out U
	err := json.Unmarshal(d.raw, &out)
	return out, err
}

// GetAs decodes the value at key (a gjson path, e.g. "c8y_Battery.level") into
// U. It returns ErrKeyNotFound (also matched by model.ErrFragmentNotFound) when
// the key is absent, so "missing" can be told apart from a decode error.
//
//	pos, err := mo.GetAs[model.Position]("c8y_Position")
func (d JSONDoc) GetAs[U any](key string) (U, error) {
	var out U
	res := d.Get(key)
	if !res.Exists() {
		return out, ErrKeyNotFound
	}
	err := json.Unmarshal([]byte(res.Raw), &out)
	return out, err
}

// Fragment decodes a custom fragment using the fragment type's own key, the
// read-side counterpart of writing fragments with model.Fragment. U must be a
// value type whose zero value returns its key from FragmentKey().
//
//	pos, err := mo.Fragment[model.Position]()
func (d JSONDoc) Fragment[U interface{ FragmentKey() string }]() (U, error) {
	var zero U
	return d.GetAs[U](zero.FragmentKey())
}
