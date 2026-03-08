package fakeserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// handleTenants routes /tenant/ requests.
func (fs *FakeServer) handleTenants(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasPrefix(path, "/tenant/currentTenant"):
		fs.handleCurrentTenant(w, r)

	case strings.HasPrefix(path, "/tenant/loginOptions"):
		fs.handleLoginOptions(w, r)

	case strings.HasPrefix(path, "/tenant/oauth"):
		fs.handleOAuth(w, r)

	case strings.HasPrefix(path, "/tenant/statistics"):
		fs.handleTenantStatistics(w, r)

	case strings.HasPrefix(path, "/tenant/system/options"):
		fs.handleSystemOptions(w, r)

	case strings.HasPrefix(path, "/tenant/options"):
		fs.handleTenantOptions(w, r)

	case strings.HasPrefix(path, "/tenant/trusted-certificates/settings/crl"):
		fs.handleCertificateRevocationList(w, r)

	case strings.HasPrefix(path, "/tenant/tenants"):
		fs.handleTenantsCRUD(w, r)

	default:
		writeNotFound(w, "tenant")
	}
}

func (fs *FakeServer) handleCurrentTenant(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.Tenants.Get("current")
		if !ok {
			writeNotFound(w, "tenant")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, _ := fs.Tenants.Update("current", body)
		writeJSON(w, http.StatusOK, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleTenantsCRUD(w http.ResponseWriter, r *http.Request) {
	segments := extractPathSegments(r.URL.Path, "/tenant/tenants")

	if len(segments) == 0 {
		// /tenant/tenants (collection)
		switch r.Method {
		case http.MethodGet:
			items := fs.Tenants.List()
			page := Paginate(r, items)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "tenants", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			_, doc := fs.Tenants.Create(body, fs.URL()+"/tenant/tenants")
			writeJSON(w, http.StatusCreated, doc)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	tenantID := segments[0]

	// /tenant/tenants/{id}/trusted-certificates-pop/{fingerprint}/...
	if len(segments) >= 2 && segments[1] == "trusted-certificates-pop" {
		fs.handleTrustedCertificatesPop(w, r, tenantID, segments[2:])
		return
	}

	// /tenant/tenants/{id}/trusted-certificates
	if len(segments) >= 2 && segments[1] == "trusted-certificates" {
		fs.handleTrustedCertificates(w, r, tenantID, segments[2:])
		return
	}

	// /tenant/tenants/{id}/tfa
	if len(segments) >= 2 && segments[1] == "tfa" {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, marshalJSON(map[string]any{
				"strategy":       "SMS",
				"tfaEnforced":    false,
				"lastTfaRequest": "",
			}))
		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			writeJSON(w, http.StatusOK, body)
		case http.MethodDelete:
			writeNoContent(w)
		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /tenant/tenants/{id}
	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.Tenants.Get(tenantID)
		if !ok {
			writeNotFound(w, "tenant")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, ok := fs.Tenants.Update(tenantID, body)
		if !ok {
			writeNotFound(w, "tenant")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodDelete:
		if !fs.Tenants.Delete(tenantID) {
			writeNotFound(w, "tenant")
			return
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleTenantOptions(w http.ResponseWriter, r *http.Request) {
	segments := extractPathSegments(r.URL.Path, "/tenant/options")

	if len(segments) == 0 {
		// /tenant/options (collection)
		switch r.Method {
		case http.MethodGet:
			items := fs.TenantOptions.List()
			page := Paginate(r, items)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "options", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			category := getJSONString(body, "category")
			key := getJSONString(body, "key")
			compID := category + "/" + key
			body = mergeFields(body, map[string]any{
				"self": fs.URL() + "/tenant/options/" + compID,
			})
			fs.TenantOptions.CreateWithID(compID, body)
			writeJSON(w, http.StatusOK, body)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	if len(segments) == 1 {
		// /tenant/options/{category}
		category := segments[0]
		switch r.Method {
		case http.MethodGet:
			// Return flat key:value map for the category
			result := map[string]any{}
			for _, doc := range fs.TenantOptions.List() {
				if getJSONString(doc, "category") == category {
					result[getJSONString(doc, "key")] = getJSONString(doc, "value")
				}
			}
			writeJSON(w, http.StatusOK, marshalJSON(result))
		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			// Body is a flat map like {"prop1":"value1","prop2":"value2"}
			var kvPairs map[string]string
			json.Unmarshal(body, &kvPairs)
			for k, v := range kvPairs {
				compID := category + "/" + k
				optDoc := marshalJSON(map[string]any{
					"category": category,
					"key":      k,
					"value":    v,
					"self":     fs.URL() + "/tenant/options/" + compID,
				})
				fs.TenantOptions.CreateWithID(compID, optDoc)
			}
			// Return flat key:value map
			result := map[string]any{}
			for _, doc := range fs.TenantOptions.List() {
				if getJSONString(doc, "category") == category {
					result[getJSONString(doc, "key")] = getJSONString(doc, "value")
				}
			}
			writeJSON(w, http.StatusOK, marshalJSON(result))
		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /tenant/options/{category}/{key}
	compID := segments[0] + "/" + segments[1]

	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.TenantOptions.Get(compID)
		if !ok {
			writeNotFound(w, "option")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, ok := fs.TenantOptions.Update(compID, body)
		if !ok {
			// Create on PUT if not exists
			body = mergeFields(body, map[string]any{
				"self": fs.URL() + "/tenant/options/" + compID,
			})
			fs.TenantOptions.CreateWithID(compID, body)
			doc = body
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodDelete:
		if !fs.TenantOptions.Delete(compID) {
			writeNotFound(w, "option")
			return
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleSystemOptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}

	// GET /tenant/system/options/{category}/{key}
	path := strings.TrimPrefix(r.URL.Path, "/tenant/system/options")
	path = strings.TrimPrefix(path, "/")
	if path != "" {
		// Lookup by category/key
		doc, ok := fs.SystemOptions.Get(path)
		if !ok {
			writeNotFound(w, "option")
			return
		}
		writeJSON(w, http.StatusOK, doc)
		return
	}

	// List all system options
	items := fs.SystemOptions.List()
	resp := marshalJSON(map[string]any{
		"options": items,
		"self":    fs.URL() + "/tenant/system/options",
		"statistics": map[string]int{
			"currentPage":   1,
			"pageSize":      len(items),
			"totalPages":    1,
			"totalElements": len(items),
		},
	})
	writeJSON(w, http.StatusOK, resp)
}

func (fs *FakeServer) handleLoginOptions(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/tenant/loginOptions")

	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.LoginOptions.Get(id)
			if !ok {
				writeNotFound(w, "loginOption")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.LoginOptions.Update(id, body)
			if !ok {
				writeNotFound(w, "loginOption")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.LoginOptions.Delete(id) {
				writeNotFound(w, "loginOption")
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
		items := fs.LoginOptions.List()
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "loginOptions", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		_, doc := fs.LoginOptions.Create(body, fs.URL()+"/tenant/loginOptions")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) handleTenantStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}

	path := r.URL.Path

	if strings.HasSuffix(path, "/summary") || strings.HasSuffix(path, "/allTenantsSummary") {
		resp := marshalJSON(map[string]any{
			"self":                    fs.URL() + path,
			"day":                     "2024-01-01",
			"deviceCount":             1,
			"requestCount":            42,
			"storageSize":             1024,
			"inventoriesCreatedCount": 5,
		})
		writeJSON(w, http.StatusOK, resp)
		return
	}

	// /tenant/statistics — collection of daily stats
	day := marshalJSON(map[string]any{
		"day":                     "2024-01-01",
		"deviceCount":             1,
		"requestCount":            42,
		"storageSize":             1024,
		"inventoriesCreatedCount": 5,
	})
	items := []json.RawMessage{day}
	page := Paginate(r, items)
	writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "usageStatistics", page))
}

func (fs *FakeServer) handleTrustedCertificates(w http.ResponseWriter, r *http.Request, tenantID string, rest []string) {
	if len(rest) == 0 {
		// /tenant/tenants/{id}/trusted-certificates
		switch r.Method {
		case http.MethodGet:
			items := fs.TrustedCerts.List()

			// Filter by certificateAuthority query param
			if caStr := r.URL.Query().Get("certificateAuthority"); caStr == "true" {
				var filtered []json.RawMessage
				for _, item := range items {
					if getJSONString(item, "tenantCertificateAuthority") == "true" {
						filtered = append(filtered, item)
					}
				}
				items = filtered
			}

			page := Paginate(r, items)
			writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "certificates", page))

		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			fingerprint := getJSONString(body, "fingerprint")
			if fingerprint == "" {
				_, doc := fs.TrustedCerts.Create(body, fs.URL()+"/tenant/tenants/"+tenantID+"/trusted-certificates")
				writeJSON(w, http.StatusCreated, doc)
				return
			}
			// Use fingerprint as the ID
			body = mergeFields(body, map[string]any{
				"self": fs.URL() + "/tenant/tenants/" + tenantID + "/trusted-certificates/" + fingerprint,
				"proofOfPossessionUnsignedVerificationCode":    "proof-verification-code-" + fingerprint[:8],
				"proofOfPossessionValid":                       false,
				"proofOfPossessionVerificationCodeUsableUntil": "2099-01-01T00:00:00Z",
			})
			fs.TrustedCerts.CreateWithID(fingerprint, body)
			writeJSON(w, http.StatusCreated, body)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	certID := rest[0]

	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.TrustedCerts.Get(certID)
		if !ok {
			writeNotFound(w, "certificate")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, ok := fs.TrustedCerts.Update(certID, body)
		if !ok {
			writeNotFound(w, "certificate")
			return
		}
		writeJSON(w, http.StatusOK, doc)

	case http.MethodDelete:
		if !fs.TrustedCerts.Delete(certID) {
			writeNotFound(w, "certificate")
			return
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleOAuth handles /tenant/oauth/* endpoints (token creation).
func (fs *FakeServer) handleOAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", "Invalid form data")
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	// Simple credential check — accept the seeded admin user.
	if username != "admin" || password != "admin-pass" {
		writeError(w, http.StatusUnauthorized, "security/Unauthorized", "Invalid credentials")
		return
	}

	// Return a fake OAI-Secure token (JWT with sub, ten, xsrfToken claims)
	writeJSON(w, http.StatusOK, marshalJSON(map[string]any{
		"access_token": "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiAiYWRtaW4iLCAidGVuIjogInQxMjM0NSIsICJ4c3JmVG9rZW4iOiAiZmFrZVhTUkZUb2tlbjEyMyIsICJpc3MiOiAiaHR0cDovL2xvY2FsaG9zdCJ9.fakesig",
		"token_type":   "Bearer",
		"expires_in":   604800,
	}))
}

// handleTrustedCertificatesPop handles /tenant/tenants/{id}/trusted-certificates-pop/{fingerprint}/...
func (fs *FakeServer) handleTrustedCertificatesPop(w http.ResponseWriter, r *http.Request, tenantID string, rest []string) {
	if len(rest) < 2 {
		writeError(w, http.StatusNotFound, "general/notFound", "Endpoint not found")
		return
	}
	fingerprint := rest[0]
	action := rest[1]

	cert, ok := fs.TrustedCerts.Get(fingerprint)
	if !ok {
		writeNotFound(w, "certificate")
		return
	}

	switch action {
	case "verification-code":
		// POST /tenant/tenants/{id}/trusted-certificates-pop/{fingerprint}/verification-code
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			return
		}
		// Return cert with a fresh verification code
		doc := mergeFields(cert, map[string]any{
			"proofOfPossessionUnsignedVerificationCode":    "proof-verification-code-" + fingerprint[:8],
			"proofOfPossessionVerificationCodeUsableUntil": "2099-01-01T00:00:00Z",
		})
		fs.TrustedCerts.Update(fingerprint, doc)
		writeJSON(w, http.StatusOK, doc)

	case "pop":
		// POST /tenant/tenants/{id}/trusted-certificates-pop/{fingerprint}/pop
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
			return
		}
		// Accept any signed verification code and mark as valid
		doc := mergeFields(cert, map[string]any{
			"proofOfPossessionValid": true,
		})
		fs.TrustedCerts.Update(fingerprint, doc)
		writeJSON(w, http.StatusOK, doc)

	default:
		writeError(w, http.StatusNotFound, "general/notFound", "Endpoint not found: "+action)
	}
}

// handleCertificateAuthority handles /certificate-authority endpoints
func (fs *FakeServer) handleCertificateAuthority(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		// Check if a CA certificate already exists
		for _, item := range fs.TrustedCerts.List() {
			if getJSONString(item, "tenantCertificateAuthority") == "true" {
				writeError(w, http.StatusConflict, "certificate/conflict", "Certificate authority already exists")
				return
			}
		}
		// No CA cert exists — create one
		writeJSON(w, http.StatusCreated, marshalJSON(map[string]any{
			"fingerprint":                "auto-generated-ca",
			"name":                       "Tenant CA Certificate",
			"status":                     "ENABLED",
			"tenantCertificateAuthority": true,
			"self":                       fs.URL() + "/tenant/tenants/t12345/trusted-certificates/auto-generated-ca",
		}))

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleCertificateRevocationList handles /tenant/trusted-certificates/settings/crl
func (fs *FakeServer) handleCertificateRevocationList(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return a minimal DER-encoded CRL
		w.Header().Set("Content-Type", "application/pkix-crl")
		w.WriteHeader(http.StatusOK)
		// Minimal ASN.1 CRL structure (empty revocation list)
		// This is a DER-encoded X.509 CRL with no revoked certs
		crl := fs.generateMinimalCRL()
		w.Write(crl)

	case http.MethodPut:
		// Accept CRL additions (JSON or multipart)
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

func (fs *FakeServer) generateMinimalCRL() []byte {
	// Generate a proper self-signed CRL using Go's crypto libraries
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil
	}
	now := time.Now()
	issuerTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Fake CA",
		},
		NotBefore: now.Add(-time.Hour),
		NotAfter:  now.Add(24 * 365 * time.Hour),
		IsCA:      true,
		KeyUsage:  x509.KeyUsageCRLSign | x509.KeyUsageCertSign,
	}
	issuerDER, err := x509.CreateCertificate(rand.Reader, issuerTemplate, issuerTemplate, &key.PublicKey, key)
	if err != nil {
		return nil
	}
	issuer, err := x509.ParseCertificate(issuerDER)
	if err != nil {
		return nil
	}
	template := &x509.RevocationList{
		Number:     big.NewInt(1),
		ThisUpdate: now,
		NextUpdate: now.Add(24 * time.Hour),
	}
	crlBytes, err := x509.CreateRevocationList(rand.Reader, template, issuer, key)
	if err != nil {
		return nil
	}
	return crlBytes
}
