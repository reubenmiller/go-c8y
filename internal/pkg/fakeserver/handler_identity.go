package fakeserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleIdentity routes /identity/ requests.
//
// Endpoints:
//   - POST /identity/globalIds/{id}/externalIds
//   - GET  /identity/globalIds/{id}/externalIds
//   - GET  /identity/externalIds/{type}/{value}
//   - DELETE /identity/externalIds/{type}/{value}
func (fs *FakeServer) handleIdentity(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /identity/externalIds/{type}/{value}
	if strings.HasPrefix(path, "/identity/externalIds/") {
		rest := strings.TrimPrefix(path, "/identity/externalIds/")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			writeError(w, http.StatusBadRequest, "identity/badRequest", "Expected /identity/externalIds/{type}/{value}")
			return
		}
		extType := parts[0]
		extValue := parts[1]
		storeKey := extType + ":" + extValue

		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.ExternalIDs.Get(storeKey)
			if !ok {
				writeNotFound(w, "identity/externalId")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.ExternalIDs.Delete(storeKey) {
				writeNotFound(w, "identity/externalId")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /identity/globalIds/{id}/externalIds
	if strings.HasPrefix(path, "/identity/globalIds/") {
		segments := extractPathSegments(path, "/identity/globalIds")
		if len(segments) < 2 || segments[1] != "externalIds" {
			writeError(w, http.StatusBadRequest, "identity/badRequest", "Expected /identity/globalIds/{id}/externalIds")
			return
		}
		moID := segments[0]

		switch r.Method {
		case http.MethodGet:
			// List all external IDs for this managed object
			allIDs := fs.ExternalIDs.List()
			var matching []json.RawMessage
			for _, doc := range allIDs {
				if getJSONPath(doc, "managedObject.id") == moID {
					matching = append(matching, doc)
				}
			}
			if matching == nil {
				matching = []json.RawMessage{}
			}
			page := Paginate(r, matching)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "externalIds", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}

			extType := getJSONString(body, "type")
			extID := getJSONString(body, "externalId")
			if extType == "" || extID == "" {
				writeError(w, http.StatusBadRequest, "identity/badRequest", "Missing type or externalId")
				return
			}

			storeKey := extType + ":" + extID

			doc := marshalJSON(map[string]any{
				"type":       extType,
				"externalId": extID,
				"managedObject": map[string]any{
					"id":   moID,
					"self": fs.URL() + "/inventory/managedObjects/" + moID,
				},
				"self": fs.URL() + "/identity/externalIds/" + extType + "/" + extID,
			})

			fs.ExternalIDs.CreateWithID(storeKey, doc)
			writeJSON(w, http.StatusCreated, doc)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	writeNotFound(w, "identity")
}
