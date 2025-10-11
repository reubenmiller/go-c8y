package authentication

import "fmt"

type AuthOptions struct {
	// Username / Password Auth
	Tenant   string
	Username string
	Password string

	// Token Auth
	Token string

	// Client Certificate Auth
	// Certificate key file path or contents
	CertificateKey string

	// Certificate public file path or contents
	Certificate string

	// Auth preference (to control which credentials are used when more than 1 value is provided)
	AuthType []AuthType
}

func (a *AuthOptions) GetAuthTypes() []AuthType {
	if len(a.AuthType) > 0 {
		return a.AuthType
	}
	return []AuthType{AuthTypeUnset}
}

// AuthType request authorization type
type AuthType int

const (
	// AuthTypeUnset no auth type set
	AuthTypeUnset AuthType = 0

	// AuthTypeNone don't use an Authorization
	AuthTypeNone AuthType = 1

	// AuthTypeBasic Basic Authorization
	AuthTypeBasic AuthType = 2

	// AuthTypeBearer Bearer Authorization
	AuthTypeBearer AuthType = 3
)

func (a AuthType) String() string {
	switch a {
	case AuthTypeUnset:
		return "UNSET"
	case AuthTypeNone:
		return "NONE"
	case AuthTypeBasic:
		return "BASIC"
	case AuthTypeBearer:
		return "BEARER"
	}
	return "UNKNOWN"
}

func JoinTenantUser(tenant string, user string) string {
	if tenant != "" {
		return fmt.Sprintf("%s/%s", tenant, user)
	}
	return user
}
