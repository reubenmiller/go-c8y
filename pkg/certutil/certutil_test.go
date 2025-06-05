package certutil_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"math/big"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

func TestMakeEllipticPrivateKeyPEM(t *testing.T) {
	pemData, err := certutil.MakeEllipticPrivateKeyPEM()
	if err != nil {
		t.Fatalf("MakeEllipticPrivateKeyPEM() error = %v", err)
	}
	if len(pemData) == 0 {
		t.Errorf("MakeEllipticPrivateKeyPEM() returned empty data")
	}
	if !strings.Contains(string(pemData), certutil.PrivateKeyBlockType) {
		t.Errorf("MakeEllipticPrivateKeyPEM() PEM data does not contain expected block type. Got: %s", string(pemData))
	}
}

func TestMakeEllipticPrivateKeyWithCurvePEM(t *testing.T) {
	pemData, err := certutil.MakeEllipticPrivateKeyWithCurvePEM(elliptic.P224())
	if err != nil {
		t.Fatalf("MakeEllipticPrivateKeyWithCurvePEM() error = %v", err)
	}
	if len(pemData) == 0 {
		t.Errorf("MakeEllipticPrivateKeyWithCurvePEM() returned empty data")
	}
	// Further validation could involve parsing the key and checking its curve
}

func TestMakeRSAPrivateKeyPEM(t *testing.T) {
	pemData, err := certutil.MakeRSAPrivateKeyPEM(2048)
	if err != nil {
		t.Fatalf("MakeRSAPrivateKeyPEM() error = %v", err)
	}
	if len(pemData) == 0 {
		t.Errorf("MakeRSAPrivateKeyPEM() returned empty data")
	}
	if !strings.Contains(string(pemData), certutil.PrivateKeyBlockType) {
		t.Errorf("MakeRSAPrivateKeyPEM() PEM data does not contain expected block type. Got: %s", string(pemData))
	}
}

func TestMakeRSAPrivateKeyPEM_InvalidKeySize(t *testing.T) {
	_, err := certutil.MakeRSAPrivateKeyPEM(12)
	if err == nil {
		t.Fatalf("Expected an error returned by MakeRSAPrivateKeyPEM")
	}
}

func TestWriteKey(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_key.pem")
	keyData := []byte("test key data")

	err := certutil.WriteKey(keyPath, keyData)
	if err != nil {
		t.Fatalf("WriteKey() error = %v", err)
	}

	readData, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("Failed to read key file: %v", err)
	}
	if string(readData) != string(keyData) {
		t.Errorf("WriteKey() wrote %q, want %q", string(readData), string(keyData))
	}
}

func TestLoadOrGenerateKeyFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_key.pem")

	// Test generation
	data, wasGenerated, err := certutil.LoadOrGenerateKeyFile(keyPath)
	if err != nil {
		t.Fatalf("LoadOrGenerateKeyFile() (generate) error = %v", err)
	}
	if !wasGenerated {
		t.Errorf("LoadOrGenerateKeyFile() (generate) wasGenerated = false, want true")
	}
	if len(data) == 0 {
		t.Errorf("LoadOrGenerateKeyFile() (generate) returned empty data")
	}

	// Test loading
	loadedData, wasGenerated, err := certutil.LoadOrGenerateKeyFile(keyPath)
	if err != nil {
		t.Fatalf("LoadOrGenerateKeyFile() (load) error = %v", err)
	}
	if wasGenerated {
		t.Errorf("LoadOrGenerateKeyFile() (load) wasGenerated = true, want false")
	}
	if string(loadedData) != string(data) {
		t.Errorf("LoadOrGenerateKeyFile() (load) loaded %q, want %q", string(loadedData), string(data))
	}

	// Test with empty/corrupt file
	corruptKeyPath := filepath.Join(tempDir, "corrupt_key.pem")
	if err := os.WriteFile(corruptKeyPath, []byte("corrupt"), 0600); err != nil {
		t.Fatalf("Failed to write corrupt key file: %v", err)
	}
	_, wasGenerated, err = certutil.LoadOrGenerateKeyFile(corruptKeyPath)
	if err == nil && wasGenerated { // if it generates, it's fine
		// This is acceptable, it means it treated the corrupt file as non-existent
	} else if err != nil && !strings.Contains(err.Error(), "error loading key from") && !strings.Contains(err.Error(), "data does not contain a valid") {
		// If there's an error, it should be about loading or parsing
		t.Errorf("LoadOrGenerateKeyFile() with corrupt file returned unexpected error: %v", err)
	} else if err == nil && !wasGenerated {
		t.Errorf("LoadOrGenerateKeyFile() with corrupt file should have either errored or generated a new key")
	}
}

