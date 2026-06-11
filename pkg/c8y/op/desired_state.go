package op

import (
	"encoding/json"
	"reflect"
)

// DesiredStateMatches reports whether every field in the desired body already
// has an equal value on the existing JSON document, comparing after JSON
// normalisation so Go values and raw JSON compare consistently.
//
// The comparison is a subset match: fields present on the existing document
// but absent from desired are ignored (Cumulocity PUT merges top-level
// fragments, and the platform adds bookkeeping fields such as lastUpdated and
// additionParents that callers do not manage). Nested objects are compared
// recursively with the same subset semantics; arrays and scalars are compared
// wholesale.
//
// Upsert implementations use this to skip no-op updates: when the desired
// state already matches, the update is skipped and StatusSkipped is returned,
// so re-applying the same desired state performs no writes.
//
// desired must marshal to a JSON object; existing must be a JSON object
// document. Any marshalling/parsing failure returns false (treat as changed).
func DesiredStateMatches(desired any, existing []byte) bool {
	desiredJSON, err := json.Marshal(desired)
	if err != nil {
		return false
	}

	var desiredValue any
	if err := json.Unmarshal(desiredJSON, &desiredValue); err != nil {
		return false
	}
	desiredMap, ok := desiredValue.(map[string]any)
	if !ok {
		return false
	}

	var existingValue any
	if err := json.Unmarshal(existing, &existingValue); err != nil {
		return false
	}
	existingMap, ok := existingValue.(map[string]any)
	if !ok {
		return false
	}

	return subsetMatch(desiredMap, existingMap)
}

// subsetMatch compares desired against existing: objects are compared key by
// key (extra keys on existing are ignored), everything else with DeepEqual
func subsetMatch(desired, existing any) bool {
	desiredMap, ok := desired.(map[string]any)
	if !ok {
		return reflect.DeepEqual(desired, existing)
	}

	existingMap, ok := existing.(map[string]any)
	if !ok {
		return false
	}

	for key, desiredValue := range desiredMap {
		existingValue, exists := existingMap[key]
		if !exists {
			return false
		}
		if !subsetMatch(desiredValue, existingValue) {
			return false
		}
	}
	return true
}
