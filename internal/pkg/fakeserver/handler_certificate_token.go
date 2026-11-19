package fakeserver

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/jwe"
)

// handleCertificateToken mimics POST /tenant/oauth/certificate: it reads the
// PEM chain, requires tenant_id and an RSA leaf key, and returns an access
// token (a JWT) encrypted as a JWE to the leaf certificate's public key.
func (fs *FakeServer) handleCertificateToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}
	tenant := r.URL.Query().Get("tenant_id")
	if tenant == "" {
		tenant = r.Header.Get("X-Cumulocity-TenantId")
	}
	if tenant == "" {
		writeError(w, http.StatusBadRequest, "security/badRequest", "The tenant could not be determined")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
		return
	}
	block, _ := pem.Decode(body)
	if block == nil || block.Type != "CERTIFICATE" {
		writeError(w, http.StatusBadRequest, "security/badRequest", "Unable to parse certificate")
		return
	}
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		writeError(w, http.StatusBadRequest, "security/badRequest", "Unable to parse certificate")
		return
	}
	pub, ok := leaf.PublicKey.(*rsa.PublicKey)
	if !ok {
		writeError(w, http.StatusUnprocessableEntity, "security/unsupportedKey", "The certificate key must be RSA")
		return
	}

	b64 := base64.RawURLEncoding
	header := b64.EncodeToString([]byte(`{"alg":"RS256","typ":"JWT"}`))
	claims, _ := json.Marshal(map[string]any{
		"sub": leaf.Subject.CommonName,
		"ten": tenant,
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	jwt := header + "." + b64.EncodeToString(claims) + ".fakesig"

	token, err := jwe.Encrypt([]byte(jwt), pub)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "general/internalError", fmt.Sprintf("encrypt token: %v", err))
		return
	}
	writeJSON(w, http.StatusOK, marshalJSON(map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   3600,
	}))
}
