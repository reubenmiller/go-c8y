package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type User struct {
	jsondoc.Facade
}

func NewUser(b []byte) User {
	return User{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (u User) ID() string {
	return u.Get("id").String()
}

func (u User) UserName() string {
	return u.Get("userName").String()
}

func (u User) Email() string {
	return u.Get("email").String()
}

func (u User) FirstName() string {
	return u.Get("firstName").String()
}

func (u User) LastName() string {
	return u.Get("lastName").String()
}

// Enabled returns whether the user is enabled. Cumulocity omits this field
// when the user is requested with withCertificates=true.
func (u User) Enabled() bool {
	return u.Get("enabled").Bool()
}

func (u User) Self() string {
	return u.Get("self").String()
}

func (u User) LastPasswordChange() time.Time {
	return u.Get("lastPasswordChange").Time()
}

// ProvisionedCertificate is a trusted certificate provisioned for a device user.
type ProvisionedCertificate struct {
	SerialNumber string    `json:"serialNumber"`
	NotBefore    time.Time `json:"notBefore"`
	NotAfter     time.Time `json:"notAfter"`
}

// ProvisionedCertificates returns the certificates provisioned for the user.
// It is only populated when the user was requested with withCertificates=true
// (see users.GetOptions.WithCertificates).
func (u User) ProvisionedCertificates() []ProvisionedCertificate {
	var out []ProvisionedCertificate
	for _, c := range u.Get("provisionedCertificates").Array() {
		out = append(out, ProvisionedCertificate{
			SerialNumber: c.Get("serialNumber").String(),
			NotBefore:    c.Get("notBefore").Time(),
			NotAfter:     c.Get("notAfter").Time(),
		})
	}
	return out
}
