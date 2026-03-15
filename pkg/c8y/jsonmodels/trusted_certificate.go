package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type TrustedCertificate struct {
	jsondoc.Facade
}

func NewTrustedCertificate(b []byte) TrustedCertificate {
	return TrustedCertificate{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (t TrustedCertificate) Fingerprint() string {
	return t.Get("fingerprint").String()
}

func (t TrustedCertificate) Name() string {
	return t.Get("name").String()
}

func (t TrustedCertificate) Status() string {
	return t.Get("status").String()
}

func (t TrustedCertificate) NotAfter() time.Time {
	return t.Get("notAfter").Time()
}

func (t TrustedCertificate) NotBefore() time.Time {
	return t.Get("notBefore").Time()
}

func (t TrustedCertificate) Self() string {
	return t.Get("self").String()
}

func (t TrustedCertificate) ProofOfPossessionUnsignedVerificationCode() string {
	return t.Get("proofOfPossessionUnsignedVerificationCode").String()
}

func (t TrustedCertificate) ProofOfPossessionVerificationCodeUsableUntil() time.Time {
	return t.Get("proofOfPossessionVerificationCodeUsableUntil").Time()
}

func (t TrustedCertificate) ProofOfPossessionValid() bool {
	return t.Get("proofOfPossessionValid").Bool()
}

func (t TrustedCertificate) TenantCertificateAuthority() bool {
	return t.Get("tenantCertificateAuthority").Bool()
}
