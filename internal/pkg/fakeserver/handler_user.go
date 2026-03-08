package fakeserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleUsers routes /user/ requests.
func (fs *FakeServer) handleUsers(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /user/currentUser
	if strings.HasPrefix(path, "/user/currentUser") {
		fs.handleCurrentUser(w, r)
		return
	}

	// /user/roles
	if strings.HasPrefix(path, "/user/roles") {
		fs.handleUserRolesGlobal(w, r)
		return
	}

	// /user/inventoryroles
	if strings.HasPrefix(path, "/user/inventoryroles") {
		fs.handleInventoryRoles(w, r)
		return
	}

	// /user/{tenantId}/users, /user/{tenantId}/groups, etc.
	segments := extractPathSegments(path, "/user")
	if len(segments) < 2 {
		writeNotFound(w, "user")
		return
	}

	// tenantID := segments[0]  // e.g., "t12345"
	resource := segments[1] // e.g., "users", "groups"

	switch resource {
	case "users":
		fs.handleTenantUsers(w, r, segments)
	case "groups":
		fs.handleUserGroups(w, r, segments)
	case "groupByName":
		fs.handleGroupByName(w, r, segments)
	case "userByName":
		fs.handleUserByName(w, r, segments)
	default:
		writeNotFound(w, "user")
	}
}

