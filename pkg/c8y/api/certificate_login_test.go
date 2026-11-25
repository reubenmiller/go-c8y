package api

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/jwe"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/currentuser"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/currentuser/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testCert struct {
	certPEM []byte
	keyPEM  []byte
	key     crypto.Signer
}

func newTestCert(t *testing.T, key crypto.Signer, cn string, parent *x509.Certificate, parentKey crypto.Signer) (testCert, *x509.Certificate) {
	t.Helper()
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().UnixNano()),
		Subject:               pkix.Name{CommonName: cn},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  parent == nil,
		BasicConstraintsValid: true,
	}
	signer, issuer := key, tmpl
	if parent != nil {
		signer, issuer = parentKey, parent
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, issuer, key.Public(), signer)
	require.NoError(t, err)
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	parsed, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return testCert{
		certPEM: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		keyPEM:  pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER}),
		key:     key,
	}, parsed
}

func rsaCert(t *testing.T) testCert {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	c, _ := newTestCert(t, k, "user1", nil, nil)
	return c
}

func ecCert(t *testing.T) testCert {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	c, _ := newTestCert(t, k, "device1", nil, nil)
	return c
}

func fakeJWT(tenant string, exp time.Time) string {
	b64 := base64.RawURLEncoding
	claims, _ := json.Marshal(map[string]any{"ten": tenant, "sub": "user1", "exp": exp.Unix()})
	return b64.EncodeToString([]byte(`{"alg":"RS256"}`)) + "." + b64.EncodeToString(claims) + ".sig"
}

// certServer records requests and answers the certificate exchange with a JWE
// encrypted to the presented RSA key.
type certServer struct {
	mu        sync.Mutex
	requests  []recorded
	auth      []string
	server    *httptest.Server
	jwt       string
	expiresIn int
	// tfaFirst makes the first exchange answer with a TFA challenge.
	tfaFirst  bool
	tfaHeader string
	// tamper changes the returned JWE.
	tamper func(string) string
	// verifyOK is the verify-cert-chain answer.
	verifyOK bool
}

func newCertServer(t *testing.T, pub *rsa.PublicKey) *certServer {
	t.Helper()
	cs := &certServer{jwt: fakeJWT("t100", time.Now().Add(time.Hour)), expiresIn: 3600, verifyOK: true}
	cs.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		cs.mu.Lock()
		cs.requests = append(cs.requests, recorded{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(b)})
		cs.auth = append(cs.auth, r.Header.Get("Authorization"))
		exchanges := 0
		for _, req := range cs.requests {
			if req.Path == "/tenant/oauth/certificate" {
				exchanges++
			}
		}
		cs.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/tenant/oauth/certificate":
			if cs.tfaFirst && exchanges == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"security/Unauthorized","message":"TFA TOTP code required"}`)) //nolint:misspell // platform error type
				return
			}
			cs.mu.Lock()
			cs.tfaHeader = r.Header.Get("X-Cumulocity-TFA-Code")
			cs.mu.Unlock()
			token, err := jwe.Encrypt([]byte(cs.jwt), pub)
			require.NoError(t, err)
			if cs.tamper != nil {
				token = cs.tamper(token)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": token, "token_type": "Bearer", "expires_in": cs.expiresIn})
		case "/devicecontrol/deviceAccessToken":
			_ = json.NewEncoder(w).Encode(map[string]any{"accessToken": cs.jwt})
		case "/tenant/trusted-certificates/verify-cert-chain":
			if !cs.verifyOK {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"security/Unauthorized","message":"chain not trusted"}`)) //nolint:misspell // platform error type
				return
			}
			_, _ = w.Write([]byte(`{"successfullyValidated":true,"tenantId":"t100"}`))
		default:
			_, _ = w.Write([]byte(`{"managedObjects":[],"statistics":{"currentPage":1,"pageSize":5}}`))
		}
	}))
	t.Cleanup(cs.server.Close)
	return cs
}

// reset forgets recorded requests, e.g. the eager login NewClient performs
// when a certificate is configured.
func (cs *certServer) reset() {
	cs.mu.Lock()
	cs.requests, cs.auth = nil, nil
	cs.mu.Unlock()
}

func (cs *certServer) paths() []string {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var out []string
	for _, r := range cs.requests {
		out = append(out, r.Method+" "+r.Path)
	}
	return out
}

func (cs *certServer) find(path string) []recorded {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	var out []recorded
	for _, r := range cs.requests {
		if r.Path == path {
			out = append(out, r)
		}
	}
	return out
}

func serverPort(t *testing.T, s *httptest.Server) string {
	u, err := url.Parse(s.URL)
	require.NoError(t, err)
	_, port, err := net.SplitHostPort(u.Host)
	require.NoError(t, err)
	return port
}

func certClient(cs *certServer, cert testCert, transport authentication.CertificateTransport, mtlsPort string) *Client {
	c := NewClient(ClientOptions{
		BaseURL:          cs.server.URL,
		SkipTenantLookup: true,
		MTLSPort:         mtlsPort,
		Auth: authentication.AuthOptions{
			Tenant:               "t100",
			Certificate:          string(cert.certPEM),
			CertificateKey:       string(cert.keyPEM),
			CertificateTransport: transport,
		},
	})
	cs.reset()
	return c
}

