package jsonUtilities

import (
	"encoding/json"
	"fmt"
)

// MergePatch performs a deep merge of two JSON objects following RFC 7386 (JSON Merge Patch) rules:
// - If both values are objects, merge them recursively
// - If the patch value is null, remove the key from the result
// - Otherwise, the patch value replaces the base value
//
// Both base and patch should be valid JSON objects (not arrays or primitives).
// Returns the merged JSON bytes.
func MergePatch(base, patch []byte) ([]byte, error) {
	if len(base) == 0 {
		base = []byte("{}")
	}
	if len(patch) == 0 {
		return base, nil
	}

	var baseMap map[string]interface{}
	if err := json.Unmarshal(base, &baseMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal base JSON: %w", err)
	}

	var patchMap map[string]interface{}
	if err := json.Unmarshal(patch, &patchMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal patch JSON: %w", err)
	}

	merged := mergeMaps(baseMap, patchMap)

	result, err := json.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged JSON: %w", err)
	}

	return result, nil
}

// mergeMaps recursively merges two maps following JSON Merge Patch rules
func mergeMaps(base, patch map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy all base values
	for k, v := range base {
		result[k] = v
	}

	// Apply patch
	for k, patchValue := range patch {
		// If patch value is nil, remove the key
		if patchValue == nil {
			delete(result, k)
			continue
		}

		// If both are maps, merge recursively
		if patchMap, ok := patchValue.(map[string]interface{}); ok {
			if baseMap, ok := result[k].(map[string]interface{}); ok {
				result[k] = mergeMaps(baseMap, patchMap)
				continue
			}
		}

		// Otherwise, patch value replaces base value
		result[k] = patchValue
	}

	return result
}
