package fakeserver

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"
)

// filterByOwnerTenant keeps the applications owned by the given tenant.
func filterByOwnerTenant(items []json.RawMessage, tenant string) []json.RawMessage {
	out := []json.RawMessage{}
	for _, item := range items {
		if getJSONPath(item, "owner.tenant.id") == tenant {
			out = append(out, item)
		}
	}
	return out
}

// filterItemsBy keeps the items for which keep returns true.
func filterItemsBy(items []json.RawMessage, keep func(json.RawMessage) bool) []json.RawMessage {
	out := []json.RawMessage{}
	for _, item := range items {
		if keep(item) {
			out = append(out, item)
		}
	}
	return out
}

// applicationVersions returns the uploaded versions of the application.
func (fs *FakeServer) applicationVersions(appID string) []json.RawMessage {
	var versions []json.RawMessage
	for _, v := range fs.AppVersions.List() {
		if getJSONString(v, "applicationId") == appID {
			versions = append(versions, v)
		}
	}
	return versions
}

// hasApplicationVersions reports whether the application has versions: uploaded
// ones, or an applicationVersions array given when the application was created.
func (fs *FakeServer) hasApplicationVersions(app json.RawMessage) bool {
	if len(fs.applicationVersions(getJSONString(app, "id"))) > 0 {
		return true
	}
	var m struct {
		Versions []json.RawMessage `json:"applicationVersions"`
	}
	return json.Unmarshal(app, &m) == nil && len(m.Versions) > 0
}

