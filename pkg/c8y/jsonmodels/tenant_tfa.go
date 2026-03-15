package jsonmodels

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

// TenantTFA represents the TFA settings of a tenant (tenantTfaData schema).
type TenantTFA struct {
	jsondoc.JSONDoc
}

func NewTenantTFA(b []byte) TenantTFA {
	return TenantTFA{jsondoc.New(b)}
}

// EnabledOnSystemLevel indicates whether two-factor authentication is enabled on system level.
func (t TenantTFA) EnabledOnSystemLevel() bool {
	return t.Get("enabledOnSystemLevel").Bool()
}

// EnabledOnTenantLevel indicates whether two-factor authentication is enabled on tenant level.
func (t TenantTFA) EnabledOnTenantLevel() bool {
	return t.Get("enabledOnTenantLevel").Bool()
}

// EnforcedOnSystemLevel indicates whether two-factor authentication is enforced on system level.
func (t TenantTFA) EnforcedOnSystemLevel() bool {
	return t.Get("enforcedOnSystemLevel").Bool()
}

// EnforcedUsersGroup returns the group for which two-factor authentication is enforced.
func (t TenantTFA) EnforcedUsersGroup() string {
	return t.Get("enforcedUsersGroup").String()
}

// Strategy returns the two-factor authentication strategy (SMS or TOTP).
func (t TenantTFA) Strategy() string {
	return t.Get("strategy").String()
}

// TOTPEnforcedOnTenantLevel indicates whether TOTP two-factor authentication is enforced on tenant level.
func (t TenantTFA) TOTPEnforcedOnTenantLevel() bool {
	return t.Get("totpEnforcedOnTenantLevel").Bool()
}
