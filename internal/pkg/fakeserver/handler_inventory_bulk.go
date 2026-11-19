package fakeserver

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// maxBulkItems is the platform limit for bulk managed object requests.
const maxBulkItems = 50

type externalIDRef struct {
	Type       string `json:"type"`
	ExternalID string `json:"externalId"`
}

// splitExternalIDs returns the managed object body without its "externalIds"
// field, and the external IDs it listed.
func splitExternalIDs(item json.RawMessage) (json.RawMessage, []externalIDRef) {
	var m map[string]any
	if err := json.Unmarshal(item, &m); err != nil {
		return item, nil
	}
	raw, ok := m["externalIds"]
	if !ok {
		return item, nil
	}
	delete(m, "externalIds")
	var ids []externalIDRef
	if b, err := json.Marshal(raw); err == nil {
		_ = json.Unmarshal(b, &ids)
	}
	stripped, err := json.Marshal(m)
	if err != nil {
		return item, ids
	}
	return stripped, ids
}

func (fs *FakeServer) externalIDBound(id externalIDRef) bool {
	_, ok := fs.ExternalIDs.Get(id.Type + ":" + id.ExternalID)
	return ok
}

func (fs *FakeServer) bindExternalIDs(moID string, ids []externalIDRef) {
	for _, id := range ids {
		doc := marshalJSON(map[string]any{
			"type":       id.Type,
			"externalId": id.ExternalID,
			"managedObject": map[string]any{
				"id":   moID,
				"self": fs.URL() + "/inventory/managedObjects/" + moID,
			},
			"self": fs.URL() + "/identity/externalIds/" + id.Type + "/" + id.ExternalID,
		})
		fs.ExternalIDs.CreateWithID(id.Type+":"+id.ExternalID, doc)
	}
}

// writeBulkError writes a Cumulocity bulk request error. failedIndex < 0 omits
// the bulkRequest object (error about the whole request).
func writeBulkError(w http.ResponseWriter, status int, errorType, message string, failedIndex int) {
	body := map[string]any{
		"error":   errorType,
		"message": message,
		"info":    "https://www.cumulocity.com/guides/reference-guide/#error_reporting",
	}
	if failedIndex >= 0 {
		body["bulkRequest"] = map[string]any{"failedIndex": failedIndex}
	}
	writeJSON(w, status, marshalJSON(body))
}

// createManagedObject creates one managed object and binds its external IDs.
// It returns false (and writes a 409) when an external ID is already bound.
func (fs *FakeServer) createManagedObject(w http.ResponseWriter, item json.RawMessage) (json.RawMessage, bool) {
	body, ids := splitExternalIDs(item)
	for _, id := range ids {
		if fs.externalIDBound(id) {
			writeError(w, http.StatusConflict, "identity/Duplicate",
				fmt.Sprintf("External ID %s/%s is already bound to another managed object", id.Type, id.ExternalID))
			return nil, false
		}
	}
	body = setDefaultField(body, "owner", "admin")
	id, doc := fs.ManagedObjects.Create(body, fs.URL()+"/inventory/managedObjects")
	fs.bindExternalIDs(id, ids)
	return doc, true
}

// validateBulkCreate checks a bulk create the way the platform does: all items
// and external IDs are checked before anything is created.
func (fs *FakeServer) validateBulkCreate(w http.ResponseWriter, items []json.RawMessage) bool {
	if len(items) == 0 || len(items) > maxBulkItems {
		writeBulkError(w, http.StatusUnprocessableEntity, "undefined/validationError",
			fmt.Sprintf("A bulk request must contain 1 to %d managed objects", maxBulkItems), -1)
		return false
	}
	seen := map[string]bool{}
	for i, item := range items {
		_, ids := splitExternalIDs(item)
		for _, id := range ids {
			key := id.Type + ":" + id.ExternalID
			if seen[key] {
				writeBulkError(w, http.StatusUnprocessableEntity, "undefined/validationError",
					fmt.Sprintf("External ID %s/%s occurs more than once", id.Type, id.ExternalID), i)
				return false
			}
			seen[key] = true
			if fs.externalIDBound(id) {
				writeBulkError(w, http.StatusConflict, "identity/Duplicate",
					fmt.Sprintf("External ID %s/%s is already bound to another managed object", id.Type, id.ExternalID), i)
				return false
			}
		}
	}
	return true
}

// validateBulkUpdate checks that every managed object of a bulk update exists.
func (fs *FakeServer) validateBulkUpdate(w http.ResponseWriter, items []json.RawMessage) bool {
	if len(items) == 0 || len(items) > maxBulkItems {
		writeBulkError(w, http.StatusUnprocessableEntity, "undefined/validationError",
			fmt.Sprintf("A bulk request must contain 1 to %d managed objects", maxBulkItems), -1)
		return false
	}
	for i, item := range items {
		id := getJSONString(item, "id")
		if _, ok := fs.ManagedObjects.Get(id); id == "" || !ok {
			writeBulkError(w, http.StatusNotFound, "inventory/Not Found",
				fmt.Sprintf("Managed object %q not found", id), i)
			return false
		}
	}
	return true
}
