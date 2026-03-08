package fakeserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleAlarms routes /alarm/alarms requests.
func (fs *FakeServer) handleAlarms(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /alarm/alarms/count
	if strings.HasSuffix(path, "/count") {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			return
		}
		items := FilterItems(r, fs.Alarms.List())
		writeCount(w, len(items))
		return
	}

	// /alarm/alarms/{id}
	id := extractID(path, "/alarm/alarms")
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Alarms.Get(id)
			if !ok {
				writeNotFound(w, "alarm")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.Alarms.Update(id, body)
			if !ok {
				writeNotFound(w, "alarm")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.Alarms.Delete(id) {
				writeNotFound(w, "alarm")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /alarm/alarms (collection)
	switch r.Method {
	case http.MethodGet:
		items := ReverseItems(FilterItems(r, fs.Alarms.List()))
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "alarms", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		// Set defaults for alarm fields
		body = setDefaultField(body, "status", "ACTIVE")
		body = enrichSource(body, fs.URL())
		_, doc := fs.Alarms.Create(body, fs.URL()+"/alarm/alarms")
		writeJSON(w, http.StatusCreated, doc)

	case http.MethodPut:
		// Bulk update alarms — Cumulocity accepts PUT on the collection to update matching alarms
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		items := FilterItems(r, fs.Alarms.List())
		for _, item := range items {
			id := getJSONString(item, "id")
			if id != "" {
				fs.Alarms.Update(id, body)
			}
		}
		w.WriteHeader(http.StatusOK)

	case http.MethodDelete:
		// Delete matching alarms
		items := FilterItems(r, fs.Alarms.List())
		for _, item := range items {
			id := getJSONString(item, "id")
			if id != "" {
				fs.Alarms.Delete(id)
			}
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// setDefaultField sets a field on a JSON document only if it's not already present.
func setDefaultField(doc json.RawMessage, key, value string) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(doc, &m); err != nil {
		return doc
	}
	if _, ok := m[key]; !ok {
		m[key], _ = json.Marshal(value)
		out, _ := json.Marshal(m)
		return out
	}
	return doc
}
