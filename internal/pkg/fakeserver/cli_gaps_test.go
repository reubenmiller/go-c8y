package fakeserver

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"slices"
	"strings"
	"testing"
)

const jsonAccept = "application/json"

// postJSON creates an object and returns its id.
func postJSON(t *testing.T, fs *FakeServer, path, body string) string {
	t.Helper()
	status, out := doRequest(t, fs, http.MethodPost, path, body, jsonAccept)
	if status != http.StatusCreated && status != http.StatusOK {
		t.Fatalf("POST %s: status=%d body=%s", path, status, out)
	}
	return getJSONString(json.RawMessage(out), "id")
}

// upload sends a multipart request with a file part and returns the response.
func upload(t *testing.T, fs *FakeServer, method, path, field, filename, contentType string, content []byte, extra map[string]string) (int, http.Header, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, value := range extra {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="`+name+`"; filename="`+name+`"`)
		h.Set("Content-Type", "application/json")
		part, err := mw.CreatePart(h)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte(value))
	}
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", `form-data; name="`+field+`"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(content)
	_ = mw.Close()

	req, _ := http.NewRequest(method, fs.URL()+path, &buf)
	req.SetBasicAuth("t12345/admin", "admin-pass")
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", jsonAccept)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	out, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(out)
}

func TestBinaryDownload_HasContentDisposition(t *testing.T) {
	fs := New(t)
	status, _, out := upload(t, fs, http.MethodPost, "/inventory/binaries", "file", "my file.txt", "text/plain", []byte("hello"), nil)
	if status != http.StatusCreated {
		t.Fatalf("upload: status=%d body=%s", status, out)
	}
	id := getJSONString(json.RawMessage(out), "id")

	req, _ := http.NewRequest(http.MethodGet, fs.URL()+"/inventory/binaries/"+id, nil)
	req.SetBasicAuth("t12345/admin", "admin-pass")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "hello" {
		t.Errorf("content = %q, want hello", body)
	}
	if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="my file.txt"` {
		t.Errorf("Content-Disposition = %q", got)
	}
}

func TestEventBinary_ReplaceContent(t *testing.T) {
	fs := New(t)
	source := postJSON(t, fs, "/inventory/managedObjects", `{"name":"device"}`)
	event := postJSON(t, fs, "/event/events", `{"type":"t","text":"x","time":"2026-01-01T00:00:00Z","source":{"id":"`+source+`"}}`)

	path := "/event/events/" + event + "/binaries"
	if status, _, out := upload(t, fs, http.MethodPost, path, "file", "first.txt", "text/plain", []byte("one"), nil); status != http.StatusCreated {
		t.Fatalf("create: status=%d body=%s", status, out)
	}

	req, _ := http.NewRequest(http.MethodPut, fs.URL()+path, strings.NewReader("two two"))
	req.SetBasicAuth("t12345/admin", "admin-pass")
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Accept", jsonAccept)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("replace: status=%d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, fs.URL()+path, nil)
	req.SetBasicAuth("t12345/admin", "admin-pass")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "two two" {
		t.Errorf("content = %q, want the replaced content", body)
	}
	if got := resp.Header.Get("Content-Disposition"); got != `attachment; filename="first.txt"` {
		t.Errorf("Content-Disposition = %q", got)
	}

	// Without an attachment there is nothing to replace
	other := postJSON(t, fs, "/event/events", `{"type":"t","text":"y","time":"2026-01-01T00:00:00Z","source":{"id":"`+source+`"}}`)
	req, _ = http.NewRequest(http.MethodPut, fs.URL()+"/event/events/"+other+"/binaries", strings.NewReader("x"))
	req.SetBasicAuth("t12345/admin", "admin-pass")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("replace without an attachment: status=%d, want 404", resp.StatusCode)
	}
}

func TestInventoryQuery_ByGroupIDAndNot(t *testing.T) {
	fs := New(t)
	group := postJSON(t, fs, "/inventory/managedObjects", `{"name":"group","c8y_IsDeviceGroup":{}}`)
	inGroup := postJSON(t, fs, "/inventory/managedObjects", `{"name":"in-group","c8y_IsDevice":{}}`)
	nested := postJSON(t, fs, "/inventory/managedObjects", `{"name":"nested","c8y_IsDevice":{}}`)
	addition := postJSON(t, fs, "/inventory/managedObjects", `{"name":"addition","c8y_Software":{}}`)
	outside := postJSON(t, fs, "/inventory/managedObjects", `{"name":"outside","c8y_IsDevice":{}}`)
	for _, link := range []struct{ parent, rel, child string }{
		{group, "childAssets", inGroup},
		{inGroup, "childDevices", nested},
		{group, "childAdditions", addition},
	} {
		if status, out := doRequest(t, fs, http.MethodPost, "/inventory/managedObjects/"+link.parent+"/"+link.rel, `{"managedObject":{"id":"`+link.child+`"}}`, jsonAccept); status != http.StatusCreated {
			t.Fatalf("link %v: status=%d body=%s", link, status, out)
		}
	}

	list := func(query string) []string {
		t.Helper()
		status, out := doRequest(t, fs, http.MethodGet, "/inventory/managedObjects?pageSize=100&query="+urlQueryEscape(query), "", jsonAccept)
		if status != http.StatusOK {
			t.Fatalf("query %q: status=%d body=%s", query, status, out)
		}
		var res struct {
			ManagedObjects []json.RawMessage `json:"managedObjects"`
		}
		if err := json.Unmarshal([]byte(out), &res); err != nil {
			t.Fatal(err)
		}
		return ids(res.ManagedObjects)
	}
	sameSet := func(got []string, want ...string) bool {
		if len(got) != len(want) {
			return false
		}
		for _, w := range want {
			found := false
			for _, g := range got {
				found = found || g == w
			}
			if !found {
				return false
			}
		}
		return true
	}

	if got := list("$filter=bygroupid(" + group + ")"); !sameSet(got, inGroup, nested, addition) {
		t.Errorf("bygroupid(group) = %v, want %v", got, []string{inGroup, nested, addition})
	}
	if got := list("$filter=bygroupid(" + group + ") and has(c8y_IsDevice)"); !sameSet(got, inGroup, nested) {
		t.Errorf("bygroupid and has = %v", got)
	}
	if got := list("$filter=(bygroupid(" + group + ") and not(has(c8y_IsDevice)))"); !sameSet(got, addition) {
		t.Errorf("bygroupid and not(has) = %v, want [%s]", got, addition)
	}
	// The tenant has devices of its own, so only look at the test objects
	got := list("$filter=has(c8y_IsDevice) and not(name eq 'in-*')")
	for want, shouldMatch := range map[string]bool{nested: true, outside: true, inGroup: false} {
		if slices.Contains(got, want) != shouldMatch {
			t.Errorf("not(name eq 'in-*'): %s listed = %v, want %v", want, !shouldMatch, shouldMatch)
		}
	}
	if got := list("$filter=bygroupid(" + outside + ")"); len(got) != 0 {
		t.Errorf("bygroupid of an object without children = %v, want none", got)
	}
}

func urlQueryEscape(s string) string {
	return strings.NewReplacer("%", "%25", " ", "%20", "$", "%24", "'", "%27", "(", "%28", ")", "%29", "&", "%26", "+", "%2B", "*", "%2A").Replace(s)
}

func TestManagedObjectList_WithParents(t *testing.T) {
	fs := New(t)
	parent := postJSON(t, fs, "/inventory/managedObjects", `{"name":"parent","type":"c8y_Software","c8y_Extra":"x"}`)
	child := postJSON(t, fs, "/inventory/managedObjects", `{"name":"child"}`)
	doRequest(t, fs, http.MethodPost, "/inventory/managedObjects/"+parent+"/childAdditions", `{"managedObject":{"id":"`+child+`"}}`, jsonAccept)

	status, out := doRequest(t, fs, http.MethodGet, "/inventory/managedObjects?pageSize=100&withParents=true&query="+urlQueryEscape("$filter=bygroupid("+parent+")"), "", jsonAccept)
	if status != http.StatusOK {
		t.Fatalf("status=%d body=%s", status, out)
	}
	var res struct {
		ManagedObjects []json.RawMessage `json:"managedObjects"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil || len(res.ManagedObjects) != 1 {
		t.Fatalf("got %s (%v)", out, err)
	}
	item := res.ManagedObjects[0]
	if got := getJSONPath(item, "additionParents.references"); !strings.Contains(got, parent) {
		t.Errorf("additionParents.references = %s, want the parent %s", got, parent)
	}
	// The parent is the managed object itself, so clients can read its type
	var got struct {
		AdditionParents struct {
			References []struct {
				ManagedObject struct {
					ID, Name, Type string
					CExtra         string `json:"c8y_Extra"`
				} `json:"managedObject"`
			} `json:"references"`
		} `json:"additionParents"`
	}
	if err := json.Unmarshal(item, &got); err != nil {
		t.Fatal(err)
	}
	mo := got.AdditionParents.References[0].ManagedObject
	if mo.ID != parent || mo.Name != "parent" || mo.Type != "c8y_Software" || mo.CExtra != "x" {
		t.Errorf("parent = %+v", mo)
	}

	// Not added to the stored object
	_, plain := doRequest(t, fs, http.MethodGet, "/inventory/managedObjects/"+child, "", jsonAccept)
	if strings.Contains(plain, "additionParents") {
		t.Errorf("the parents must not be stored: %s", plain)
	}
}

func TestStandardWebApplications_AreOwnedByManagementTenant(t *testing.T) {
	fs := New(t)
	status, out := doRequest(t, fs, http.MethodGet, "/application/applications?name=cockpit&type=HOSTED&owner=t12345", "", jsonAccept)
	if status != http.StatusOK {
		t.Fatalf("status=%d", status)
	}
	var res struct {
		Applications []json.RawMessage `json:"applications"`
	}
	_ = json.Unmarshal([]byte(out), &res)
	if len(res.Applications) != 0 {
		t.Errorf("cockpit is owned by the tenant: %s", out)
	}
	_, out = doRequest(t, fs, http.MethodGet, "/application/applications?name=cockpit&type=HOSTED&owner=management", "", jsonAccept)
	_ = json.Unmarshal([]byte(out), &res)
	if len(res.Applications) != 1 {
		t.Errorf("cockpit is not owned by the management tenant: %s", out)
	}
}

func zipWithManifest(t *testing.T, manifest string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("cumulocity.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(manifest))
	_ = zw.Close()
	return buf.Bytes()
}

func TestApplicationVersions_Plugins(t *testing.T) {
	fs := New(t)
	app := postJSON(t, fs, "/application/applications", `{"name":"plugin1","key":"plugin1-key","contextPath":"plugin1","type":"HOSTED"}`)
	versions := "/application/applications/" + app + "/versions"

	// The version comes from the manifest of the zip when it is not given
	status, _, out := upload(t, fs, http.MethodPost, versions, "applicationBinary", "plugin.zip", "application/octet-stream",
		zipWithManifest(t, `{"version":"1.0.1"}`), map[string]string{"applicationVersion": `{"tags":["latest"]}`})
	if status != http.StatusCreated || getJSONString(json.RawMessage(out), "version") != "1.0.1" {
		t.Fatalf("create 1.0.1: status=%d body=%s", status, out)
	}
	if getJSONString(json.RawMessage(out), "binaryId") == "" {
		t.Errorf("the uploaded zip is kept as a binary: %s", out)
	}
	status, _, out = upload(t, fs, http.MethodPost, versions, "applicationBinary", "plugin.zip", "application/octet-stream",
		zipWithManifest(t, `{"version":"1.0.2"}`), map[string]string{"applicationVersion": `{"version":"1.0.2","tags":["latest"]}`})
	if status != http.StatusCreated {
		t.Fatalf("create 1.0.2: status=%d body=%s", status, out)
	}

	// A tag belongs to a single version: latest moved to 1.0.2
	byTag := func(tag string) string {
		req, _ := http.NewRequest(http.MethodGet, fs.URL()+versions+"?tag="+tag, nil)
		req.SetBasicAuth("t12345/admin", "admin-pass")
		req.Header.Set("Accept", "application/vnd.com.nsn.cumulocity.applicationversion+json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != http.StatusOK {
			return ""
		}
		return getJSONString(b, "version")
	}
	if got := byTag("latest"); got != "1.0.2" {
		t.Errorf("latest = %q, want 1.0.2", got)
	}

	// Update the tags by version (the version is part of the path)
	status, out = doRequest(t, fs, http.MethodPut, versions+"/1.0.1", `{"tags":["latest","v1"]}`, jsonAccept)
	if status != http.StatusOK {
		t.Fatalf("update: status=%d body=%s", status, out)
	}
	if got := byTag("latest"); got != "1.0.1" {
		t.Errorf("latest after the update = %q, want 1.0.1", got)
	}
	if got := byTag("v1"); got != "1.0.1" {
		t.Errorf("v1 = %q, want 1.0.1", got)
	}

	// Without the Accept header of a single version, the filters are ignored
	status, out = doRequest(t, fs, http.MethodGet, versions+"?tag=latest", "", jsonAccept)
	var collection struct {
		ApplicationVersions []json.RawMessage `json:"applicationVersions"`
	}
	if err := json.Unmarshal([]byte(out), &collection); err != nil || status != http.StatusOK || len(collection.ApplicationVersions) != 2 {
		t.Errorf("collection = %s (%v)", out, err)
	}

	// The versions are embedded in the applications which have some
	_, out = doRequest(t, fs, http.MethodGet, "/application/applications?hasVersions=true&type=HOSTED", "", jsonAccept)
	var apps struct {
		Applications []struct {
			Name     string `json:"name"`
			Versions []struct {
				Version string   `json:"version"`
				Tags    []string `json:"tags"`
			} `json:"applicationVersions"`
		} `json:"applications"`
	}
	if err := json.Unmarshal([]byte(out), &apps); err != nil {
		t.Fatal(err)
	}
	found := map[string]int{}
	for _, a := range apps.Applications {
		found[a.Name] = len(a.Versions)
	}
	if found["plugin1"] != 2 || found["Cumulocity community plugins"] != 1 {
		t.Errorf("applications with versions = %v", found)
	}
	if _, ok := found["cockpit"]; ok {
		t.Errorf("cockpit has no versions, but is listed with hasVersions=true")
	}

	// Filter by availability: the shared plugin of the management tenant
	_, out = doRequest(t, fs, http.MethodGet, "/application/applications?availability=SHARED", "", jsonAccept)
	if !strings.Contains(out, "sag-pkg-community-plugins") || strings.Contains(out, "plugin1") {
		t.Errorf("shared applications = %s", out)
	}
}

func TestUserGroups_HaveNumericIDs(t *testing.T) {
	fs := New(t)
	_, out := doRequest(t, fs, http.MethodGet, "/user/t12345/groups", "", jsonAccept)
	var res struct {
		Groups []struct {
			ID   json.Number `json:"id"`
			Name string      `json:"name"`
		} `json:"groups"`
	}
	if err := json.Unmarshal([]byte(out), &res); err != nil || len(res.Groups) < 2 {
		t.Fatalf("groups = %s (%v)", out, err)
	}
	if !strings.Contains(out, `"id":`+res.Groups[0].ID.String()+`,`) && !strings.Contains(out, `"id":`+res.Groups[0].ID.String()+`}`) {
		t.Errorf("group ids must be numbers: %s", out)
	}
	_, created := doRequest(t, fs, http.MethodPost, "/user/t12345/groups", `{"name":"mygroup"}`, jsonAccept)
	var group struct {
		ID json.Number `json:"id"`
	}
	if err := json.Unmarshal([]byte(created), &group); err != nil || group.ID.String() == "" || strings.Contains(created, `"id":"`) {
		t.Errorf("created group = %s (%v)", created, err)
	}
	if status, _ := doRequest(t, fs, http.MethodGet, "/user/t12345/groups/"+group.ID.String(), "", jsonAccept); status != http.StatusOK {
		t.Errorf("GET group by its id: status=%d", status)
	}
}

func TestUsers_SelfLinkEscapesAt(t *testing.T) {
	fs := New(t)
	_, out := doRequest(t, fs, http.MethodPost, "/user/t12345/users", `{"userName":"peter@example.com","email":"peter@example.com","password":"s3cret!Pass"}`, jsonAccept)
	if got := getJSONString(json.RawMessage(out), "self"); !strings.HasSuffix(got, "/user/t12345/users/peter%40example.com") {
		t.Errorf("self = %q", got)
	}
	if strings.Contains(out, "s3cret") {
		t.Errorf("the password is never returned: %s", out)
	}
	// The user is still found by name
	if status, _ := doRequest(t, fs, http.MethodGet, "/user/t12345/users/peter%40example.com", "", jsonAccept); status != http.StatusOK {
		t.Errorf("GET user: status=%d", status)
	}
}

func TestGlobalRoles_AreSeeded(t *testing.T) {
	fs := New(t)
	status, out := doRequest(t, fs, http.MethodGet, "/user/roles?pageSize=2000", "", jsonAccept)
	if status != http.StatusOK || !strings.Contains(out, "ROLE_ALARM_READ") || !strings.Contains(out, "ROLE_INVENTORY_ADMIN") {
		t.Errorf("roles = %d %s", status, out)
	}
}

func TestBasicAuth_WrongPasswordIsRejected(t *testing.T) {
	fs := New(t)
	get := func(user, password string) int {
		req, _ := http.NewRequest(http.MethodGet, fs.URL()+"/inventory/managedObjects", nil)
		req.SetBasicAuth(user, password)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if got := get("t12345/admin", "admin-pass"); got != http.StatusOK {
		t.Errorf("the right password: status=%d", got)
	}
	if got := get("admin", "admin-pass"); got != http.StatusOK {
		t.Errorf("the right password, without the tenant: status=%d", got)
	}
	if got := get("t12345/admin", "wrong"); got != http.StatusUnauthorized {
		t.Errorf("a wrong password: status=%d, want 401", got)
	}

	// A user created with a password must use it
	doRequest(t, fs, http.MethodPost, "/user/t12345/users", `{"userName":"alice","password":"alice-pass"}`, jsonAccept)
	if got := get("t12345/alice", "alice-pass"); got != http.StatusOK {
		t.Errorf("alice with her password: status=%d", got)
	}
	if got := get("t12345/alice", "admin-pass"); got != http.StatusUnauthorized {
		t.Errorf("alice with another password: status=%d, want 401", got)
	}
	// Users the server does not know the password of are accepted
	if got := get("t12345/device_xyz", "anything"); got != http.StatusOK {
		t.Errorf("unknown user: status=%d", got)
	}
}

func TestMicroserviceInstances_AreSeeded(t *testing.T) {
	fs := New(t)
	_, out := doRequest(t, fs, http.MethodGet, "/application/applicationsByName/advanced-software-mgmt?type=MICROSERVICE", "", jsonAccept)
	var apps struct {
		Applications []struct {
			ID string `json:"id"`
		} `json:"applications"`
	}
	if err := json.Unmarshal([]byte(out), &apps); err != nil || len(apps.Applications) != 1 {
		t.Fatalf("microservice = %s (%v)", out, err)
	}
	_, out = doRequest(t, fs, http.MethodGet, "/inventory/managedObjects?type=c8y_Application_"+apps.Applications[0].ID, "", jsonAccept)
	if !strings.Contains(out, "advanced-software-mgmt-scope-") {
		t.Errorf("the status of the instances = %s", out)
	}
}

func TestNewTLSServer_ServesHTTPS(t *testing.T) {
	fs, err := NewTLSServer("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer fs.Server.Close()
	if !strings.HasPrefix(fs.URL(), "https://") {
		t.Fatalf("URL = %s, want https", fs.URL())
	}
	client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec // self-signed test certificate
	req, _ := http.NewRequest(http.MethodGet, fs.URL()+"/tenant/currentTenant", nil)
	req.SetBasicAuth("t12345/admin", "admin-pass")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "https://") {
		t.Errorf("status=%d body=%s: the self links should use https", resp.StatusCode, body)
	}
}
