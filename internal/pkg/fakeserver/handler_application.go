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

	// /application/applicationsByUser/{username}
	if strings.HasPrefix(path, "/application/applicationsByUser/") {
		fs.handleApplicationsByUser(w, r)
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

	// findVersionByValue locates a stored version document whose "version" or
	// tag value matches.
	findVersionBy := func(field, value string) (string, json.RawMessage, bool) {
		for _, v := range fs.AppVersions.List() {
			if getJSONString(v, "applicationId") != appID {
				continue
			}
			if field == "version" && getJSONString(v, "version") == value {
				return getJSONString(v, "id"), v, true
			}
			if field == "tag" {
				var tmp struct {
					Tags []string `json:"tags"`
					ID   string   `json:"id"`
				}
				if err := json.Unmarshal(v, &tmp); err != nil {
					continue
				}
				for _, t := range tmp.Tags {
					if t == value {
						return tmp.ID, v, true
					}
				}
			}
		}
		return "", nil, false
	}

	// Version ID is stored as "appID:versionID"
	if len(remaining) == 0 {
		switch r.Method {
		case http.MethodGet:
			// /application/applications/{id}/versions?version=...|tag=...
			versionFilter := r.URL.Query().Get("version")
			tagFilter := r.URL.Query().Get("tag")
			if versionFilter != "" || tagFilter != "" {
				field := "version"
				value := versionFilter
				if tagFilter != "" {
					field = "tag"
					value = tagFilter
				}
				if _, doc, ok := findVersionBy(field, value); ok {
					// SDK calls ExecuteCollection so return a collection
					page := Paginate(r, []json.RawMessage{doc})
					writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "applicationVersions", page))
					return
				}
				writeNotFound(w, "application/version")
				return
			}

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
			// The SDK sends multipart with an "applicationVersion" JSON part
			// containing version + tags fields plus a binary part.  Parse
			// out the version metadata when it is present, otherwise fall
			// back to the raw body.
			body := readApplicationVersionBody(r)
			body = mergeFields(body, map[string]any{"applicationId": appID})
			_, doc := fs.AppVersions.Create(body, fs.URL()+"/application/applications/"+appID+"/versions")
			writeJSON(w, http.StatusCreated, doc)

		case http.MethodDelete:
			versionFilter := r.URL.Query().Get("version")
			tagFilter := r.URL.Query().Get("tag")
			field := "version"
			value := versionFilter
			if tagFilter != "" {
				field = "tag"
				value = tagFilter
			}
			if value == "" {
				writeError(w, http.StatusBadRequest, "general/badRequest", "version or tag is required")
				return
			}
			if id, _, ok := findVersionBy(field, value); ok {
				fs.AppVersions.Delete(id)
				writeNoContent(w)
				return
			}
			writeNotFound(w, "application/version")

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

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		// `versionID` may be the literal version string, locate the stored
		// document accordingly.
		id, existing, ok := findVersionBy("version", versionID)
		if !ok {
			writeNotFound(w, "application/version")
			return
		}
		merged := mergeFields(existing, mustUnmarshalMap(body))
		doc, _ := fs.AppVersions.Update(id, merged)
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

// mustUnmarshalMap decodes the body into a map[string]any (returning an empty
// map on failure) so it can be passed to mergeFields.
func mustUnmarshalMap(body json.RawMessage) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		return map[string]any{}
	}
	return m
}

// readApplicationVersionBody parses the multipart body the SDK uses to upload
// an application version and returns the metadata (version + tags) as JSON.
// Falls back to readBody when the request isn't multipart.
func readApplicationVersionBody(r *http.Request) json.RawMessage {
	ct := r.Header.Get("Content-Type")
	if !strings.Contains(ct, "multipart/") {
		b, _ := readBody(r)
		return b
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return json.RawMessage(`{}`)
	}
	if values, ok := r.MultipartForm.Value["applicationVersion"]; ok && len(values) > 0 {
		return json.RawMessage(values[0])
	}
	// Try files (the SDK puts applicationVersion as a multipart "file" with
	// JSON content-type rather than a form value).
	for _, files := range r.MultipartForm.File {
		for _, fh := range files {
			if fh.Filename == "applicationVersion" || strings.HasPrefix(fh.Header.Get("Content-Type"), "application/json") {
				f, err := fh.Open()
				if err != nil {
					continue
				}
				defer f.Close()
				buf := make([]byte, fh.Size)
				_, _ = f.Read(buf)
				return json.RawMessage(buf)
			}
		}
	}
	return json.RawMessage(`{}`)
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

	// /application/currentApplication/settings
	if strings.HasSuffix(path, "/settings") {
		// Settings are returned as a JSON array of {key,value} objects, but the
		// SDK calls ExecuteCollection with empty result property so the body is
		// the raw array.
		resp := marshalJSON([]map[string]any{
			{"key": "host", "value": "https://example.cumulocity.com"},
		})
		writeJSON(w, http.StatusOK, resp)
		return
	}

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

func (fs *FakeServer) handleApplicationsByUser(w http.ResponseWriter, r *http.Request) {
	items := fs.Applications.List()
	page := Paginate(r, items)
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
