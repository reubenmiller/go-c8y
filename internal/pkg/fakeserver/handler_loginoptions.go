package fakeserver

import (
	"encoding/json"
	"net/http"
)

// accessMappingStore returns the (lazily-created) access-mapping store for a login option.
// The fake server processes requests synchronously, so no locking is required (matching
// the other per-parent maps such as ChildDevices).
func (fs *FakeServer) accessMappingStore(typeOrID string) *Store {
	s, ok := fs.LoginOptionAccessMappings[typeOrID]
	if !ok {
		s = NewStore()
		fs.LoginOptionAccessMappings[typeOrID] = s
	}
	return s
}

// inventoryAccessMappingStore returns the inventory-access-mapping store for a login option.
func (fs *FakeServer) inventoryAccessMappingStore(typeOrID string) *Store {
	s, ok := fs.LoginOptionInventoryAccessMappings[typeOrID]
	if !ok {
		s = NewStore()
		fs.LoginOptionInventoryAccessMappings[typeOrID] = s
	}
	return s
}

// handleLoginOptionMappings serves the CRUD surface for a login option's access mappings
// (both accessMappings and inventoryAccessMappings, which are structurally identical).
// resultProperty is the collection field name; itemID is empty for collection requests.
func (fs *FakeServer) handleLoginOptionMappings(w http.ResponseWriter, r *http.Request, store *Store, resultProperty, itemID string) {
	if itemID != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := store.Get(itemID)
			if !ok {
				writeNotFound(w, "accessMapping")
				return
			}
			writeJSON(w, http.StatusOK, doc)
		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := store.Update(itemID, body)
			if !ok {
				writeNotFound(w, "accessMapping")
				return
			}
			writeJSON(w, http.StatusOK, doc)
		case http.MethodDelete:
			if !store.Delete(itemID) {
				writeNotFound(w, "accessMapping")
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
		items := store.List()
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), resultProperty, page))
	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		_, doc := store.Create(body, fs.URL()+r.URL.Path)
		writeJSON(w, http.StatusCreated, doc)
	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleLoginOptionRestrict serves PUT .../{typeOrId}/restrict, returning the updated
// login option (the fake echoes the restriction back).
func (fs *FakeServer) handleLoginOptionRestrict(w http.ResponseWriter, r *http.Request, typeOrID string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
		return
	}
	onlyMgmt := getJSONString(body, "onlyManagementTenantAccess")
	resp, _ := json.Marshal(map[string]any{
		"self":                       fs.URL() + "/tenant/loginOptions/" + typeOrID,
		"type":                       typeOrID,
		"onlyManagementTenantAccess": onlyMgmt == "true",
	})
	writeJSON(w, http.StatusOK, resp)
}
