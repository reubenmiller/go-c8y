package fakeserver

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/smallstep/pkcs7"
)

// estCA is the fake tenant certificate authority which signs the certificate
// signing requests of enrolling devices. It is created on first use.
type estCA struct {
	once sync.Once
	key  *ecdsa.PrivateKey
	cert *x509.Certificate
	err  error
}

func (ca *estCA) init() error {
	ca.once.Do(func() {
		ca.key, ca.err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if ca.err != nil {
			return
		}
		tmpl := &x509.Certificate{
			SerialNumber:          big.NewInt(1),
			Subject:               pkix.Name{CommonName: "Fake Tenant CA", Organization: []string{"t12345"}}, //nolint:misspell
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().AddDate(10, 0, 0),
			IsCA:                  true,
			BasicConstraintsValid: true,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		}
		der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &ca.key.PublicKey, ca.key)
		if err != nil {
			ca.err = err
			return
		}
		ca.cert, ca.err = x509.ParseCertificate(der)
	})
	return ca.err
}

// handleEST handles /.well-known/est/simpleenroll. A device authenticates with
// its external ID and the one-time password it was registered with (in the bulk
// registration's ENROLLMENT_OTP column), and gets its certificate signing
// request signed, as a base64 encoded PKCS#7 response. The one-time password
// can be used once. Until the device is registered the request is rejected, so
// a device polls until a user registers it.
func (fs *FakeServer) handleEST(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/.well-known/est/simpleenroll" {
		writeError(w, http.StatusNotFound, "general/notFound", "Endpoint not found: "+r.URL.Path)
		return
	}
	externalID, otp, ok := r.BasicAuth()
	if !ok {
		writeError(w, http.StatusUnauthorized, "security/Unauthorized", "Missing credentials") //nolint:misspell
		return
	}
	enrollment, found := fs.Enrollments.Get(externalID)
	if !found || getJSONString(enrollment, "otp") != otp {
		writeError(w, http.StatusUnauthorized, "security/Unauthorized", "Invalid credentials") //nolint:misspell
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
		return
	}
	der, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", "the certificate signing request is not base64 encoded")
		return
	}
	csr, err := x509.ParseCertificateRequest(der)
	if err == nil {
		err = csr.CheckSignature()
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "general/badRequest", "invalid certificate signing request: "+err.Error())
		return
	}
	if csr.Subject.CommonName != externalID {
		writeError(w, http.StatusBadRequest, "general/badRequest", "the common name of the certificate signing request must be the external ID")
		return
	}

	if err := fs.ca.init(); err != nil {
		writeError(w, http.StatusInternalServerError, "general/internalError", err.Error())
		return
	}
	certDER, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      csr.Subject,
		NotBefore:    time.Now().Add(-time.Minute),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}, fs.ca.cert, csr.PublicKey, fs.ca.key)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "general/internalError", err.Error())
		return
	}
	p7, err := pkcs7.DegenerateCertificate(certDER)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "general/internalError", err.Error())
		return
	}
	fs.Enrollments.Delete(externalID)

	w.Header().Set("Content-Type", "application/pkcs7-mime; smime-type=certs-only")
	w.Header().Set("Content-Transfer-Encoding", "base64")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(base64.StdEncoding.EncodeToString(p7)))
}
