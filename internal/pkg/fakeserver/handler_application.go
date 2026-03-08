package fakeserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleApplications routes /application/ requests.
func (fs *FakeServer) handleApplications(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /application/currentApplication/subscriptions
	if strings.HasPrefix(path, "/application/currentApplication") {
		fs.handleCurrentApplication(w, r)
		return
	}

	// /application/applicationsByOwner/{tenant}
	if strings.HasPrefix(path, "/application/applicationsByOwner/") {
		fs.handleApplicationsByOwner(w, r)
		return
	}

	// /application/applicationsByName/{name}
	if strings.HasPrefix(path, "/application/applicationsByName/") {
		fs.handleApplicationsByName(w, r)
		return
	}

	// /application/applicationsByTenant/{tenant}
	if strings.HasPrefix(path, "/application/applicationsByTenant/") {
		fs.handleApplicationsByTenant(w, r)
		return
	}

	// /application/applications
	if !strings.HasPrefix(path, "/application/applications") {
		writeNotFound(w, "application")
		return
	}

	segments := extractPathSegments(path, "/application/applications")

	// /application/applications (collection)
	if len(segments) == 0 {
		switch r.Method {
		case http.MethodGet:
			items := FilterItems(r, fs.Applications.List())
			page := Paginate(r, items)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "applications", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			body = setDefaultFieldAny(body, "owner", map[string]any{
				"tenant": map[string]any{"id": "t12345"},
				"self":   fs.URL() + "/tenant/tenants/t12345",
			})
			body = setDefaultField(body, "availability", "PRIVATE")
			_, doc := fs.Applications.Create(body, fs.URL()+"/application/applications")
			writeJSON(w, http.StatusCreated, doc)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	appID := segments[0]

	// /application/applications/{id}/versions
	if len(segments) >= 2 && segments[1] == "versions" {
		fs.handleAppVersions(w, r, appID, segments[2:])
		return
	}

	// /application/applications/{id}/binaries
	if len(segments) >= 2 && segments[1] == "binaries" {
		fs.handleAppBinaries(w, r, appID)
		return
	}

	// /application/applications/{id}/clone
	if len(segments) >= 2 && segments[1] == "clone" {
		if r.Method == http.MethodPost {
			doc, ok := fs.Applications.Get(appID)
			if !ok {
				writeNotFound(w, "application")
				return
			}
			// Cumulocity prepends "clone" to the name
			name := getJSONString(doc, "name")
			cloneDoc := mergeFields(doc, map[string]any{
				"name": "clone" + name,
			})
			_, cloned := fs.Applications.Create(cloneDoc, fs.URL()+"/application/applications")
			writeJSON(w, http.StatusCreated, cloned)
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}

	// /application/applications/{id}
	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.Applications.Get(appID)
		if !ok {
			writeNotFound(w, "application")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, ok := fs.Applications.Update(appID, body)
		if !ok {
			writeNotFound(w, "application")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodDelete:
		if !fs.Applications.Delete(appID) {
			writeNotFound(w, "application")
			return
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleAppVersions(w http.ResponseWriter, r *http.Request, appID string, remaining []string) {
	if _, ok := fs.Applications.Get(appID); !ok {
		writeNotFound(w, "application")
		return
	}

	// Version ID is stored as "appID:versionID"
	if len(remaining) == 0 {
		switch r.Method {
		case http.MethodGet:
			var versions []json.RawMessage
			for _, v := range fs.AppVersions.List() {
				if getJSONString(v, "applicationId") == appID {
					versions = append(versions, v)
				}
			}
			if versions == nil {
				versions = []json.RawMessage{}
			}
			page := Paginate(r, versions)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "applicationVersions", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			body = mergeFields(body, map[string]any{"applicationId": appID})
			_, doc := fs.AppVersions.Create(body, fs.URL()+"/application/applications/"+appID+"/versions")
			writeJSON(w, http.StatusCreated, doc)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	versionID := remaining[0]
	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.AppVersions.Get(versionID)
		if !ok {
			writeNotFound(w, "application/version")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodDelete:
		if !fs.AppVersions.Delete(versionID) {
			writeNotFound(w, "application/version")
			return
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleAppBinaries(w http.ResponseWriter, r *http.Request, appID string) {
	if _, ok := fs.Applications.Get(appID); !ok {
		writeNotFound(w, "application")
		return
	}

	switch r.Method {
	case http.MethodPost:
		// Accept any binary upload — just acknowledge
		body, _ := readBody(r)
		_, doc := fs.Binaries.Create(body, fs.URL()+"/inventory/binaries")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleCurrentApplication(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /application/currentApplication/subscriptions
	if strings.HasSuffix(path, "/subscriptions") {
		resp := marshalJSON(map[string]any{
			"users": []map[string]any{
				{
					"tenant": "t12345",
					"name":   "admin",
					"self":   fs.URL() + "/tenant/tenants/t12345",
				},
			},
			"self": fs.URL() + "/application/currentApplication/subscriptions",
		})
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// /application/currentApplication
	resp := marshalJSON(map[string]any{
		"id":   "1",
		"name": "test-application",
		"key":  "test-application-key",
		"type": "MICROSERVICE",
		"self": fs.URL() + "/application/currentApplication",
	})
	writeJSON(w, http.StatusOK, resp)
}

func (fs *FakeServer) handleApplicationsByOwner(w http.ResponseWriter, r *http.Request) {
	items := fs.Applications.List()
	page := Paginate(r, items)
	writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "applications", page))
}

func (fs *FakeServer) handleApplicationsByName(w http.ResponseWriter, r *http.Request) {
	name := extractID(r.URL.Path, "/application/applicationsByName")
	var filtered []json.RawMessage
	for _, item := range fs.Applications.List() {
		if getJSONString(item, "name") == name {
			filtered = append(filtered, item)
		}
	}
	if filtered == nil {
		filtered = []json.RawMessage{}
	}
	page := Paginate(r, filtered)
	writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "applications", page))
}

func (fs *FakeServer) handleApplicationsByTenant(w http.ResponseWriter, r *http.Request) {
	// Return all apps (our fake server is single-tenant)
	items := fs.Applications.List()
	page := Paginate(r, items)
	writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "applications", page))
}

// setDefaultField for non-string values -- overloaded via any type
func setDefaultFieldAny(doc json.RawMessage, key string, value any) json.RawMessage {
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
