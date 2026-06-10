package model

import (
	"encoding/json"
	"errors"

	"github.com/reubenmiller/go-c8y/v2/pkg/jsonUtilities"
)

// ErrFragmentNotFound is returned by the read-side fragment helpers when the requested
// fragment key is absent from the document.
var ErrFragmentNotFound = errors.New("fragment not found")

// Fragment is a self-describing custom fragment: a value that knows the top-level JSON
// key it serialises under. Any user-defined type can implement it to gain typed,
// discoverable create/update bodies. The value serialises to its JSON representation
// under FragmentKey() (see Raw for the ad-hoc case).
//
// Example:
//
//	type Position struct {
//	    Lat float64 `json:"lat,omitempty"`
//	    Lng float64 `json:"lng,omitempty"`
//	}
//	func (Position) FragmentKey() string { return "c8y_Position" }
type Fragment interface {
	FragmentKey() string
}

// Raw is the escape hatch for fragments whose shape is not known upfront. Unlike a bare
// any blob it is explicit and named. It marshals to its Value (not to the Raw wrapper),
// so it composes with the same body-merge path as typed fragments.
type Raw struct {
	Key   string
	Value any // struct, map[string]any, or any JSON-serializable value
}

// FragmentKey implements Fragment.
func (r Raw) FragmentKey() string { return r.Key }

// MarshalJSON marshals the underlying Value, so that map[string]any{r.FragmentKey(): r}
// produces {"<key>": <value>} rather than wrapping the Raw struct.
func (r Raw) MarshalJSON() ([]byte, error) { return json.Marshal(r.Value) }

// Frag is sugar for constructing an ad-hoc Raw fragment:
//
//	model.Frag("c8y_Custom", map[string]any{"foo": "bar"})
func Frag(key string, value any) Raw { return Raw{Key: key, Value: value} }

// MergeFragments deep-merges each fragment into body as {fragmentKey: value}, in order
// (later entries win). nil fragments are skipped. It is the shared body-assembly helper
// used by the resource Create/Upsert paths.
func MergeFragments(body []byte, fragments []Fragment) ([]byte, error) {
	for _, fr := range fragments {
		if fr == nil {
			continue
		}
		fragJSON, err := json.Marshal(map[string]any{fr.FragmentKey(): fr})
		if err != nil {
			return nil, err
		}
		body, err = jsonUtilities.MergePatch(body, fragJSON)
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

// ApplyFragments merges fragments into an arbitrary JSON-object body and
// returns the merged body as a map. The body must marshal to a JSON object
// (e.g. a map or a struct). Used by upsert paths to attach annotation
// fragments to a caller-provided body without mutating it.
func ApplyFragments(body any, fragments []Fragment) (map[string]any, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	merged, err := MergeFragments(raw, fragments)
	if err != nil {
		return nil, err
	}

	result := map[string]any{}
	if err := json.Unmarshal(merged, &result); err != nil {
		return nil, err
	}
	return result, nil
}
