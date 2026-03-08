package fakeserver

import (
	"net/http"
)

// handleOperations routes /devicecontrol/operations requests.
func (fs *FakeServer) handleOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /devicecontrol/operations/{id}
	id := extractID(path, "/devicecontrol/operations")
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Operations.Get(id)
			if !ok {
				writeNotFound(w, "devicecontrol/operation")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.Operations.Update(id, body)
			if !ok {
				writeNotFound(w, "devicecontrol/operation")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.Operations.Delete(id) {
				writeNotFound(w, "devicecontrol/operation")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /devicecontrol/operations (collection)
	switch r.Method {
	case http.MethodGet:
		items := ReverseItems(FilterItems(r, fs.Operations.List()))
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "operations", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		body = setDefaultField(body, "status", "PENDING")
		_, doc := fs.Operations.Create(body, fs.URL()+"/devicecontrol/operations")
		writeJSON(w, http.StatusCreated, doc)

	case http.MethodDelete:
		// Delete matching operations
		items := FilterItems(r, fs.Operations.List())
		for _, item := range items {
			id := getJSONString(item, "id")
			if id != "" {
				fs.Operations.Delete(id)
			}
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleBulkOperations routes /devicecontrol/bulkoperations requests.
func (fs *FakeServer) handleBulkOperations(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	id := extractID(path, "/devicecontrol/bulkoperations")
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.BulkOperations.Get(id)
			if !ok {
				writeNotFound(w, "devicecontrol/bulkOperation")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.BulkOperations.Update(id, body)
			if !ok {
				writeNotFound(w, "devicecontrol/bulkOperation")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.BulkOperations.Delete(id) {
				writeNotFound(w, "devicecontrol/bulkOperation")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		items := FilterItems(r, fs.BulkOperations.List())
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "bulkOperations", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		body = setDefaultField(body, "status", "ACTIVE")
		body = setDefaultField(body, "generalStatus", "SCHEDULED")
		_, doc := fs.BulkOperations.Create(body, fs.URL()+"/devicecontrol/bulkoperations")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}
