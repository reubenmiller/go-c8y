package fakeserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// CapturedRequest is a lightweight record of a request the fake server handled,
// used by tests to assert pagination request patterns (e.g. that the id keyset
// always re-requests page 1 with an advancing _id cursor).
type CapturedRequest struct {
	Method   string
	Path     string
	RawQuery string
}

// FakeServer holds all in-memory stores and the httptest.Server.
type FakeServer struct {
	Server *httptest.Server

	// Stores for each resource type
	Alarms                     *Store
	Events                     *Store
	Operations                 *Store
	Measurements               *Store
	ManagedObjects             *Store
	Binaries                   *Store
	Applications               *Store
	AppVersions                *Store
	AppBinaries                *Store // hosted application binaries (attachments)
	Users                      *Store
	UserGroups                 *Store
	UserRoles                  *Store
	InventoryRoles             *Store
	Tenants                    *Store
	TenantOptions              *Store
	AuditRecords               *Store
	RetentionRules             *Store
	TrustedCerts               *Store
	BulkOperations             *Store
	LoginOptions               *Store
	Features                   *Store
	DeviceRequests             *Store
	Notification2Subscriptions *Store
	RemoteAccessConfigs        *Store
	SystemOptions              *Store

	// Enrollments: external ID → the one-time password a device requests its certificate with
	// (from a bulk registration); see handleEST
	Enrollments *Store
	// ca signs the certificates of enrolling devices
	ca estCA

	// passwords holds the passwords of the users, by user name, which are not
	// part of the user documents (Cumulocity never returns them). Users without
	// an entry accept any password.
	passwordsMu sync.RWMutex
	passwords   map[string]string

	// Identity stores: externalType+externalValue → managedObjectID
	ExternalIDs *Store

	// Child relationship stores: parentID → list of childIDs
	ChildDevices   map[string][]string // guarded by ManagedObjects.mu
	ChildAssets    map[string][]string
	ChildAdditions map[string][]string

	// Event binary store: eventID → binary bytes
	EventBinaries map[string][]byte

	// Binary data store: binaryID → binary bytes
	BinaryData map[string][]byte
	// binaryMu guards BinaryData for the application binary handlers
	binaryMu sync.Mutex

	// Group membership: groupID → list of userIDs
	GroupMembers map[string][]string

	// Login-option access mappings: loginOption typeOrId → Store of access mappings
	LoginOptionAccessMappings map[string]*Store
	// Login-option inventory access mappings: loginOption typeOrId → Store
	LoginOptionInventoryAccessMappings map[string]*Store

	// Realtime is the state of any active /cep/realtime websocket connections.
	Realtime *realtimeState
	// Notification2 is the state of any active /notification2/consumer/
	// websocket connections.
	Notification2 *notification2State

	// Request capture (for pagination assertions in tests).
	capturedMu sync.Mutex
	captured   []CapturedRequest
}

// Requests returns a copy of the requests captured so far.
func (fs *FakeServer) Requests() []CapturedRequest {
	fs.capturedMu.Lock()
	defer fs.capturedMu.Unlock()
	return append([]CapturedRequest(nil), fs.captured...)
}

// ResetRequests clears the captured request log.
func (fs *FakeServer) ResetRequests() {
	fs.capturedMu.Lock()
	defer fs.capturedMu.Unlock()
	fs.captured = nil
}

func (fs *FakeServer) record(r *http.Request) {
	fs.capturedMu.Lock()
	fs.captured = append(fs.captured, CapturedRequest{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery})
	fs.capturedMu.Unlock()
}

// New creates a new FakeServer with an httptest.Server and registers cleanup.
func New(t *testing.T) *FakeServer {
	fs := newUnstarted()
	fs.Server = httptest.NewServer(fs.handler())
	t.Cleanup(fs.Server.Close)

	// Seed default data (current tenant, current user)
	fs.seedDefaults()

	return fs
}

