package model

import (
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/csv"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

const (
	TrustedCertificateStatusEnabled  = "ENABLED"
	TrustedCertificateStatusDisabled = "DISABLED"
)

type ProofOfPossession struct {
	ProofOfPossessionSignedVerificationCode string `json:"proofOfPossessionSignedVerificationCode,omitempty"`
}

func NewTrustedCertificateRevocationCollectionFromCertificates(certs ...TrustedCertificate) *TrustedCertificateRevocationCollection {
	item := &TrustedCertificateRevocationCollection{
		CRLS: make([]TrustedCertificateRevocation, 0, len(certs)),
	}
	for _, cert := range certs {
		item.CRLS = append(item.CRLS, TrustedCertificateRevocation{
			SerialNumberInHex: cert.SerialNumberInHex(),
		})
	}
	return item
}

// Write the revocation list to CSV
func (c *TrustedCertificateRevocationCollection) WriteCSV(w io.Writer) error {
	crlList := csv.NewWriter(w)
	if err := crlList.Write([]string{"SERIALNO", "DATE"}); err != nil {
		return err
	}
	for _, item := range c.CRLS {
		revocationDate := ""
		if !item.RevocationDate.IsZero() {
			if v, err := item.RevocationDate.MarshalText(); err == nil {
				revocationDate = string(v)
			}
		}
		crlList.Write([]string{
			item.SerialNumberInHex,
			revocationDate,
		})
	}
	crlList.Flush()
	return crlList.Error()
}

type TrustedCertificateRevocation struct {
	SerialNumberInHex string    `json:"serialNumberInHex,omitempty"`
	RevocationDate    time.Time `json:"revocationDate,omitzero"`
}

type TrustedCertificateRevocationCollection struct {
	CRLS []TrustedCertificateRevocation `json:"crls,omitempty"`
}

// Certificate properties
type TrustedCertificate struct {
	AlgorithmName              string    `json:"algorithmName,omitempty"`
	CertInPemFormat            string    `json:"certInPemFormat,omitempty"`
	Fingerprint                string    `json:"fingerprint,omitempty"`
	Issuer                     string    `json:"issuer,omitempty"`
	Name                       string    `json:"name,omitempty"`
	NotAfter                   time.Time `json:"notAfter,omitzero"`
	NotBefore                  time.Time `json:"notBefore,omitzero"`
	Self                       string    `json:"self,omitempty"`
	SerialNumber               string    `json:"serialNumber,omitempty"`
	Status                     string    `json:"status,omitempty"`
	Subject                    string    `json:"subject,omitempty"`
	AutoRegistrationEnabled    *bool     `json:"autoRegistrationEnabled,omitempty"`
	TenantCertificateAuthority bool      `json:"tenantCertificateAuthority,omitempty"`
	Version                    int       `json:"version,omitempty"`

	// Proof of Possession
	ProofOfPossessionUnsignedVerificationCode    string    `json:"proofOfPossessionUnsignedVerificationCode,omitempty"`
	ProofOfPossessionValid                       bool      `json:"proofOfPossessionValid,omitempty"`
	ProofOfPossessionVerificationCodeUsableUntil time.Time `json:"proofOfPossessionVerificationCodeUsableUntil,omitzero"`
}

// Check if auto registration is enabled or not
func (c *TrustedCertificate) IsAutoRegistrationEnabled() bool {
	if c.AutoRegistrationEnabled == nil {
		return false
	}
	return *c.AutoRegistrationEnabled
}

// Set the certificate status, ENABLED or DISABLED
func (c *TrustedCertificate) WithStatus(v string) *TrustedCertificate {
	c.Status = v
	return c
}

// Set the auto registration status
func (c *TrustedCertificate) WithAutoRegistration(enabled bool) *TrustedCertificate {
	c.AutoRegistrationEnabled = &enabled
	return c
}

func (c *TrustedCertificate) Certificate() *x509.Certificate {
	derBytes, err := base64.StdEncoding.DecodeString(c.CertInPemFormat)
	if err != nil {
		return &x509.Certificate{}
	}
	if cert, err := x509.ParseCertificate(derBytes); err == nil {
		return cert
	}
	return &x509.Certificate{}
}

func (c *TrustedCertificate) SerialNumberInHex() string {
	return fmt.Sprintf("%x", c.Certificate().SerialNumber)
}

func (c *TrustedCertificate) WriteCertificate(w io.Writer) (err error) {
	derBytes, err := base64.StdEncoding.DecodeString(c.CertInPemFormat)
	if err != nil {
		return err
	}
	pemBlock := &pem.Block{
		Type:    certutil.CertificateBlockType,
		Headers: nil,
		Bytes:   derBytes,
	}
	return pem.Encode(w, pemBlock)
}

func (c *TrustedCertificate) CalculateFingerprint() string {
	hash := sha1.Sum(c.Certificate().Raw)
	fingerprint := fmt.Sprintf("%x", hash)
	c.Fingerprint = fingerprint
	return fingerprint
}

func NewTrustedCertificateFromFile(p string) (*TrustedCertificate, error) {
	if _, err := os.Stat(p); errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	r, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	return NewTrustedCertificate(r)
}

func NewTrustedCertificate(in []byte) (*TrustedCertificate, error) {
	var contents []byte
	if derBytes, err := certutil.Base64Decode(in); err == nil {
		// Format: encoded der bytes without the PEM headers (e.g. the format stored in Cumulocity)
		contents = derBytes
	} else {
		// Format: PEM format with headers
		block, _ := pem.Decode(contents)
		if block == nil || block.Type != certutil.CertificateBlockType {
			// Assume that contents should be unchanged
			contents = in
			// return nil, fmt.Errorf("invalid PEM block. expected 'CERTIFICATE'")
		} else {
			contents = block.Bytes
		}
	}

	x509Cert, err := x509.ParseCertificate(contents)
	if err != nil {
		return nil, err
	}

	autoRegistration := false
	cert := &TrustedCertificate{
		Name:                    x509Cert.Subject.CommonName,
		CertInPemFormat:         base64.StdEncoding.EncodeToString(contents),
		AutoRegistrationEnabled: &autoRegistration,
		Status:                  TrustedCertificateStatusEnabled,
	}
	cert.CalculateFingerprint()
	return cert, nil

}

// TrustedCertificateCollection a list of the trusted certificates
type TrustedCertificateCollection struct {
	*BaseResponse

	Certificates []TrustedCertificate `json:"certificates"`
}
