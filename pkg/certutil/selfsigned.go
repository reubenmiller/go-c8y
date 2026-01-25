package certutil

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"
)

// CertTemplate is a helper function to create a cert template with a serial number and other required fields
func CertTemplate() (*x509.Certificate, error) {
	// generate a random serial number (a real cert authority would have some logic behind this)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, errors.New("failed to generate serial number: " + err.Error())
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"go-c8y-cli"}},
		SignatureAlgorithm:    x509.SHA256WithRSA,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 30), // valid for a month
		BasicConstraintsValid: true,
	}
	return &tmpl, nil
}

// CreateCert invokes x509.CreateCertificate and returns it in the x509.Certificate format
func CreateCert(template, parent *x509.Certificate, pub interface{}, parentPriv interface{}) (
	cert *x509.Certificate, certPEM []byte, err error) {

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	// parse the resulting certificate so we can use it again
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}
	// PEM encode the certificate (this is a standard TLS encoding)
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	return
}

func NewSelfSignedCertificate(key any, commonName string) (*x509.Certificate, error) {
	rootCertTmpl, err := CertTemplate()
	if err != nil {
		return nil, err
	}

	// this cert will be the CA that we will use to sign the server cert
	rootCertTmpl.IsCA = true
	// describe what the certificate will be used for
	rootCertTmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	rootCertTmpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}
	rootCertTmpl.Subject.CommonName = commonName

	switch typedKey := key.(type) {
	case *rsa.PrivateKey:
		rootCertTmpl.SignatureAlgorithm = x509.SHA256WithRSA
		rootCert, _, err := CreateCert(rootCertTmpl, rootCertTmpl, &typedKey.PublicKey, key)
		return rootCert, err
	case *ecdsa.PrivateKey:
		rootCertTmpl.SignatureAlgorithm = x509.ECDSAWithSHA256
		rootCert, _, err := CreateCert(rootCertTmpl, rootCertTmpl, &typedKey.PublicKey, key)
		return rootCert, err
	default:
		return nil, fmt.Errorf("invalid key type. Only accepts *rsa.PrivateKey or *ecdsa.PrivateKey")
	}
}
