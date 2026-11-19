package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// VerifyCertificateChain is the result of verifying a certificate chain
// against a tenant's trusted certificates.
type VerifyCertificateChain struct {
	jsondoc.JSONDoc
}

func NewVerifyCertificateChain(b []byte) VerifyCertificateChain {
	return VerifyCertificateChain{jsondoc.New(b)}
}

// SuccessfullyValidated reports whether the chain is valid and not revoked.
func (v VerifyCertificateChain) SuccessfullyValidated() bool {
	return v.Get("successfullyValidated").Bool()
}

// TenantID returns the tenant used for validation.
func (v VerifyCertificateChain) TenantID() string {
	return v.Get("tenantId").String()
}

// Subject returns the subject of the leaf certificate.
func (v VerifyCertificateChain) Subject() string {
	return v.Get("subject").String()
}

// Issuer returns the issuer of the leaf certificate.
func (v VerifyCertificateChain) Issuer() string {
	return v.Get("issuer").String()
}
