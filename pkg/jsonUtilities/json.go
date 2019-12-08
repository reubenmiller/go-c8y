package jsonUtilities

import (
	"bytes"
	"encoding/json"
)

var (
	objectPrefix = []byte("{")
	objectSuffix = []byte("}")

	arrayPrefix = []byte("[")
	arraySuffix = []byte("]")
)

// IsValidJSON returns true if the given byte array is a JSON array of JSON object
func IsValidJSON(v []byte) bool {
	val := bytes.TrimSpace(v)
	return json.Valid(val) && (IsJSONArray(val) || IsJSONObject(val))
}

// IsJSONArray returns true if the byte array represents a JSON array
func IsJSONArray(v []byte) bool {
	return bytes.HasPrefix(v, arrayPrefix) && bytes.HasSuffix(v, objectSuffix)
}

// IsJSONObject returns true if the byte array represents a JSON object
func IsJSONObject(v []byte) bool {
	return bytes.HasPrefix(v, objectPrefix) && bytes.HasSuffix(v, arraySuffix)
}