func (fs *FakeServer) handleCurrentUser(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /user/currentUser/password
	if strings.HasSuffix(path, "/password") {
		if r.Method == http.MethodPut {
			writeNoContent(w)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}

	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.Users.Get("admin")
		if !ok {
			// Return a default
			doc = marshalJSON(map[string]any{
				"id":        "admin",
				"userName":  "admin",
				"email":     "admin@example.com",
				"firstName": "Admin",
				"lastName":  "User",
				"self":      fs.URL() + "/user/currentUser",
			})
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, _ := fs.Users.Update("admin", body)
		if doc == nil {
			doc = body
		}
		writeJSON(w, http.StatusOK, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleTenantUsers(w http.ResponseWriter, r *http.Request, segments []string) {
	// segments: [tenantId, "users", ...rest]
	if len(segments) == 2 {
		// /user/{tenant}/users (collection)
		switch r.Method {
		case http.MethodGet:
			items := FilterItems(r, fs.Users.List())
			page := Paginate(r, items)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "users", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			userName := getJSONString(body, "userName")
			if userName == "" {
				userName = NextID()
			}
			body = mergeFields(body, map[string]any{
				"id":   userName,
				"self": fs.URL() + "/user/" + segments[0] + "/users/" + userName,
			})
			fs.Users.CreateWithID(userName, body)
			writeJSON(w, http.StatusCreated, body)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	if len(segments) >= 3 {
		userID := segments[2]

		// /user/{tenant}/users/{id}/roles/inventory
		if len(segments) >= 5 && segments[3] == "roles" && segments[4] == "inventory" {
			fs.handleUserInventoryRoleAssignments(w, r, userID)
			return
		}

		// /user/{tenant}/users/{id}/roles/{roleId}
		if len(segments) >= 5 && segments[3] == "roles" {
			if r.Method == http.MethodDelete {
				writeNoContent(w)
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			return
		}

		// /user/{tenant}/users/{id}/roles
		if len(segments) >= 4 && segments[3] == "roles" {
			switch r.Method {
			case http.MethodGet:
				items := fs.UserRoles.List()
				page := Paginate(r, items)
				writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "roles", page))
			case http.MethodPost:
				body, _ := readBody(r)
				// Body: {"role":{"self":"..."}}, find role and return reference
				var payload struct {
					Role struct {
						Self string `json:"self"`
					} `json:"role"`
				}
				json.Unmarshal(body, &payload)
				for _, doc := range fs.UserRoles.List() {
					if getJSONString(doc, "self") == payload.Role.Self {
						resp := marshalJSON(map[string]any{"role": json.RawMessage(doc)})
						writeJSON(w, http.StatusCreated, resp)
						return
					}
				}
				writeJSON(w, http.StatusCreated, body)
			default:
				writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			}
			return
		}

		// /user/{tenant}/users/{id}/groups
		if len(segments) >= 4 && segments[3] == "groups" {
			fs.handleUserGroupMembership(w, r, userID)
			return
		}

		// /user/{tenant}/users/{id}/devicePermissions
		if len(segments) >= 4 && segments[3] == "devicePermissions" {
			fs.handleDevicePermissions(w, r, userID)
			return
		}

		// /user/{tenant}/users/{id}/tfa
		if len(segments) >= 4 && segments[3] == "tfa" {
			if r.Method == http.MethodDelete {
				writeNoContent(w)
				return
			}
		}

		// /user/{tenant}/users/{id}
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Users.Get(userID)
			if !ok {
				writeNotFound(w, "user")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.Users.Update(userID, body)
			if !ok {
				writeNotFound(w, "user")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.Users.Delete(userID) {
				writeNotFound(w, "user")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
	}
}

func (fs *FakeServer) handleUserGroups(w http.ResponseWriter, r *http.Request, segments []string) {
	// segments: [tenantId, "groups", ...rest]
	if len(segments) == 2 {
		// /user/{tenant}/groups (collection)
		switch r.Method {
		case http.MethodGet:
			items := fs.UserGroups.List()
			page := Paginate(r, items)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "groups", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			_, doc := fs.UserGroups.Create(body, fs.URL()+"/user/"+segments[0]+"/groups")
			writeJSON(w, http.StatusCreated, doc)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	if len(segments) >= 3 {
		groupID := segments[2]

		// /user/{tenant}/groups/{id}/users
		if len(segments) >= 4 && segments[3] == "users" {
			if len(segments) >= 5 {
				// /user/{tenant}/groups/{id}/users/{userId} — DELETE
				if r.Method == http.MethodDelete {
					writeNoContent(w)
					return
				}
				writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
				return
			}
			switch r.Method {
			case http.MethodGet:
				// Return users that are members of this group as references: [{user: {...}}, ...]
				members := fs.GroupMembers[groupID]
				refs := make([]map[string]json.RawMessage, 0, len(members))
				for _, uid := range members {
					if doc, ok := fs.Users.Get(uid); ok {
						refs = append(refs, map[string]json.RawMessage{"user": doc})
					}
				}
				refsJSON, _ := json.Marshal(refs)
				resp := marshalJSON(map[string]any{
					"self": fs.URL() + r.URL.Path,
					"statistics": map[string]int{
						"currentPage": 1,
						"pageSize":    5,
						"totalPages":  1,
					},
				})
				resp = mergeFields(resp, map[string]any{"references": json.RawMessage(refsJSON)})
				writeJSON(w, http.StatusOK, resp)
			case http.MethodPost:
				body, _ := readBody(r)
				// Body: {"user":{"self":"..."}}, extract user self link
				var payload struct {
					User struct {
						Self string `json:"self"`
					} `json:"user"`
				}
				json.Unmarshal(body, &payload)
				if payload.User.Self != "" {
					for _, doc := range fs.Users.List() {
						if getJSONString(doc, "self") == payload.User.Self {
							uid := getJSONString(doc, "id")
							fs.GroupMembers[groupID] = append(fs.GroupMembers[groupID], uid)
							break
						}
					}
				}
				writeJSON(w, http.StatusCreated, body)
			default:
				writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			}
			return
		}

		// /user/{tenant}/groups/{id}/roles/{roleId}
		if len(segments) >= 5 && segments[3] == "roles" {
			if r.Method == http.MethodDelete {
				writeNoContent(w)
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			return
		}

		// /user/{tenant}/groups/{id}/roles
		if len(segments) >= 4 && segments[3] == "roles" {
			switch r.Method {
			case http.MethodGet:
				roles := fs.UserRoles.List()
				refs := make([]map[string]json.RawMessage, 0, len(roles))
				for _, role := range roles {
					refs = append(refs, map[string]json.RawMessage{"role": role})
				}
				refsJSON, _ := json.Marshal(refs)
				resp := marshalJSON(map[string]any{
					"self": fs.URL() + r.URL.Path,
					"statistics": map[string]int{
						"currentPage": 1,
						"pageSize":    2000,
						"totalPages":  1,
					},
				})
				resp = mergeFields(resp, map[string]any{"references": json.RawMessage(refsJSON)})
				writeJSON(w, http.StatusOK, resp)
			case http.MethodPost:
				body, _ := readBody(r)
				var payload struct {
					Role struct {
						Self string `json:"self"`
					} `json:"role"`
				}
				json.Unmarshal(body, &payload)
				for _, doc := range fs.UserRoles.List() {
					if getJSONString(doc, "self") == payload.Role.Self {
						resp := marshalJSON(map[string]any{"role": json.RawMessage(doc)})
						writeJSON(w, http.StatusCreated, resp)
						return
					}
				}
				writeJSON(w, http.StatusCreated, body)
			default:
				writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			}
			return
		}

		// /user/{tenant}/groups/{id}
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.UserGroups.Get(groupID)
			if !ok {
				writeNotFound(w, "user/group")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.UserGroups.Update(groupID, body)
			if !ok {
				writeNotFound(w, "user/group")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.UserGroups.Delete(groupID) {
				writeNotFound(w, "user/group")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
	}
}

func (fs *FakeServer) handleGroupByName(w http.ResponseWriter, r *http.Request, segments []string) {
	// segments: [tenantId, "groupByName", groupName]
	if len(segments) < 3 || r.Method != http.MethodGet {
		writeNotFound(w, "user/group")
		return
	}
	groupName := segments[2]
	for _, doc := range fs.UserGroups.List() {
		if getJSONString(doc, "name") == groupName {
			writeJSON(w, http.StatusOK, doc)
			return
		}
	}
	writeNotFound(w, "user/group")
}

func (fs *FakeServer) handleUserByName(w http.ResponseWriter, r *http.Request, segments []string) {
	// segments: [tenantId, "userByName", username]
	if len(segments) < 3 || r.Method != http.MethodGet {
		writeNotFound(w, "user")
		return
	}
	username := segments[2]
	for _, doc := range fs.Users.List() {
		if getJSONString(doc, "userName") == username {
			writeJSON(w, http.StatusOK, doc)
			return
		}
	}
	writeNotFound(w, "user")
}

func (fs *FakeServer) handleUserRolesGlobal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	// GET /user/roles/{name}
	name := extractID(r.URL.Path, "/user/roles")
	if name != "" {
		doc, ok := fs.UserRoles.Get(name)
		if !ok {
			writeNotFound(w, "role")
			return
		}
		writeJSON(w, http.StatusOK, doc)
		return
	}
	// GET /user/roles (list)
	items := fs.UserRoles.List()
	page := Paginate(r, items)
	writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "roles", page))
}

func (fs *FakeServer) handleInventoryRoles(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/user/inventoryroles")
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.InventoryRoles.Get(id)
			if !ok {
				writeNotFound(w, "user/inventoryRole")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.InventoryRoles.Update(id, body)
			if !ok {
				writeNotFound(w, "user/inventoryRole")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.InventoryRoles.Delete(id) {
				writeNotFound(w, "user/inventoryRole")
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
		items := fs.InventoryRoles.List()
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "roles", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		_, doc := fs.InventoryRoles.Create(body, fs.URL()+"/user/inventoryroles")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleUserInventoryRoleAssignments(w http.ResponseWriter, r *http.Request, _ string) {
	switch r.Method {
	case http.MethodGet:
		resp := marshalJSON(map[string]any{
			"inventoryAssignments": []json.RawMessage{},
			"self":                 fs.URL() + r.URL.Path,
			"statistics": map[string]int{
				"currentPage":   1,
				"pageSize":      5,
				"totalPages":    1,
				"totalElements": 0,
			},
		})
		writeJSON(w, http.StatusOK, resp)

	case http.MethodPost:
		body, _ := readBody(r)
		writeJSON(w, http.StatusCreated, body)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleUserGroupMembership(w http.ResponseWriter, r *http.Request, _ string) {
	switch r.Method {
	case http.MethodGet:
		// Return groups as references: [{group: {...}}, ...]
		groups := fs.UserGroups.List()
		refs := make([]map[string]json.RawMessage, 0, len(groups))
		for _, g := range groups {
			refs = append(refs, map[string]json.RawMessage{"group": g})
		}
		refsJSON, _ := json.Marshal(refs)
		resp := marshalJSON(map[string]any{
			"self": fs.URL() + r.URL.Path,
			"statistics": map[string]int{
				"currentPage": 1,
				"pageSize":    5,
				"totalPages":  1,
			},
		})
		// Inject references array
		resp = mergeFields(resp, map[string]any{"references": json.RawMessage(refsJSON)})
		writeJSON(w, http.StatusOK, resp)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleDevicePermissions(w http.ResponseWriter, r *http.Request, _ string) {
	switch r.Method {
	case http.MethodGet:
		resp := marshalJSON(map[string]any{
			"devicePermissions": map[string]any{},
			"self":              fs.URL() + r.URL.Path,
		})
		writeJSON(w, http.StatusOK, resp)

	case http.MethodPut:
		body, _ := readBody(r)
		writeJSON(w, http.StatusOK, body)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}