// NewServer builds and starts a FakeServer listening on addr (e.g.
// "127.0.0.1:8111"; an empty addr picks a random free port like the test
// constructor). It is the entry point for the standalone c8y-fakeserver
// binary, which needs the same stateful server the test suite uses but without
// a *testing.T. Call fs.Server.Close() to stop it.
func NewServer(addr string) (*FakeServer, error) {
	return newServer(addr, false)
}

// NewTLSServer is like NewServer but serves https, using the self-signed
// certificate of net/http/httptest (valid for 127.0.0.1 and example.com). The
// self links then use https like the ones of a real tenant; clients need to
// accept the certificate (e.g. the insecure setting of go-c8y-cli).
func NewTLSServer(addr string) (*FakeServer, error) {
	return newServer(addr, true)
}

func newServer(addr string, useTLS bool) (*FakeServer, error) {
	fs := newUnstarted()
	srv := httptest.NewUnstartedServer(fs.handler())
	if addr != "" {
		l, err := net.Listen("tcp", addr)
		if err != nil {
			return nil, err
		}
		_ = srv.Listener.Close()
		srv.Listener = l
	}
	if useTLS {
		srv.StartTLS()
	} else {
		srv.Start()
	}
	fs.Server = srv

	// Seed default data (current tenant, current user)
	fs.seedDefaults()

	return fs, nil
}

// newUnstarted allocates the FakeServer with all stores initialised but no
// running HTTP server. Shared by New (tests) and NewServer (binary) so the two
// entry points never drift.
func newUnstarted() *FakeServer {
	fs := &FakeServer{
		Alarms:                     NewStore(),
		Events:                     NewStore(),
		Operations:                 NewStore(),
		Measurements:               NewStore(),
		ManagedObjects:             NewStore(),
		Binaries:                   NewStore(),
		Applications:               NewStore(),
		AppVersions:                NewStore(),
		AppBinaries:                NewStore(),
		Users:                      NewStore(),
		UserGroups:                 NewStore(),
		UserRoles:                  NewStore(),
		InventoryRoles:             NewStore(),
		Tenants:                    NewStore(),
		TenantOptions:              NewStore(),
		AuditRecords:               NewStore(),
		RetentionRules:             NewStore(),
		TrustedCerts:               NewStore(),
		BulkOperations:             NewStore(),
		LoginOptions:               NewStore(),
		Features:                   NewStore(),
		DeviceRequests:             NewStore(),
		ExternalIDs:                NewStore(),
		Notification2Subscriptions: NewStore(),
		RemoteAccessConfigs:        NewStore(),
		SystemOptions:              NewStore(),
		Enrollments:                NewStore(),

		ChildDevices:   make(map[string][]string),
		ChildAssets:    make(map[string][]string),
		ChildAdditions: make(map[string][]string),
		EventBinaries:  make(map[string][]byte),
		BinaryData:     make(map[string][]byte),
		GroupMembers:   make(map[string][]string),

		LoginOptionAccessMappings:          make(map[string]*Store),
		LoginOptionInventoryAccessMappings: make(map[string]*Store),

		passwords:     map[string]string{"admin": "admin-pass"},
		Realtime:      newRealtimeState(),
		Notification2: newNotification2State(),
	}
	return fs
}

// handler builds the routed, auth-wrapped http.Handler for this server.
func (fs *FakeServer) handler() http.Handler {
	mux := http.NewServeMux()
	fs.registerRoutes(mux)

	// Wrap with basic auth check – returns 401 when no credentials are supplied.
	// Some endpoints are public (no auth required).
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		public := strings.HasPrefix(p, "/tenant/loginOptions") ||
			strings.HasPrefix(p, "/tenant/oauth") ||
			p == "/cep/realtime" ||
			strings.HasPrefix(p, "/notification2/consumer")
		if !public && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized, "security/Unauthorized", "Missing credentials") //nolint:misspell
			return
		}
		if !public && !fs.credentialsValid(r) {
			writeError(w, http.StatusUnauthorized, "security/Unauthorized", "Bad credentials") //nolint:misspell
			return
		}
		fs.record(r)
		if isNoAcceptJSONWrite(r) {
			// Cumulocity only returns the created/updated object when the request
			// asks for it with an Accept header (e.g. the cli's --noAccept flag).
			// The realtime notification still carries the object.
			w = &noAcceptResponseWriter{ResponseWriter: w}
		}
		fs.withRealtimePublish(w, r, mux.ServeHTTP)
	})
}