func Test_CertLogin_RSAUsesHTTPS(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	client := certClient(cs, cert, authentication.CertificateTransportAuto, "1")

	token, err := client.LoginWithOptions(context.Background(), LoginOptions{Method: authentication.LoginMethodCertificate})
	require.NoError(t, err)
	assert.Equal(t, cs.jwt, token, "the decrypted JWT is the token")
	require.Len(t, cs.find("/tenant/oauth/certificate"), 1)
	assert.Equal(t, "tenant_id=t100", cs.find("/tenant/oauth/certificate")[0].Query)
	assert.Empty(t, cs.find("/devicecontrol/deviceAccessToken"))

	// Later API requests carry the decrypted token.
	require.NoError(t, get(client, context.Background()).Error)
	cs.mu.Lock()
	last := cs.auth[len(cs.auth)-1]
	cs.mu.Unlock()
	assert.Equal(t, "Bearer "+cs.jwt, last)
}

func Test_CertLogin_ECUsesMTLS(t *testing.T) {
	cert := ecCert(t)
	cs := newCertServer(t, nil)
	client := certClient(cs, cert, authentication.CertificateTransportAuto, serverPort(t, cs.server))

	_, err := client.LoginWithOptions(context.Background(), LoginOptions{Method: authentication.LoginMethodCertificate})
	require.NoError(t, err)
	assert.Len(t, cs.find("/devicecontrol/deviceAccessToken"), 1)
	assert.Empty(t, cs.find("/tenant/oauth/certificate"))
}

func Test_CertLogin_NoFallbackWhenMTLSPortUnreachable(t *testing.T) {
	cert := ecCert(t)
	cs := newCertServer(t, nil)
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	_, closedPort, _ := net.SplitHostPort(l.Addr().String())
	require.NoError(t, l.Close())
	client := certClient(cs, cert, authentication.CertificateTransportAuto, closedPort)
	client.Resty().SetRetryCount(0)

	_, err = client.LoginWithOptions(context.Background(), LoginOptions{Method: authentication.LoginMethodCertificate})
	require.Error(t, err)
	assert.Contains(t, err.Error(), closedPort)
	assert.Contains(t, err.Error(), "MTLSPort")
	assert.Empty(t, cs.find("/tenant/oauth/certificate"), "no HTTPS fallback")
}

func Test_CertLogin_ForcedHTTPSWithECFails(t *testing.T) {
	cert := ecCert(t)
	cs := newCertServer(t, nil)
	client := certClient(cs, cert, authentication.CertificateTransportHTTPS, "")

	_, err := client.LoginWithOptions(context.Background(), LoginOptions{Method: authentication.LoginMethodCertificate})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RSA")
	assert.Empty(t, cs.paths(), "nothing is sent")
}

func Test_CertLogin_RequestFormat(t *testing.T) {
	caKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ca, caCert := newTestCert(t, caKey, "intermediate", nil, nil)
	leafKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	leaf, _ := newTestCert(t, leafKey, "user1", caCert, caKey)
	chain := testCert{certPEM: append(append([]byte{}, leaf.certPEM...), ca.certPEM...), keyPEM: leaf.keyPEM, key: leafKey}

	cs := newCertServer(t, &leafKey.PublicKey)
	client := certClient(cs, chain, authentication.CertificateTransportAuto, "")
	client.Auth.Tenant = "t123"
	client.SetAuth(client.Auth)
	cs.reset()

	_, err := client.LoginWithOptions(context.Background(), LoginOptions{Method: authentication.LoginMethodCertificate})
	require.NoError(t, err)
	reqs := cs.find("/tenant/oauth/certificate")
	require.Len(t, reqs, 1)
	assert.Equal(t, "tenant_id=t123", reqs[0].Query)
	assert.Equal(t, 2, strings.Count(reqs[0].Body, "BEGIN CERTIFICATE"), "leaf and intermediate are sent")
	cs.mu.Lock()
	assert.Empty(t, cs.auth[0], "no Authorization header on the exchange")
	cs.mu.Unlock()
}

func Test_CertLogin_UnsupportedAlgorithm(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	cs.tamper = func(token string) string {
		parts := strings.Split(token, ".")
		parts[0] = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RSA1_5","enc":"A256GCM"}`))
		return strings.Join(parts, ".")
	}
	client := certClient(cs, cert, authentication.CertificateTransportAuto, "")

	_, err := client.LoginWithOptions(context.Background(), LoginOptions{Method: authentication.LoginMethodCertificate})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RSA1_5")
}

func Test_CertLogin_TamperedToken(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	cs.tamper = func(token string) string {
		parts := strings.Split(token, ".")
		ct, _ := base64.RawURLEncoding.DecodeString(parts[3])
		ct[0] ^= 0xff
		parts[3] = base64.RawURLEncoding.EncodeToString(ct)
		return strings.Join(parts, ".")
	}
	client := certClient(cs, cert, authentication.CertificateTransportAuto, "")

	_, err := client.LoginWithOptions(context.Background(), LoginOptions{Method: authentication.LoginMethodCertificate})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decrypt")
	assert.Empty(t, client.Auth.Token, "no token is stored")
}

