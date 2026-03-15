package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

type LoginOption struct {
	jsondoc.JSONDoc
}

func NewLoginOption(b []byte) LoginOption {
	return LoginOption{jsondoc.New(b)}
}

// ID returns the unique identifier of this login option
func (o LoginOption) ID() string {
	return o.Get("id").String()
}

// Type returns the authentication configuration type (BASIC, OAUTH2, OAUTH2_INTERNAL)
func (o LoginOption) Type() string {
	return o.Get("type").String()
}

// ProviderName returns the name of the authentication provider
func (o LoginOption) ProviderName() string {
	return o.Get("providerName").String()
}

// Self returns the URL to this resource
func (o LoginOption) Self() string {
	return o.Get("self").String()
}

// VisibleOnLoginPage returns whether the authentication form should be visible on the login page
func (o LoginOption) VisibleOnLoginPage() bool {
	return o.Get("visibleOnLoginPage").Bool()
}

// UserManagementSource returns the user management source
func (o LoginOption) UserManagementSource() string {
	return o.Get("userManagementSource").String()
}

// TFAStrategy returns the two-factor authentication strategy
func (o LoginOption) TFAStrategy() string {
	return o.Get("tfaStrategy").String()
}

// InitRequest returns the init request URL
func (o LoginOption) InitRequest() string {
	return o.Get("initRequest").String()
}

// GrantType returns the OAuth2 grant type
func (o LoginOption) GrantType() string {
	return o.Get("grantType").String()
}