// noAcceptPaths are the REST resources whose POST/PUT response body depends on
// the Accept header. Other endpoints (e.g. EST certificate enrollment, binary
// uploads, oauth) return their body regardless.
var noAcceptPaths = []string{
	"/inventory/managedObjects",
	"/alarm/alarms",
	"/event/events",
	"/measurement/measurements",
	"/devicecontrol/operations",
	"/identity/",
	"/user/",
}

// isNoAcceptJSONWrite reports whether r is a json POST or PUT request, without
// an Accept header, to one of the noAcceptPaths.
func isNoAcceptJSONWrite(r *http.Request) bool {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		return false
	}
	if r.Header.Get("Accept") != "" {
		return false
	}
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "json") {
		return false
	}
	for _, prefix := range noAcceptPaths {
		if strings.HasPrefix(r.URL.Path, prefix) {
			return true
		}
	}
	return false
}

// noAcceptResponseWriter drops the body of a successful response, like
// Cumulocity does for a POST or PUT request without an Accept header. Error
// responses keep their body.
type noAcceptResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *noAcceptResponseWriter) WriteHeader(status int) {
	w.status = status
	if status >= 200 && status < 300 {
		w.Header().Del("Content-Type")
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *noAcceptResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.status >= 200 && w.status < 300 {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

// URL returns the base URL of the fake server.
func (fs *FakeServer) URL() string {
	return fs.Server.URL
}

func (fs *FakeServer) registerRoutes(mux *http.ServeMux) {
	// We use a catch-all handler and route based on path prefix,
	// since Go 1.22+ ServeMux supports method+path patterns but we need
	// flexible sub-path matching for IDs.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		// Alarm endpoints
		case strings.HasPrefix(path, "/alarm/alarms"):
			fs.handleAlarms(w, r)

		// Event endpoints
		case strings.HasPrefix(path, "/event/events"):
			fs.handleEvents(w, r)

		// Operation endpoints (must be before /devicecontrol/ general)
		case strings.HasPrefix(path, "/devicecontrol/bulkoperations"):
			fs.handleBulkOperations(w, r)
		case strings.HasPrefix(path, "/devicecontrol/operations"):
			fs.handleOperations(w, r)
		case strings.HasPrefix(path, "/devicecontrol/newDeviceRequests"):
			fs.handleDeviceRequests(w, r)
		case strings.HasPrefix(path, "/devicecontrol/bulkNewDeviceRequests"):
			fs.handleBulkNewDeviceRequests(w, r)
		case strings.HasPrefix(path, "/devicecontrol/deviceCredentials"):
			fs.handleDeviceCredentials(w, r)

		// Measurement endpoints
		case strings.HasPrefix(path, "/measurement/measurements"):
			fs.handleMeasurements(w, r)

		// Inventory endpoints
		case strings.HasPrefix(path, "/inventory/binaries"):
			fs.handleBinaries(w, r)
		case strings.HasPrefix(path, "/inventory/managedObjects"):
			fs.handleManagedObjects(w, r)

		// Identity endpoints
		case strings.HasPrefix(path, "/identity/"):
			fs.handleIdentity(w, r)

		// Application endpoints
		case strings.HasPrefix(path, "/application/"):
			fs.handleApplications(w, r)

		// User endpoints
		case strings.HasPrefix(path, "/user/"):
			fs.handleUsers(w, r)

		// Tenant endpoints
		case strings.HasPrefix(path, "/tenant/"):
			fs.handleTenants(w, r)

		// Audit
		case strings.HasPrefix(path, "/audit/auditRecords"):
			fs.handleAuditRecords(w, r)

		// Retention rules
		case strings.HasPrefix(path, "/retention/retentions"):
			fs.handleRetentionRules(w, r)

		// Features
		case strings.HasPrefix(path, "/features"):
			fs.handleFeatures(w, r)

		// Realtime (Bayeux) websocket
		case path == "/cep/realtime":
			fs.handleRealtime(w, r)

		// Notification2 consumer websocket
		case strings.HasPrefix(path, "/notification2/consumer"):
			fs.handleNotification2Stream(w, r)

		// Notification2
		case strings.HasPrefix(path, "/notification2/"):
			fs.handleNotification2(w, r)

		// Remote access
		case strings.HasPrefix(path, "/service/remoteaccess/"):
			fs.handleRemoteAccess(w, r)

		// Certificate authority
		case strings.HasPrefix(path, "/certificate-authority"):
			fs.handleCertificateAuthority(w, r)

		// Device certificate enrollment (EST)
		case strings.HasPrefix(path, "/.well-known/est/"):
			fs.handleEST(w, r)

		default:
			// Like the platform's gateway, an unknown endpoint answers with an
			// HTML page, not the JSON error of a known resource.
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("<html>\n<head><title>404 Not Found</title></head>\n<body>\n<h1>404 Not Found</h1>\n</body>\n</html>\n"))
		}
	})
}