// withApplicationVersions adds the applicationVersions (version, tags and
// binary id of each version) to an application which has versions.
func (fs *FakeServer) withApplicationVersions(app json.RawMessage) json.RawMessage {
	versions := fs.applicationVersions(getJSONString(app, "id"))
	if len(versions) == 0 {
		return app
	}
	summaries := make([]map[string]any, 0, len(versions))
	for _, v := range versions {
		summary := map[string]any{
			"version": getJSONString(v, "version"),
			"tags":    getJSONStrings(v, "tags"),
		}
		if tags, _ := summary["tags"].([]string); tags == nil {
			summary["tags"] = []string{}
		}
		if binaryID := getJSONString(v, "binaryId"); binaryID != "" {
			summary["binaryId"] = binaryID
		}
		summaries = append(summaries, summary)
	}
	return mergeFields(app, map[string]any{"applicationVersions": summaries})
}

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

	// /application/applications/{id}/bootstrapUser
	if strings.HasPrefix(path, "/application/applications/") && strings.HasSuffix(path, "/bootstrapUser") {
		fs.handleBootstrapUser(w, r, strings.TrimSuffix(strings.TrimPrefix(path, "/application/applications/"), "/bootstrapUser"))
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
			if owner := r.URL.Query().Get("owner"); owner != "" {
				items = filterByOwnerTenant(items, owner)
			}
			if availability := r.URL.Query().Get("availability"); availability != "" {
				items = filterItemsBy(items, func(item json.RawMessage) bool {
					return getJSONString(item, "availability") == availability
				})
			}
			if hasVersions := r.URL.Query().Get("hasVersions"); hasVersions != "" {
				// Embeds the versions and only returns the applications which have
				// (true) or do not have (false) a version
				withVersions := hasVersions == "true"
				items = filterItemsBy(items, func(item json.RawMessage) bool {
					return fs.hasApplicationVersions(item) == withVersions
				})
				embedded := make([]json.RawMessage, len(items))
				for i, item := range items {
					embedded[i] = fs.withApplicationVersions(item)
				}
				items = embedded
			}
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
			body = withMicroserviceRoles(body)
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
		fs.handleAppBinaries(w, r, appID, segments[2:])
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
		writeJSON(w, http.StatusOK, fs.withApplicationVersions(doc))

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
			// Like the platform, the filters only take effect with the Accept header
			// of a single version, which then answers the version itself. Otherwise
			// they are ignored and the whole collection is returned.
			versionFilter := r.URL.Query().Get("version")
			tagFilter := r.URL.Query().Get("tag")
			if (versionFilter != "" || tagFilter != "") && strings.Contains(r.Header.Get("Accept"), "applicationversion+json") {
				field := "version"
				value := versionFilter
				if tagFilter != "" {
					field = "tag"
					value = tagFilter
				}
				if _, doc, ok := findVersionBy(field, value); ok {
					writeJSON(w, http.StatusOK, doc)
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
			body, binary := readApplicationVersionUpload(r)
			fields := map[string]any{"applicationId": appID}
			if getJSONString(body, "version") == "" {
				// Like the platform, take the version from the manifest of the
				// uploaded web application (cumulocity.json)
				if version := manifestVersion(binary); version != "" {
					fields["version"] = version
				}
			}
			if binary != nil {
				// The platform keeps the uploaded zip as a binary of the application
				meta := marshalJSON(map[string]any{
					"name":          "upload.zip",
					"contentType":   "application/zip",
					"length":        strconv.Itoa(len(binary)),
					"applicationId": appID,
					"created":       time.Now().UTC().Format(time.RFC3339Nano),
				})
				binaryID, _ := fs.AppBinaries.Create(meta, fs.URL()+"/application/applications/"+appID+"/binaries")
				fs.binaryMu.Lock()
				fs.BinaryData[binaryID] = binary
				fs.binaryMu.Unlock()
				fields["binaryId"] = binaryID
			}
			body = mergeFields(body, fields)
			fs.moveTags(appID, "", getJSONStrings(body, "tags"))
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
	// The version is addressed by its id or by its version (e.g. 1.0.1)
	if doc, ok := fs.AppVersions.Get(versionID); !ok || getJSONString(doc, "applicationId") != appID {
		if id, _, found := findVersionBy("version", versionID); found {
			versionID = id
		}
	}
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
		existing, ok := fs.AppVersions.Get(versionID)
		if !ok {
			writeNotFound(w, "application/version")
			return
		}
		// A tag belongs to a single version of an application
		fs.moveTags(appID, versionID, getJSONStrings(body, "tags"))
		merged := mergeFields(existing, mustUnmarshalMap(body))
		doc, _ := fs.AppVersions.Update(versionID, merged)
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

// readApplicationVersionUpload parses the multipart body the SDK uses to upload
// an application version: the metadata (version + tags) as JSON and the
// uploaded zip file (nil when there is none). Falls back to readBody when the
// request isn't multipart.
func readApplicationVersionUpload(r *http.Request) (json.RawMessage, []byte) {
	ct := r.Header.Get("Content-Type")
	if !strings.Contains(ct, "multipart/") {
		b, _ := readBody(r)
		return b, nil
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return json.RawMessage(`{}`), nil
	}
	meta := json.RawMessage(`{}`)
	var binary []byte
	if values, ok := r.MultipartForm.Value["applicationVersion"]; ok && len(values) > 0 {
		meta = json.RawMessage(values[0])
	}
	// The SDK puts applicationVersion as a multipart "file" with JSON content-type
	// rather than a form value, and the zip as applicationBinary.
	for name, files := range r.MultipartForm.File {
		for _, fh := range files {
			f, err := fh.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(f)
			f.Close()
			if name == "applicationVersion" || fh.Filename == "applicationVersion" || strings.HasPrefix(fh.Header.Get("Content-Type"), "application/json") {
				meta = data
			} else if name == "applicationBinary" || binary == nil {
				binary = data
			}
		}
	}
	return meta, binary
}

// manifestVersion returns the version in the cumulocity.json manifest of a
// zipped web application, or "" when there is none.
func manifestVersion(zipData []byte) string {
	if len(zipData) == 0 {
		return ""
	}
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return ""
	}
	for _, f := range zr.File {
		if f.Name != "cumulocity.json" && !strings.HasSuffix(f.Name, "/cumulocity.json") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ""
		}
		data, _ := io.ReadAll(rc)
		rc.Close()
		return getJSONString(data, "version")
	}
	return ""
}

// getJSONStrings returns the string array stored under key, or nil.
func getJSONStrings(doc json.RawMessage, key string) []string {
	var m map[string][]string
	if err := json.Unmarshal(doc, &m); err != nil {
		// other fields are not strings arrays
		var generic map[string]json.RawMessage
		if json.Unmarshal(doc, &generic) != nil {
			return nil
		}
		var tags []string
		_ = json.Unmarshal(generic[key], &tags)
		return tags
	}
	return m[key]
}

// moveTags removes the tags from the other versions of the application: a tag
// (e.g. latest) belongs to a single version. exceptID is the version which is
// keeping the tags.
func (fs *FakeServer) moveTags(appID, exceptID string, tags []string) {
	if len(tags) == 0 {
		return
	}
	for _, v := range fs.AppVersions.List() {
		id := getJSONString(v, "id")
		if id == exceptID || getJSONString(v, "applicationId") != appID {
			continue
		}
		current := getJSONStrings(v, "tags")
		kept := make([]string, 0, len(current))
		for _, tag := range current {
			if !slices.Contains(tags, tag) {
				kept = append(kept, tag)
			}
		}
		if len(kept) != len(current) {
			fs.AppVersions.Update(id, marshalJSON(map[string]any{"tags": kept}))
		}
	}
}

// handleAppBinaries handles /application/applications/{id}/binaries[/{binaryId}].
// Uploaded binaries are stored in fs.AppBinaries (metadata, with
// "applicationId" and "created") and fs.BinaryData (contents).
func (fs *FakeServer) handleAppBinaries(w http.ResponseWriter, r *http.Request, appID string, remaining []string) {
	if _, ok := fs.Applications.Get(appID); !ok {
		writeNotFound(w, "application")
		return
	}

	if len(remaining) > 0 {
		binaryID := remaining[0]
		doc, ok := fs.AppBinaries.Get(binaryID)
		if !ok || getJSONString(doc, "applicationId") != appID {
			writeNotFound(w, "application/binary")
			return
		}
		switch r.Method {
		case http.MethodGet:
			fs.binaryMu.Lock()
			data := fs.BinaryData[binaryID]
			fs.binaryMu.Unlock()
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		case http.MethodDelete:
			fs.AppBinaries.Delete(binaryID)
			fs.binaryMu.Lock()
			delete(fs.BinaryData, binaryID)
			fs.binaryMu.Unlock()
			writeNoContent(w)
		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		attachments := []json.RawMessage{}
		for _, doc := range fs.AppBinaries.List() {
			if getJSONString(doc, "applicationId") == appID {
				attachments = append(attachments, doc)
			}
		}
		writeJSON(w, http.StatusOK, marshalJSON(map[string]any{"attachments": attachments}))

	case http.MethodPost:
		name := "upload"
		var data []byte
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
			if err := r.ParseMultipartForm(32 << 20); err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", "Missing file in multipart form")
				return
			}
			defer file.Close()
			data, _ = io.ReadAll(file)
			name = header.Filename
		} else {
			// Raw upload: keep the bytes as sent (readBody would turn an
			// empty body into {} and normalise the numbers of a JSON file).
			data, _ = io.ReadAll(r.Body)
			_ = r.Body.Close()
		}

		// The platform returns the binary (attachment) metadata, whose id is
		// used to activate the binary via the application's activeVersionId.
		meta := marshalJSON(map[string]any{
			"name":          name,
			"contentType":   "application/zip",
			"length":        strconv.Itoa(len(data)),
			"applicationId": appID,
			"created":       time.Now().UTC().Format(time.RFC3339Nano),
		})
		id, doc := fs.AppBinaries.Create(meta, fs.URL()+"/application/applications/"+appID+"/binaries")
		fs.binaryMu.Lock()
		fs.BinaryData[id] = data
		fs.binaryMu.Unlock()
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
					"tenant":   "t12345",
					"name":     "service_t12345",
					"password": "service-pass",
					"self":     fs.URL() + "/tenant/tenants/t12345",
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

// withMicroserviceRoles adds the roles and requiredRoles of a microservice
// application, as the platform does: they are taken from the manifest when it
// declares them and are empty lists otherwise. Other application types and
// values which are already set are left alone.
func withMicroserviceRoles(doc json.RawMessage) json.RawMessage {
	if getJSONString(doc, "type") != "MICROSERVICE" {
		return doc
	}
	var manifest struct {
		Manifest struct {
			Roles         []string `json:"roles"`
			RequiredRoles []string `json:"requiredRoles"`
		} `json:"manifest"`
	}
	_ = json.Unmarshal(doc, &manifest)
	roles, required := manifest.Manifest.Roles, manifest.Manifest.RequiredRoles
	if roles == nil {
		roles = []string{}
	}
	if required == nil {
		required = []string{}
	}
	doc = setDefaultFieldAny(doc, "roles", roles)
	return setDefaultFieldAny(doc, "requiredRoles", required)
}

// handleBootstrapUser handles GET /application/applications/{id}/bootstrapUser:
// the credentials a microservice uses to read its subscriptions.
func (fs *FakeServer) handleBootstrapUser(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	app, ok := fs.Applications.Get(id)
	if !ok {
		writeNotFound(w, "application")
		return
	}
	writeJSON(w, http.StatusOK, marshalJSON(map[string]any{
		"tenant":   "t12345",
		"name":     "servicebootstrap_" + getJSONString(app, "name"),
		"password": "bootstrap-pass",
	}))
}
