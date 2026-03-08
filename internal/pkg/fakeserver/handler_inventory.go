package fakeserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleManagedObjects routes /inventory/managedObjects requests.
func (fs *FakeServer) handleManagedObjects(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Extract segments after /inventory/managedObjects/
	segments := extractPathSegments(path, "/inventory/managedObjects")

	// /inventory/managedObjects (collection)
	if len(segments) == 0 {
		fs.handleManagedObjectCollection(w, r)
		return
	}

	moID := segments[0]

	// /inventory/managedObjects/{id}/childDevices
	if len(segments) >= 2 {
		switch segments[1] {
		case "childDevices":
			fs.handleChildRelationship(w, r, moID, "childDevices", fs.ChildDevices)
			return
		case "childAssets":
			fs.handleChildRelationship(w, r, moID, "childAssets", fs.ChildAssets)
			return
		case "childAdditions":
			fs.handleChildRelationship(w, r, moID, "childAdditions", fs.ChildAdditions)
			return
		case "supportedMeasurements":
			fs.handleSupportedMeasurements(w, r, moID)
			return
		case "supportedSeries":
			fs.handleSupportedSeries(w, r, moID)
			return
		case "availability":
			fs.handleAvailability(w, r, moID)
			return
		case "user":
			fs.handleMOUser(w, r, moID)
			return
		}
	}

	// /inventory/managedObjects/{id}
	fs.handleManagedObjectSingle(w, r, moID)
}

