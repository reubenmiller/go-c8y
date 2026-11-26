package fakeserver

import (
	"encoding/json"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// loggerLevels are the log levels of a Spring Boot actuator.
var loggerLevels = []string{"OFF", "ERROR", "WARN", "INFO", "DEBUG", "TRACE"}

// microserviceExists reports whether an application with the given name (or
// key) exists.
func (fs *FakeServer) microserviceExists(name string) bool {
	for _, app := range fs.Applications.List() {
		if getJSONString(app, "name") == name || getJSONString(app, "key") == name {
			return true
		}
	}
	return false
}

// handleServiceProxy routes /service/{name}/... requests to a microservice.
// Only the Spring Boot actuator loggers endpoints are implemented (the log
// levels the CLI changes with "c8y microservices loglevels"):
//
//   - GET  /service/{name}/loggers
//   - GET  /service/{name}/loggers/{loggerName}
//   - POST /service/{name}/loggers/{loggerName} with {"configuredLevel": "DEBUG"}
//     (null resets the logger)
//
// Like the platform, an unknown microservice answers 404. Every microservice
// has the ROOT logger at INFO; a logger without a configured level inherits
// the effective level of ROOT.
func (fs *FakeServer) handleServiceProxy(w http.ResponseWriter, r *http.Request) {
	segments := extractPathSegments(r.URL.Path, "/service")
	if len(segments) < 2 || segments[1] != "loggers" || len(segments) > 3 {
		writeUnrouted(w, "service")
		return
	}
	name := segments[0]
	if !fs.microserviceExists(name) {
		writeNotFound(w, "application")
		return
	}

	fs.loggersMu.Lock()
	defer fs.loggersMu.Unlock()
	configured := fs.loggers[name]
	if configured == nil {
		configured = map[string]string{"ROOT": "INFO"}
		if fs.loggers == nil {
			fs.loggers = map[string]map[string]string{}
		}
		fs.loggers[name] = configured
	}
	effective := func(logger string) string {
		if level, ok := configured[logger]; ok {
			return level
		}
		// the closest configured parent package, else ROOT
		for parent := logger; strings.Contains(parent, "."); {
			parent = parent[:strings.LastIndex(parent, ".")]
			if level, ok := configured[parent]; ok {
				return level
			}
		}
		return configured["ROOT"]
	}
	loggerDoc := func(logger string) map[string]any {
		doc := map[string]any{"effectiveLevel": effective(logger)}
		if level, ok := configured[logger]; ok {
			doc["configuredLevel"] = level
		} else {
			doc["configuredLevel"] = nil
		}
		return doc
	}

	if len(segments) == 2 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			return
		}
		names := make([]string, 0, len(configured))
		for logger := range configured {
			names = append(names, logger)
		}
		sort.Strings(names)
		loggers := map[string]any{}
		for _, logger := range names {
			loggers[logger] = loggerDoc(logger)
		}
		writeJSON(w, http.StatusOK, marshalJSON(map[string]any{
			"levels":  loggerLevels,
			"loggers": loggers,
			"groups":  map[string]any{},
		}))
		return
	}

	logger := segments[2]
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, marshalJSON(loggerDoc(logger)))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		var payload struct {
			ConfiguredLevel *string `json:"configuredLevel"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", "Invalid body")
			return
		}
		if payload.ConfiguredLevel == nil {
			if logger != "ROOT" {
				delete(configured, logger)
			}
		} else {
			level := strings.ToUpper(*payload.ConfiguredLevel)
			if !slices.Contains(loggerLevels, level) {
				writeError(w, http.StatusBadRequest, "general/badRequest", "Invalid log level "+*payload.ConfiguredLevel)
				return
			}
			configured[logger] = level
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleDatabroker routes /databroker/connectors requests: the connectors of
// the data broker (create, list, get, update with the status, delete).
func (fs *FakeServer) handleDatabroker(w http.ResponseWriter, r *http.Request) {
	segments := extractPathSegments(r.URL.Path, "/databroker/connectors")
	if !strings.HasPrefix(r.URL.Path, "/databroker/connectors") || len(segments) > 1 {
		writeUnrouted(w, "databroker")
		return
	}
	selfBase := fs.URL() + "/databroker/connectors"

	if len(segments) == 1 {
		id := segments[0]
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.DatabrokerConnectors.Get(id)
			if !ok {
				writeNotFound(w, "databroker/connector")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.DatabrokerConnectors.Update(id, body)
			if !ok {
				writeNotFound(w, "databroker/connector")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.DatabrokerConnectors.Delete(id) {
				writeNotFound(w, "databroker/connector")
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
		page := Paginate(r, fs.DatabrokerConnectors.List())
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "connectors", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		if getJSONString(body, "status") == "" {
			body = mergeFields(body, map[string]any{"status": "SUSPENDED"})
		}
		_, doc := fs.DatabrokerConnectors.Create(body, selfBase)
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// storeExternalID registers an external id for a managed object.
func (fs *FakeServer) storeExternalID(moID, extType, extID string) json.RawMessage {
	doc := marshalJSON(map[string]any{
		"type":       extType,
		"externalId": extID,
		"managedObject": map[string]any{
			"id":   moID,
			"self": fs.URL() + "/inventory/managedObjects/" + moID,
		},
		"self": fs.URL() + "/identity/externalIds/" + extType + "/" + extID,
	})
	fs.ExternalIDs.CreateWithID(extType+":"+extID, doc)
	return doc
}

// handleInventoryTemplates routes /inventory/templates requests: the binding
// of a SmartREST 1.0 template X-Id to a managed object,
// POST /inventory/templates/globalId/{id}/externalId.
func (fs *FakeServer) handleInventoryTemplates(w http.ResponseWriter, r *http.Request) {
	segments := extractPathSegments(r.URL.Path, "/inventory/templates")
	if len(segments) != 3 || segments[0] != "globalId" || segments[2] != "externalId" {
		writeUnrouted(w, "inventory")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	moID := segments[1]
	if _, ok := fs.ManagedObjects.Get(moID); !ok {
		writeNotFound(w, "inventory/managedObject")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
		return
	}
	extType, extID := getJSONString(body, "type"), getJSONString(body, "externalId")
	if extID == "" {
		writeError(w, http.StatusBadRequest, "general/badRequest", "Missing externalId")
		return
	}
	if extType == "" {
		extType = "c8y_SmartRestDeviceIdentifier"
	}
	writeJSON(w, http.StatusCreated, fs.storeExternalID(moID, extType, extID))
}

// handleIdentitySearch handles POST /identity/search: the bulk lookup of
// external ids ({"externalIds": [{"type", "externalId"}]}). Unknown ids are
// left out of the result.
func (fs *FakeServer) handleIdentitySearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
		return
	}
	var payload struct {
		ExternalIDs []struct {
			Type       string `json:"type"`
			ExternalID string `json:"externalId"`
		} `json:"externalIds"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", "Invalid body")
		return
	}
	found := []json.RawMessage{}
	for _, ext := range payload.ExternalIDs {
		if doc, ok := fs.ExternalIDs.Get(ext.Type + ":" + ext.ExternalID); ok {
			found = append(found, doc)
		}
	}
	writeJSON(w, http.StatusOK, marshalJSON(map[string]any{"externalIds": found}))
}