func TestMarshalPrivateKeyToPEM(t *testing.T) {
	// ECDSA
	ecdsaKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	pemData, err := certutil.MarshalPrivateKeyToPEM(ecdsaKey)
	if err != nil {
		t.Fatalf("MarshalPrivateKeyToPEM(ecdsa) error = %v", err)
	}
	if !strings.Contains(string(pemData), certutil.PrivateKeyBlockType) {
		t.Errorf("MarshalPrivateKeyToPEM(ecdsa) PEM data does not contain expected block type. Got: %s", string(pemData))
	}

	// RSA
	rsaKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	pemData, err = certutil.MarshalPrivateKeyToPEM(rsaKey)
	if err != nil {
		t.Fatalf("MarshalPrivateKeyToPEM(rsa) error = %v", err)
	}
	if !strings.Contains(string(pemData), certutil.PrivateKeyBlockType) {
		t.Errorf("MarshalPrivateKeyToPEM(rsa) PEM data does not contain expected block type. Got: %s", string(pemData))
	}

	// Unsupported type
	_, err = certutil.MarshalPrivateKeyToPEM("unsupported")
	if err == nil {
		t.Errorf("MarshalPrivateKeyToPEM(unsupported) expected an error, got nil")
	}
}

func TestParsePrivateKeyPEM(t *testing.T) {
	// ECDSA
	ecdsaPEM, _ := certutil.MakeEllipticPrivateKeyPEM()
	key, err := certutil.ParsePrivateKeyPEM(ecdsaPEM)
	if err != nil {
		t.Fatalf("ParsePrivateKeyPEM(ecdsa) error = %v", err)
	}
	if _, ok := key.(*ecdsa.PrivateKey); !ok {
		t.Errorf("ParsePrivateKeyPEM(ecdsa) returned type %T, want *ecdsa.PrivateKey", key)
	}

	// RSA
	rsaPEM, _ := certutil.MakeRSAPrivateKeyPEM(2048)
	key, err = certutil.ParsePrivateKeyPEM(rsaPEM)
	if err != nil {
		t.Fatalf("ParsePrivateKeyPEM(rsa) error = %v", err)
	}
	if _, ok := key.(*rsa.PrivateKey); !ok {
		t.Errorf("ParsePrivateKeyPEM(rsa) returned type %T, want *rsa.PrivateKey", key)
	}

	// Invalid data
	_, err = certutil.ParsePrivateKeyPEM([]byte("invalid data"))
	if err == nil {
		t.Errorf("ParsePrivateKeyPEM(invalid) expected an error, got nil")
	}
}

func TestPrivateKeyFromFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "key.pem")

	// ECDSA
	ecdsaPEM, _ := certutil.MakeEllipticPrivateKeyPEM()
	_ = certutil.WriteKey(keyPath, ecdsaPEM)
	key, err := certutil.PrivateKeyFromFile(keyPath)
	if err != nil {
		t.Fatalf("PrivateKeyFromFile(ecdsa) error = %v", err)
	}
	if _, ok := key.(*ecdsa.PrivateKey); !ok {
		t.Errorf("PrivateKeyFromFile(ecdsa) returned type %T, want *ecdsa.PrivateKey", key)
	}
	_ = os.Remove(keyPath)

	// RSA
	rsaPEM, _ := certutil.MakeRSAPrivateKeyPEM(2048)
	_ = certutil.WriteKey(keyPath, rsaPEM)
	key, err = certutil.PrivateKeyFromFile(keyPath)
	if err != nil {
		t.Fatalf("PrivateKeyFromFile(rsa) error = %v", err)
	}
	if _, ok := key.(*rsa.PrivateKey); !ok {
		t.Errorf("PrivateKeyFromFile(rsa) returned type %T, want *rsa.PrivateKey", key)
	}
}

