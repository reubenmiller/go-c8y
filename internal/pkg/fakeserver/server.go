package fakeserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

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

	// Group membership: groupID → list of userIDs
	GroupMembers map[string][]string
}

// New creates a new FakeServer with an httptest.Server and registers cleanup.
func New(t *testing.T) *FakeServer {
	fs := &FakeServer{
		Alarms:                     NewStore(),
		Events:                     NewStore(),
		Operations:                 NewStore(),
		Measurements:               NewStore(),
		ManagedObjects:             NewStore(),
		Binaries:                   NewStore(),
		Applications:               NewStore(),
		AppVersions:                NewStore(),
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

		ChildDevices:   make(map[string][]string),
		ChildAssets:    make(map[string][]string),
		ChildAdditions: make(map[string][]string),
		EventBinaries:  make(map[string][]byte),
		BinaryData:     make(map[string][]byte),
		GroupMembers:   make(map[string][]string),
	}

	mux := http.NewServeMux()
	fs.registerRoutes(mux)

	// Wrap with basic auth check – returns 401 when no credentials are supplied.
	// Some endpoints are public (no auth required).
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		public := strings.HasPrefix(p, "/tenant/loginOptions") ||
			strings.HasPrefix(p, "/tenant/oauth")
		if !public && r.Header.Get("Authorization") == "" {
			writeError(w, http.StatusUnauthorized, "security/Unauthorized", "Missing credentials")
			return
		}
		mux.ServeHTTP(w, r)
	})

	fs.Server = httptest.NewServer(handler)
	t.Cleanup(fs.Server.Close)

	// Seed default data (current tenant, current user)
	fs.seedDefaults()

	return fs
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

		// Notification2
		case strings.HasPrefix(path, "/notification2/"):
			fs.handleNotification2(w, r)

		// Remote access
		case strings.HasPrefix(path, "/service/remoteaccess/"):
			fs.handleRemoteAccess(w, r)

		// Certificate authority
		case strings.HasPrefix(path, "/certificate-authority"):
			fs.handleCertificateAuthority(w, r)

		default:
			writeError(w, http.StatusNotFound, "general/notFound", "Endpoint not found: "+path)
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

	// Seed a default admin role
	fs.UserRoles.CreateWithID("ROLE_TENANT_ADMIN", marshalJSON(map[string]any{
		"id":   "ROLE_TENANT_ADMIN",
		"name": "ROLE_TENANT_ADMIN",
		"self": fs.URL() + "/user/roles/ROLE_TENANT_ADMIN",
	}))

	// Seed default login options (Basic)
	fs.LoginOptions.CreateWithID("BASIC", marshalJSON(map[string]any{
		"id":                   "BASIC",
		"type":                 "BASIC",
		"self":                 fs.URL() + "/tenant/loginOptions/BASIC",
		"grantType":            "PASSWORD",
		"userManagementSource": "INTERNAL",
		"visibleOnLoginPage":   true,
	}))

	// Seed default user group
	fs.UserGroups.Create(marshalJSON(map[string]any{
		"name": "admins",
	}), fs.URL()+"/user/t12345/groups")
	fs.UserGroups.Create(marshalJSON(map[string]any{
		"name": "devices",
	}), fs.URL()+"/user/t12345/groups")

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
		{"reporting", "reporting-key", "MICROSERVICE", "reporting"},
	} {
		fs.Applications.Create(marshalJSON(map[string]any{
			"name":        app.name,
			"key":         app.key,
			"type":        app.appType,
			"contextPath": app.contextPath,
			"owner": map[string]any{
				"tenant": map[string]any{"id": "t12345"},
				"self":   fs.URL() + "/tenant/tenants/t12345",
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
		fs.Applications.Create(marshalJSON(map[string]any{
			"name":        name,
			"key":         name + "-key",
			"type":        "MICROSERVICE",
			"contextPath": name,
			"owner": map[string]any{
				"tenant": map[string]any{"id": "t12345"},
				"self":   fs.URL() + "/tenant/tenants/t12345",
			},
			"availability": "MARKET",
		}), fs.URL()+"/application/applications")
	}
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