// seedDefaults creates baseline data that Cumulocity tenants always have.
func (fs *FakeServer) seedDefaults() {
	// Seed a current tenant
	fs.Tenants.CreateWithID("current", marshalJSON(map[string]any{
		"name":               "t12345",
		"id":                 "t12345",
		"self":               fs.URL() + "/tenant/currentTenant",
		"domain":             "t12345.cumulocity.com",
		"domainName":         "t12345.cumulocity.com",
		"status":             "ACTIVE",
		"allowCreateTenants": true,
	}))

	// Seed an admin user
	fs.Users.CreateWithID("admin", marshalJSON(map[string]any{
		"id":        "admin",
		"userName":  "admin",
		"email":     "admin@example.com",
		"firstName": "Admin",
		"lastName":  "User",
		"enabled":   true,
		"self":      fs.URL() + "/user/t12345/users/admin",
	}))

	// Seed the global roles which every tenant has
	for _, role := range []string{
		"ROLE_TENANT_ADMIN",
		"ROLE_ALARM_READ", "ROLE_ALARM_ADMIN",
		"ROLE_APPLICATION_MANAGEMENT_READ", "ROLE_APPLICATION_MANAGEMENT_ADMIN",
		"ROLE_AUDIT_READ", "ROLE_AUDIT_ADMIN",
		"ROLE_BULK_OPERATION_READ", "ROLE_BULK_OPERATION_ADMIN",
		"ROLE_DEVICE_CONTROL_READ", "ROLE_DEVICE_CONTROL_ADMIN",
		"ROLE_EVENT_READ", "ROLE_EVENT_ADMIN",
		"ROLE_IDENTITY_READ", "ROLE_IDENTITY_ADMIN",
		"ROLE_INVENTORY_READ", "ROLE_INVENTORY_ADMIN", "ROLE_INVENTORY_CREATE",
		"ROLE_MEASUREMENT_READ", "ROLE_MEASUREMENT_ADMIN",
		"ROLE_OPTION_MANAGEMENT_READ", "ROLE_OPTION_MANAGEMENT_ADMIN",
		"ROLE_TENANT_MANAGEMENT_READ", "ROLE_TENANT_MANAGEMENT_ADMIN",
		"ROLE_USER_MANAGEMENT_READ", "ROLE_USER_MANAGEMENT_ADMIN",
		"ROLE_USER_MANAGEMENT_CREATE", "ROLE_USER_MANAGEMENT_OWN_READ", "ROLE_USER_MANAGEMENT_OWN_ADMIN",
	} {
		fs.UserRoles.CreateWithID(role, marshalJSON(map[string]any{
			"id":   role,
			"name": role,
			"self": fs.URL() + "/user/roles/" + role,
		}))
	}

	// Seed feature toggles, stored by key (GET /features/{key}), as every
	// platform tenant has some
	for _, feature := range []map[string]any{
		{"key": "core.example.public", "phase": "PUBLIC_PREVIEW", "active": true, "strategy": "DEFAULT"},
		{"key": "core.example.private", "phase": "PRIVATE_PREVIEW", "active": false, "strategy": "TENANT"},
	} {
		fs.Features.CreateWithID(feature["key"].(string), marshalJSON(feature))
	}

	// Seed default login options (Basic)
	fs.LoginOptions.CreateWithID("BASIC", marshalJSON(map[string]any{
		"id":                   "BASIC",
		"type":                 "BASIC",
		"self":                 fs.URL() + "/tenant/loginOptions/BASIC",
		"grantType":            "PASSWORD",
		"userManagementSource": "INTERNAL",
		"visibleOnLoginPage":   true,
	}))
	// OAI-Secure (the default of a Cumulocity tenant): the token is created by
	// POST /tenant/oauth (see handleOAuth)
	fs.LoginOptions.CreateWithID("OAUTH2_INTERNAL", marshalJSON(map[string]any{
		"id":                   "OAUTH2_INTERNAL",
		"type":                 "OAUTH2_INTERNAL",
		"self":                 fs.URL() + "/tenant/loginOptions/OAUTH2_INTERNAL",
		"grantType":            "PASSWORD",
		"initRequest":          fs.URL() + "/tenant/oauth?tenant_id=t12345",
		"userManagementSource": "INTERNAL",
		"visibleOnLoginPage":   true,
	}))

	// Seed default user group
	fs.createUserGroup(marshalJSON(map[string]any{"name": "admins"}), "t12345")
	fs.createUserGroup(marshalJSON(map[string]any{"name": "devices"}), "t12345")

	// Seed a default retention rule (every tenant has at least one)
	fs.RetentionRules.Create(marshalJSON(map[string]any{
		"dataType":     "*",
		"fragmentType": "*",
		"type":         "*",
		"source":       "*",
		"maximumAge":   90,
		"editable":     true,
		"self":         fs.URL() + "/retention/retentions",
	}), fs.URL()+"/retention/retentions")

	// Seed system options (read-only platform config)
	for _, opt := range []struct{ category, key, value string }{
		{"system", "version", "1016.0.0"},
		{"configuration", "timezones.available", "true"},
		{"two-factor-authentication", "strategy", "SMS"},
	} {
		id := opt.category + "/" + opt.key
		fs.SystemOptions.CreateWithID(id, marshalJSON(map[string]any{
			"category": opt.category,
			"key":      opt.key,
			"value":    opt.value,
		}))
	}

	// Seed tenant options
	for _, opt := range []struct{ category, key, value string }{
		{"access.control", "allow.origin", "*"},
		{"two-factor-authentication", "enforce", "false"},
	} {
		id := opt.category + "/" + opt.key
		fs.TenantOptions.CreateWithID(id, marshalJSON(map[string]any{
			"category": opt.category,
			"key":      opt.key,
			"value":    opt.value,
			"self":     fs.URL() + "/tenant/options/" + opt.category + "/" + opt.key,
		}))
	}

	// Seed managed objects (needed for tests that expect pre-existing inventory)
	fs.ManagedObjects.Create(marshalJSON(map[string]any{
		"name":         "TestDevice001",
		"type":         "c8y_Linux",
		"c8y_IsDevice": map[string]any{},
	}), fs.URL()+"/inventory/managedObjects")

	// Seed thin-edge.io devices (needed for pagination/ForEach tests)
	for i := 1; i <= 3; i++ {
		fs.ManagedObjects.Create(marshalJSON(map[string]any{
			"name":         fmt.Sprintf("thin-edge-device-%03d", i),
			"type":         "thin-edge.io",
			"c8y_IsDevice": map[string]any{},
			"c8y_Agent":    map[string]any{"name": "thin-edge.io", "version": "1.0.0"},
		}), fs.URL()+"/inventory/managedObjects")
	}

	// Seed alarms (needed for alarm count tests)
	deviceID := "10001" // TestDevice001's ID
	fs.Alarms.Create(marshalJSON(map[string]any{
		"source":   map[string]any{"id": deviceID},
		"type":     "c8y_HealthCheck",
		"severity": "WARNING",
		"status":   "ACTIVE",
		"text":     "Seeded alarm 1",
		"time":     time.Now().UTC().Format(time.RFC3339),
	}), fs.URL()+"/alarm/alarms")
	fs.Alarms.Create(marshalJSON(map[string]any{
		"source":   map[string]any{"id": deviceID},
		"type":     "c8y_HealthCheck",
		"severity": "MAJOR",
		"status":   "ACTIVE",
		"text":     "Seeded alarm 2",
		"time":     time.Now().UTC().Format(time.RFC3339),
	}), fs.URL()+"/alarm/alarms")

	// Seed a CA trusted certificate
	fs.TrustedCerts.CreateWithID("ca-fingerprint-001", marshalJSON(map[string]any{
		"fingerprint":                "ca-fingerprint-001",
		"name":                       "Tenant CA Certificate",
		"status":                     "ENABLED",
		"autoRegistrationEnabled":    false,
		"algorithmName":              "RSA",
		"version":                    3,
		"proofOfPossessionValid":     true,
		"tenantCertificateAuthority": true,
		"notBefore":                  "2024-01-01T00:00:00Z",
		"notAfter":                   "2034-01-01T00:00:00Z",
		"self":                       fs.URL() + "/tenant/tenants/t12345/trusted-certificates/ca-fingerprint-001",
	}))

	// Seed standard applications that come with every Cumulocity tenant
	for _, app := range []struct {
		name, key, appType, contextPath string
	}{
		{"cockpit", "cockpit-application-key", "HOSTED", "cockpit"},
		{"devicemanagement", "devicemanagement-application-key", "HOSTED", "devicemanagement"},
		{"administration", "administration-application-key", "HOSTED", "administration"},
		{"report-agent", "report-agent-key", "MICROSERVICE", "reporting"},
	} {
		// The standard web applications are owned by the management tenant and
		// subscribed by the tenant, so they do not count as the tenant's own.
		owner := "t12345"
		if app.appType == "HOSTED" {
			owner = "management"
		}
		fs.Applications.Create(marshalJSON(map[string]any{
			"name":        app.name,
			"key":         app.key,
			"type":        app.appType,
			"contextPath": app.contextPath,
			"owner": map[string]any{
				"tenant": map[string]any{"id": owner},
				"self":   fs.URL() + "/tenant/tenants/" + owner,
			},
			"availability": "MARKET",
		}), fs.URL()+"/application/applications")
	}

	// Seed additional microservice applications (needed for tests expecting >= 10 microservices)
	msNames := []string{
		"device-simulator", "smartrule", "oee", "apama-ctrl",
		"cep", "sms-gateway", "cloud-remote-access", "fieldbus4",
		"opcua-mgmt-service", "loriot-agent", "snmp",
	}
	for _, name := range msNames {
		fs.Applications.Create(withMicroserviceRoles(marshalJSON(map[string]any{
			"name":        name,
			"key":         name + "-key",
			"type":        "MICROSERVICE",
			"contextPath": name,
			"owner": map[string]any{
				"tenant": map[string]any{"id": "t12345"},
				"self":   fs.URL() + "/tenant/tenants/t12345",
			},
			"availability": "MARKET",
		})), fs.URL()+"/application/applications")
	}

	// The shared plugin that every tenant can install into its web applications
	fs.seedSharedPlugin("Cumulocity community plugins", "sag-pkg-community-plugins", "1.2.3", "CommunityPluginsModule")

	// A tenant specific microservice, with a running instance
	fs.seedMicroserviceWithInstance("advanced-software-mgmt", "advanced-software-mgmt-scope-6b9d7c5f4-x2k8q")
}