func TestParsePublicKeysPEM(t *testing.T) {
	// From ECDSA Private Key PEM
	ecdsaPrivPEM, _ := certutil.MakeEllipticPrivateKeyPEM()
	keys, err := certutil.ParsePublicKeysPEM(ecdsaPrivPEM)
	if err != nil {
		t.Fatalf("ParsePublicKeysPEM(from ecdsa private) error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("ParsePublicKeysPEM(from ecdsa private) expected 1 key, got %d", len(keys))
	}
	if _, ok := keys[0].(*ecdsa.PublicKey); !ok {
		t.Errorf("ParsePublicKeysPEM(from ecdsa private) returned type %T, want *ecdsa.PublicKey", keys[0])
	}

	// From RSA Private Key PEM
	rsaPrivPEM, _ := certutil.MakeRSAPrivateKeyPEM(2048)
	keys, err = certutil.ParsePublicKeysPEM(rsaPrivPEM)
	if err != nil {
		t.Fatalf("ParsePublicKeysPEM(from rsa private) error = %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("ParsePublicKeysPEM(from rsa private) expected 1 key, got %d", len(keys))
	}
	if _, ok := keys[0].(*rsa.PublicKey); !ok {
		t.Errorf("ParsePublicKeysPEM(from rsa private) returned type %T, want *rsa.PublicKey", keys[0])
	}

	// From ECDSA Public Key PEM
	ecdsaPriv, _ := certutil.ParsePrivateKeyPEM(ecdsaPrivPEM)
	ecdsaPubBytes, _ := x509.MarshalPKIXPublicKey(&ecdsaPriv.(*ecdsa.PrivateKey).PublicKey)
	ecdsaPubPEM := certutil.MarshalCertificateToPEM(ecdsaPubBytes) // Re-using Marshal func for PEM structure
	keys, err = certutil.ParsePublicKeysPEM(ecdsaPubPEM)
	testingutils.Ok(t, err)
	testingutils.Equals(t, 1, len(keys))
	// Note: This test might fail if MarshalCertificateToPEM uses "CERTIFICATE" block type.
	// A dedicated MarshalPublicKeyToPEM would be better.
	// For now, we'll assume it might fail due to block type mismatch if not handled.
	// If it passes, it means the parsing is robust enough for this case.

	// Invalid data
	_, err = certutil.ParsePublicKeysPEM([]byte("invalid data"))
	if err == nil {
		t.Errorf("ParsePublicKeysPEM(invalid) expected an error, got nil")
	}
}

func TestPublicKeysFromFile(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "pubkey.pem")

	// ECDSA
	ecdsaPEM, _ := certutil.MakeEllipticPrivateKeyPEM() // Using private key PEM as it contains public part
	_ = certutil.WriteKey(keyPath, ecdsaPEM)
	keys, err := certutil.PublicKeysFromFile(keyPath)
	if err != nil {
		t.Fatalf("PublicKeysFromFile(ecdsa) error = %v", err)
	}
	if len(keys) != 1 || keys[0] == nil {
		t.Fatalf("PublicKeysFromFile(ecdsa) expected 1 key, got %d or nil key", len(keys))
	}
	if _, ok := keys[0].(*ecdsa.PublicKey); !ok {
		t.Errorf("PublicKeysFromFile(ecdsa) returned type %T, want *ecdsa.PublicKey", keys[0])
	}
	_ = os.Remove(keyPath)

	// RSA
	rsaPEM, _ := certutil.MakeRSAPrivateKeyPEM(2048) // Using private key PEM
	_ = certutil.WriteKey(keyPath, rsaPEM)
	keys, err = certutil.PublicKeysFromFile(keyPath)
	if err != nil {
		t.Fatalf("PublicKeysFromFile(rsa) error = %v", err)
	}
	if len(keys) != 1 || keys[0] == nil {
		t.Fatalf("PublicKeysFromFile(rsa) expected 1 key, got %d or nil key", len(keys))
	}
	if _, ok := keys[0].(*rsa.PublicKey); !ok {
		t.Errorf("PublicKeysFromFile(rsa) returned type %T, want *rsa.PublicKey", keys[0])
	}
}

func generateTestCertificatePEM(t *testing.T, key interface{}) []byte {
	t.Helper()
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.example.com",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour * 24 * 30),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}

	var pubKey any
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		pubKey = &k.PublicKey
	case *rsa.PrivateKey:
		pubKey = &k.PublicKey
	default:
		t.Fatalf("Unsupported key type for cert generation: %T", key)
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, pubKey, key)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}
	return certutil.MarshalCertificateToPEM(derBytes)
}