func (fs *FakeServer) handleManagedObjectCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items := fs.filterManagedObjects(r)
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "managedObjects", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		body = setDefaultField(body, "owner", "admin")
		_, doc := fs.ManagedObjects.Create(body, fs.URL()+"/inventory/managedObjects")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleManagedObjectSingle(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.ManagedObjects.Get(id)
		if !ok {
			writeNotFound(w, "inventory/managedObject")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, ok := fs.ManagedObjects.Update(id, body)
		if !ok {
			writeNotFound(w, "inventory/managedObject")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodDelete:
		// Handle cascade delete
		cascade := r.URL.Query().Get("cascade") == "true" || r.URL.Query().Get("forceCascade") == "true"
		if cascade {
			fs.cascadeDelete(id)
		}
		if !fs.ManagedObjects.Delete(id) {
			writeNotFound(w, "inventory/managedObject")
			return
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// cascadeDelete removes child devices, assets, and additions recursively.
func (fs *FakeServer) cascadeDelete(id string) {
	fs.ManagedObjects.mu.Lock()
	childDevices := fs.ChildDevices[id]
	childAssets := fs.ChildAssets[id]
	childAdditions := fs.ChildAdditions[id]
	delete(fs.ChildDevices, id)
	delete(fs.ChildAssets, id)
	delete(fs.ChildAdditions, id)
	fs.ManagedObjects.mu.Unlock()

	for _, childID := range childDevices {
		fs.cascadeDelete(childID)
		fs.ManagedObjects.Delete(childID)
	}
	for _, childID := range childAssets {
		fs.cascadeDelete(childID)
		fs.ManagedObjects.Delete(childID)
	}
	for _, childID := range childAdditions {
		fs.cascadeDelete(childID)
		fs.ManagedObjects.Delete(childID)
	}
}

// handleChildRelationship handles child device/asset/addition endpoints.
func (fs *FakeServer) handleChildRelationship(w http.ResponseWriter, r *http.Request, parentID, relType string, relMap map[string][]string) {
	if _, ok := fs.ManagedObjects.Get(parentID); !ok {
		writeNotFound(w, "inventory/managedObject")
		return
	}

	switch r.Method {
	case http.MethodGet:
		fs.ManagedObjects.mu.RLock()
		childIDs := relMap[parentID]
		fs.ManagedObjects.mu.RUnlock()

		var refs []json.RawMessage
		for _, childID := range childIDs {
			// If a query/filter is specified, look up the actual MO and filter on it
			if q := r.URL.Query().Get("query"); q != "" {
				doc, ok := fs.ManagedObjects.Get(childID)
				if !ok {
					continue
				}
				filtered := applyCQLFilter([]json.RawMessage{doc}, q)
				if len(filtered) == 0 {
					continue
				}
			}

			ref := marshalJSON(map[string]any{
				"managedObject": map[string]any{
					"id":   childID,
					"self": fs.URL() + "/inventory/managedObjects/" + childID,
				},
				"self": fs.URL() + "/inventory/managedObjects/" + parentID + "/" + relType + "/" + childID,
			})
			refs = append(refs, ref)
		}
		if refs == nil {
			refs = []json.RawMessage{}
		}

		page := Paginate(r, refs)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "references", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}

		// Body can be {"managedObject": {"id": "123"}} or just {"id": "123"}
		childID := getNestedID(body)

		// If no ID found, this is a create-and-add request (e.g., Content-Type: application/vnd.com.nsn.cumulocity.managedobject+json)
		if childID == "" {
			// Check if this looks like a create request (has fields other than id/managedObject)
			ct := r.Header.Get("Content-Type")
			if strings.Contains(ct, "managedObject") || getJSONString(body, "name") != "" || getJSONString(body, "type") != "" {
				// Create a new managed object and add it as a child
				newID, newDoc := fs.ManagedObjects.Create(body, fs.URL()+"/inventory/managedObjects")
				childID = newID

				fs.ManagedObjects.mu.Lock()
				relMap[parentID] = append(relMap[parentID], childID)
				fs.ManagedObjects.mu.Unlock()

				writeJSON(w, http.StatusCreated, newDoc)
				return
			}
			writeError(w, http.StatusBadRequest, "general/badRequest", "Missing managedObject.id or id")
			return
		}

		fs.ManagedObjects.mu.Lock()
		relMap[parentID] = append(relMap[parentID], childID)
		fs.ManagedObjects.mu.Unlock()

		ref := marshalJSON(map[string]any{
			"managedObject": map[string]any{
				"id":   childID,
				"self": fs.URL() + "/inventory/managedObjects/" + childID,
			},
			"self": fs.URL() + "/inventory/managedObjects/" + parentID + "/" + relType + "/" + childID,
		})
		writeJSON(w, http.StatusCreated, ref)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleSupportedMeasurements(w http.ResponseWriter, r *http.Request, _ string) {
	resp := marshalJSON(map[string]any{
		"c8y_SupportedMeasurements": []string{},
	})
	writeJSON(w, http.StatusOK, resp)
}

func (fs *FakeServer) handleSupportedSeries(w http.ResponseWriter, r *http.Request, _ string) {
	resp := marshalJSON(map[string]any{
		"c8y_SupportedSeries": []string{},
	})
	writeJSON(w, http.StatusOK, resp)
}

func (fs *FakeServer) handleAvailability(w http.ResponseWriter, r *http.Request, moID string) {
	resp := marshalJSON(map[string]any{
		"id":     moID,
		"status": "AVAILABLE",
		"self":   fs.URL() + "/inventory/managedObjects/" + moID + "/availability",
	})
	writeJSON(w, http.StatusOK, resp)
}

func (fs *FakeServer) handleMOUser(w http.ResponseWriter, r *http.Request, moID string) {
	switch r.Method {
	case http.MethodGet:
		resp := marshalJSON(map[string]any{
			"id":       moID,
			"userName": "device_" + moID,
			"self":     fs.URL() + "/inventory/managedObjects/" + moID + "/user",
			"enabled":  true,
		})
		writeJSON(w, http.StatusOK, resp)
	case http.MethodPut:
		body, _ := readBody(r)
		writeJSON(w, http.StatusOK, body)
	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// filterManagedObjects applies inventory-specific query parameter filtering.
func (fs *FakeServer) filterManagedObjects(r *http.Request) []json.RawMessage {
	items := fs.ManagedObjects.List()

	// Standard filters
	items = FilterItems(r, items)

	// Inventory-specific: query (CQL), ids, text
	q := r.URL.Query()

	if ids := q.Get("ids"); ids != "" {
		idSet := make(map[string]struct{})
		for _, id := range strings.Split(ids, ",") {
			idSet[strings.TrimSpace(id)] = struct{}{}
		}
		var filtered []json.RawMessage
		for _, item := range items {
			if _, ok := idSet[getJSONString(item, "id")]; ok {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}

	if queryStr := q.Get("query"); queryStr != "" {
		items = applyCQLFilter(items, queryStr)
	}

	return items
}

// applyCQLFilter applies a very simplified CQL (Cumulocity Query Language) filter.
// Supports: has(fragment), type eq 'value', name eq 'value', and AND/OR combinations.
// Strips "order by ...", "$orderby=..." clauses before filtering.
// Handles $filter=(...) format used by inventory queries.
func applyCQLFilter(items []json.RawMessage, query string) []json.RawMessage {
	query = strings.TrimSpace(query)
	if query == "" {
		return items
	}

	// Handle "$filter=(...) $orderby=..." format
	if strings.HasPrefix(query, "$filter=") {
		query = strings.TrimPrefix(query, "$filter=")
		// Strip $orderby
		if idx := strings.Index(query, " $orderby="); idx >= 0 {
			query = strings.TrimSpace(query[:idx])
		}
		// Remove wrapping parentheses
		query = strings.TrimSpace(query)
		if strings.HasPrefix(query, "(") && strings.HasSuffix(query, ")") {
			query = query[1 : len(query)-1]
		}
		return applyCQLFilter(items, strings.TrimSpace(query))
	}

	// Strip "order by ..." suffix (case-insensitive)
	if idx := strings.Index(strings.ToLower(query), " order by "); idx >= 0 {
		query = strings.TrimSpace(query[:idx])
	}
	if query == "" {
		return items
	}

	// Handle "expr1 and expr2" (simple AND)
	if idx := caseInsensitiveIndex(query, " and "); idx >= 0 {
		left := strings.TrimSpace(query[:idx])
		right := strings.TrimSpace(query[idx+5:])
		items = applyCQLFilter(items, left)
		return applyCQLFilter(items, right)
	}

	// Handle "has(fragment)"
	if strings.HasPrefix(query, "has(") && strings.HasSuffix(query, ")") {
		fragment := strings.TrimPrefix(query, "has(")
		fragment = strings.TrimSuffix(fragment, ")")
		var filtered []json.RawMessage
		for _, item := range items {
			if hasField(fragment)(item) {
				filtered = append(filtered, item)
			}
		}
		return filtered
	}

	// Strip wrapping parentheses (e.g., "(name eq 'foo')")
	if strings.HasPrefix(query, "(") && strings.HasSuffix(query, ")") {
		inner := query[1 : len(query)-1]
		return applyCQLFilter(items, strings.TrimSpace(inner))
	}

	// Handle "field eq 'value'" – supports trailing wildcard (e.g. name eq 'foo*')
	if strings.Contains(query, " eq ") {
		parts := strings.SplitN(query, " eq ", 2)
		if len(parts) == 2 {
			field := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "'\"")
			var filtered []json.RawMessage
			wildcard := strings.HasSuffix(value, "*")
			prefix := strings.TrimSuffix(value, "*")
			for _, item := range items {
				actual := getJSONPath(item, field)
				if wildcard {
					if strings.HasPrefix(actual, prefix) {
						filtered = append(filtered, item)
					}
				} else if actual == value {
					filtered = append(filtered, item)
				}
			}
			return filtered
		}
	}

	// Fallback: no filtering
	return items
}

// getNestedID extracts an ID from {"managedObject": {"id": "X"}} or {"id": "X"}.
func getNestedID(doc json.RawMessage) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(doc, &m); err != nil {
		return ""
	}
	// Try {"managedObject": {"id": "X"}}
	if moRaw, ok := m["managedObject"]; ok {
		var mo map[string]json.RawMessage
		if err := json.Unmarshal(moRaw, &mo); err == nil {
			if idRaw, ok := mo["id"]; ok {
				return strings.Trim(string(idRaw), `"`)
			}
		}
	}
	// Try {"id": "X"}
	if idRaw, ok := m["id"]; ok {
		return strings.Trim(string(idRaw), `"`)
	}
	return ""
}

// caseInsensitiveIndex finds the first occurrence of substr in s, case-insensitive.
func caseInsensitiveIndex(s, substr string) int {
	return strings.Index(strings.ToLower(s), strings.ToLower(substr))
}