// seedSharedPlugin adds a UI plugin of the management tenant which is shared
// with all tenants, with one version (tagged latest).
func (fs *FakeServer) seedSharedPlugin(name, contextPath, version string, modules ...string) {
	id, _ := fs.Applications.Create(marshalJSON(map[string]any{
		"name":         name,
		"key":          contextPath + "-key",
		"type":         "HOSTED",
		"contextPath":  contextPath,
		"availability": "SHARED",
		"manifest": map[string]any{
			"isPackage": true,
			"package":   "plugin",
			"remotes":   map[string]any{contextPath: modules},
		},
		"owner": map[string]any{
			"tenant": map[string]any{"id": "management"},
			"self":   fs.URL() + "/tenant/tenants/management",
		},
	}), fs.URL()+"/application/applications")
	fs.AppVersions.Create(marshalJSON(map[string]any{
		"applicationId": id,
		"version":       version,
		"tags":          []string{"latest"},
	}), fs.URL()+"/application/applications/"+id+"/versions")
}

// seedMicroserviceWithInstance adds a subscribed microservice together with the
// managed object (type c8y_Application_<id>) in which the platform tracks the
// status of its running instances (c8y_Status.instances).
func (fs *FakeServer) seedMicroserviceWithInstance(name string, instances ...string) {
	id, _ := fs.Applications.Create(withMicroserviceRoles(marshalJSON(map[string]any{
		"name":        name,
		"key":         name + "-key",
		"type":        "MICROSERVICE",
		"contextPath": name,
		"owner": map[string]any{
			"tenant": map[string]any{"id": "t12345"},
			"self":   fs.URL() + "/tenant/tenants/t12345",
		},
		"availability": "MARKET",
	})), fs.URL()+"/application/applications")

	status := map[string]any{}
	for _, instance := range instances {
		status[instance] = map[string]any{"status": "UP", "memory": 0, "cpu": 0}
	}
	fs.ManagedObjects.Create(marshalJSON(map[string]any{
		"type":       "c8y_Application_" + id,
		"name":       name,
		"c8y_Status": map[string]any{"status": "UP", "instances": status},
	}), fs.URL()+"/inventory/managedObjects")
}

func marshalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// Replay sends a RecordedRequest to the fake server and returns the response.
// This is used by golden-file comparison tests to replay live-recorded requests
// against the fake server without going through the SDK client.
func (fs *FakeServer) Replay(req RecordedRequest) RecordedResponse {
	// Build the URL with query parameters
	u := fs.URL() + req.Path
	if len(req.Query) > 0 {
		params := make([]string, 0, len(req.Query))
		for k, v := range req.Query {
			params = append(params, k+"="+v)
		}
		u += "?" + strings.Join(params, "&")
	}

	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}

	httpReq, err := http.NewRequest(req.Method, u, body)
	if err != nil {
		return RecordedResponse{StatusCode: 0}
	}
	httpReq.Header.Set("Authorization", "Basic dGVzdDp0ZXN0") // dummy auth
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return RecordedResponse{StatusCode: 0}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	var respJSON json.RawMessage
	if json.Valid(respBody) && len(respBody) > 0 {
		respJSON = respBody
	}

	return RecordedResponse{
		StatusCode: resp.StatusCode,
		Body:       respJSON,
	}
}

// credentialsValid reports whether the basic credentials of the request match
// the password of a user the server knows the password of (the seeded admin
// and the users created with a password). Other requests, such as those of
// unknown users, bearer tokens and device credentials, are accepted.
func (fs *FakeServer) credentialsValid(r *http.Request) bool {
	user, password, ok := r.BasicAuth()
	if !ok {
		return true
	}
	// The user name may carry the tenant: <tenant>/<user>
	if i := strings.LastIndex(user, "/"); i >= 0 {
		user = user[i+1:]
	}
	fs.passwordsMu.RLock()
	expected, known := fs.passwords[user]
	fs.passwordsMu.RUnlock()
	return !known || expected == password
}

// storePassword records the password of a user and returns the user document
// without it.
func (fs *FakeServer) storePassword(userName string, doc json.RawMessage) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(doc, &m); err != nil {
		return doc
	}
	raw, ok := m["password"]
	if !ok {
		return doc
	}
	var password string
	if json.Unmarshal(raw, &password) == nil && password != "" && userName != "" {
		fs.passwordsMu.Lock()
		fs.passwords[userName] = password
		fs.passwordsMu.Unlock()
	}
	delete(m, "password")
	out, err := json.Marshal(m)
	if err != nil {
		return doc
	}
	return out
}