func TestParseCertificatePEM(t *testing.T) {
	privKey, _ := certutil.MakeEllipticPrivateKeyPEM()
	key, _ := certutil.ParsePrivateKeyPEM(privKey)

	certPEM := generateTestCertificatePEM(t, key)

	cert, err := certutil.ParseCertificatePEM(certPEM)
	if err != nil {
		t.Fatalf("ParseCertificatePEM() error = %v", err)
	}
	if cert.Subject.CommonName != "test.example.com" {
		t.Errorf("ParseCertificatePEM() common name = %q, want %q", cert.Subject.CommonName, "test.example.com")
	}

	// Invalid data
	_, err = certutil.ParseCertificatePEM([]byte("invalid data"))
	if err == nil {
		t.Errorf("ParseCertificatePEM(invalid) expected an error, got nil")
	}
}

func TestMarshalCertificateSigningRequestToPEM(t *testing.T) {
	privKeyPEM, _ := certutil.MakeEllipticPrivateKeyPEM()
	key, _ := certutil.ParsePrivateKeyPEM(privKeyPEM)
	csr, err := certutil.CreateCertificateSigningRequest(pkix.Name{CommonName: "test.csr"}, key)
	if err != nil {
		t.Fatalf("CreateCertificateSigningRequest failed: %v", err)
	}

	pemData := certutil.MarshalCertificateSigningRequestToPEM(csr.Raw)
	if !strings.Contains(string(pemData), certutil.CertificateRequestBlockType) {
		t.Errorf("MarshalCertificateSigningRequestToPEM() PEM data does not contain expected block type. Got: %s", string(pemData))
	}
}

func TestMarshalCertificateToPEM(t *testing.T) {
	privKeyPEM, _ := certutil.MakeEllipticPrivateKeyPEM()
	key, _ := certutil.ParsePrivateKeyPEM(privKeyPEM)
	certTemplate := x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test.cert"}}
	derBytes, _ := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, &key.(*ecdsa.PrivateKey).PublicKey, key)

	pemData := certutil.MarshalCertificateToPEM(derBytes)
	if !strings.Contains(string(pemData), certutil.CertificateBlockType) {
		t.Errorf("MarshalCertificateToPEM() PEM data does not contain expected block type. Got: %s", string(pemData))
	}
}

func TestCreateCertificateSigningRequest(t *testing.T) {
	privKeyPEM, _ := certutil.MakeEllipticPrivateKeyPEM()
	key, _ := certutil.ParsePrivateKeyPEM(privKeyPEM)

	subject := pkix.Name{CommonName: "test.example.com"}
	csr, err := certutil.CreateCertificateSigningRequest(subject, key)
	if err != nil {
		t.Fatalf("CreateCertificateSigningRequest() error = %v", err)
	}
	if csr.Subject.CommonName != subject.CommonName {
		t.Errorf("CSR subject common name = %q, want %q", csr.Subject.CommonName, subject.CommonName)
	}
}

func TestBase64Decode(t *testing.T) {
	original := "hello world"
	encoded := base64.StdEncoding.EncodeToString([]byte(original))

	decodedBytes, err := certutil.Base64Decode([]byte(encoded))
	if err != nil {
		t.Fatalf("Base64Decode() error = %v", err)
	}
	if string(decodedBytes) != original {
		t.Errorf("Base64Decode() got %q, want %q", string(decodedBytes), original)
	}

	// Invalid base64
	_, err = certutil.Base64Decode([]byte("not valid base64!!!"))
	if err == nil {
		t.Errorf("Base64Decode(invalid) expected an error, got nil")
	}
}