// handleLogout handles POST /user/logout and POST /user/logout/{tenant}/allUsers.
// The fake server has no sessions or tokens to invalidate, so it only
// acknowledges the request.
func (fs *FakeServer) handleLogout(w http.ResponseWriter, r *http.Request, segments []string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	if len(segments) == 1 || (len(segments) == 3 && segments[2] == "allUsers") {
		writeNoContent(w)
		return
	}
	writeUnrouted(w, "user")
}

// handlePasswordReset handles PUT /user/passwordReset, the token based reset
// of a forced password change. A reset token is only issued by the platform's
// login flow, which the fake server does not have, so every token is rejected.
func (fs *FakeServer) handlePasswordReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	body, _ := readBody(r)
	if getJSONString(body, "token") == "" || getJSONString(body, "newPassword") == "" {
		writeError(w, http.StatusBadRequest, "general/badRequest", "Missing token or newPassword")
		return
	}
	writeError(w, http.StatusUnauthorized, "security/Unauthorized", "Invalid password reset token") //nolint:misspell
}

// inventoryAssignment helpers ------------------------------------------------

// handleUserInventoryRoleAssignments handles
//
//   - GET/POST /user/{tenant}/users/{user}/roles/inventory
//   - GET/PUT/DELETE /user/{tenant}/users/{user}/roles/inventory/{id}
//
// An assignment gives a user inventory roles on a managed object (a device or
// group). The assignments are stored with the user name.
func (fs *FakeServer) handleUserInventoryRoleAssignments(w http.ResponseWriter, r *http.Request, userID string, rest []string) {
	selfBase := fs.URL() + strings.TrimSuffix(r.URL.Path, "/")
	if len(rest) > 0 {
		selfBase = strings.TrimSuffix(selfBase, "/"+rest[0])
	}
	own := func(doc json.RawMessage) bool {
		return getJSONString(doc, "__user") == userID
	}
	// clean removes the bookkeeping field
	clean := func(doc json.RawMessage) json.RawMessage {
		var m map[string]json.RawMessage
		if json.Unmarshal(doc, &m) != nil {
			return doc
		}
		delete(m, "__user")
		out, _ := json.Marshal(m)
		return out
	}
	resolveRoles := func(body json.RawMessage) []any {
		var payload struct {
			Roles []json.RawMessage `json:"roles"`
		}
		_ = json.Unmarshal(body, &payload)
		roles := []any{}
		for _, ref := range payload.Roles {
			id := getJSONString(ref, "id")
			if doc, ok := fs.InventoryRoles.Get(id); ok {
				roles = append(roles, json.RawMessage(doc))
				continue
			}
			roles = append(roles, ref)
		}
		return roles
	}

	if len(rest) > 1 {
		writeUnrouted(w, "user")
		return
	}

	if len(rest) == 1 {
		id := rest[0]
		doc, ok := fs.InventoryAssignments.Get(id)
		if !ok || !own(doc) {
			writeNotFound(w, "user/inventoryAssignment")
			return
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, clean(doc))

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			updated, _ := fs.InventoryAssignments.Update(id, marshalJSON(map[string]any{"roles": resolveRoles(body)}))
			writeJSON(w, http.StatusOK, clean(updated))

		case http.MethodDelete:
			fs.InventoryAssignments.Delete(id)
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		items := []json.RawMessage{}
		for _, doc := range fs.InventoryAssignments.List() {
			if !own(doc) {
				continue
			}
			if mo := r.URL.Query().Get("managedObject"); mo != "" && getJSONString(doc, "managedObject") != mo {
				continue
			}
			items = append(items, clean(doc))
		}
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "inventoryAssignments", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		mo := getJSONString(body, "managedObject")
		if mo == "" {
			writeError(w, http.StatusUnprocessableEntity, "general/badRequest", "managedObject is required")
			return
		}
		id, _ := fs.InventoryAssignments.Create(marshalJSON(map[string]any{
			"managedObject": mo,
			"roles":         resolveRoles(body),
			"__user":        userID,
		}), selfBase)
		doc, _ := fs.InventoryAssignments.Get(id)
		if n, err := strconv.ParseInt(id, 10, 64); err == nil {
			doc, _ = fs.InventoryAssignments.Update(id, marshalJSON(map[string]any{"id": n}))
		}
		writeJSON(w, http.StatusCreated, clean(doc))

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleDevicePermissionOwners handles GET/PUT /user/devicePermissions/{id}:
// the permissions per user on one managed object.
func (fs *FakeServer) handleDevicePermissionOwners(w http.ResponseWriter, r *http.Request, moID string) {
	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.DevicePermissionOwners.Get(moID)
		if !ok {
			doc = marshalJSON(map[string]any{"users": []any{}, "groups": []any{}})
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		fs.DevicePermissionOwners.CreateWithID(moID, body)
		w.WriteHeader(http.StatusOK) // the platform answers 200 without a body

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}