func Test_CertLogin_TOTPChallenge(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	cs.tfaFirst = true
	client := certClient(cs, cert, authentication.CertificateTransportAuto, "")

	_, err := client.LoginWithOptions(context.Background(), LoginOptions{
		Method:   authentication.LoginMethodCertificate,
		TOTPCode: func(ctx context.Context, _ totp.TOTPChallenge) (string, error) { return "123456", nil },
	})
	require.NoError(t, err)
	assert.Len(t, cs.find("/tenant/oauth/certificate"), 2)
	cs.mu.Lock()
	assert.Equal(t, "123456", cs.tfaHeader)
	cs.mu.Unlock()
}

func Test_CertLogin_RefreshUsesSameTransportAndExpiresIn(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	cs.expiresIn = 60
	client := certClient(cs, cert, authentication.CertificateTransportAuto, "")

	tok, err := client.fetchToken(context.Background())
	require.NoError(t, err)
	require.NotNil(t, tok)
	assert.WithinDuration(t, time.Now().Add(60*time.Second), tok.Expiry, 5*time.Second)
	_, err = client.fetchToken(context.Background())
	require.NoError(t, err)
	assert.Len(t, cs.find("/tenant/oauth/certificate"), 2, "each refresh is a new HTTPS exchange")
}

func Test_CertOptIn_Helpers(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) { return 200, `{}` })
	client := newScriptClient(ss)

	require.NoError(t, client.Users.Current.SetRequiredAuthType(context.Background(), currentuser.AuthTypeCertificates).Err)
	require.NoError(t, client.Applications.Current.SetAuthType(context.Background(), currentuser.AuthTypeCertificates).Err)
	require.NoError(t, client.Applications.ResetAuthType(context.Background(), "42").Err)

	user := ss.matching(http.MethodPut, "/user/currentUser")
	require.Len(t, user, 1)
	assert.JSONEq(t, `{"requiredAuthType":"CERTIFICATES"}`, user[0].Body)
	app := ss.matching(http.MethodPut, "/application/currentApplication/authType")
	require.Len(t, app, 1)
	assert.JSONEq(t, `{"requiredAuthType":"CERTIFICATES"}`, app[0].Body)
	reset := ss.matching(http.MethodPut, "/application/applications/42/authType")
	require.Len(t, reset, 1)
	assert.JSONEq(t, `{"requiredAuthType":"NONE"}`, reset[0].Body)
}

func Test_EnableCertificateAuth_UntrustedChainChangesNothing(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	cs.verifyOK = false
	client := certClient(cs, cert, authentication.CertificateTransportAuto, "")

	err := client.EnableCertificateAuth(context.Background(), EnableCertificateAuthOptions{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCertificateNotTrusted)
	assert.Empty(t, cs.find("/user/currentUser"), "requiredAuthType must not change")
	assert.Empty(t, cs.find("/tenant/oauth/certificate"))
}

func Test_EnableCertificateAuth_Success(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	client := certClient(cs, cert, authentication.CertificateTransportAuto, "")

	require.NoError(t, client.EnableCertificateAuth(context.Background(), EnableCertificateAuthOptions{}))
	assert.Equal(t, []string{
		"POST /tenant/trusted-certificates/verify-cert-chain",
		"PUT /user/currentUser",
		"POST /tenant/oauth/certificate",
	}, cs.paths())
}

func Test_ContextCertificateSource_ServiceUser(t *testing.T) {
	cert := rsaCert(t)
	cs := newCertServer(t, cert.key.Public().(*rsa.PublicKey))
	cs.jwt = fakeJWT("t123", time.Now().Add(time.Hour))
	tlsCert, err := tls.X509KeyPair(cert.certPEM, cert.keyPEM)
	require.NoError(t, err)

	client := NewClient(ClientOptions{
		BaseURL:              cs.server.URL,
		SkipTenantLookup:     true,
		ContextTokenExchange: true,
		ContextCertificateSource: func(tenant string) (tls.Certificate, error) {
			if tenant == "t123" {
				return tlsCert, nil
			}
			return tls.Certificate{}, nil
		},
		Auth: authentication.AuthOptions{Tenant: "t100", Username: "bootstrap", Password: "x",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic}},
	})

	ctx := WithServiceUser(context.Background(), model.ServiceUser{Tenant: "t123", Username: "service_app", Password: "pw"})
	require.NoError(t, get(client, ctx).Error)

	reqs := cs.find("/tenant/oauth/certificate")
	require.Len(t, reqs, 1)
	assert.Equal(t, "tenant_id=t123", reqs[0].Query)
	assert.Empty(t, cs.find("/tenant/oauth/token"), "the password is not used")
	cs.mu.Lock()
	last := cs.auth[len(cs.auth)-1]
	cs.mu.Unlock()
	assert.Equal(t, "Bearer "+cs.jwt, last)
}
