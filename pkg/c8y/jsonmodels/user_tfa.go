package jsonmodels

import (
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// UserTFA represents the TFA settings of a user (userTfaData schema).
type UserTFA struct {
	jsondoc.JSONDoc
}

func NewUserTFA(b []byte) UserTFA {
	return UserTFA{jsondoc.New(b)}
}

// TFAEnabled indicates whether the user has enabled two-factor authentication.
func (u UserTFA) TFAEnabled() bool {
	return u.Get("tfaEnabled").Bool()
}

// TFAEnforced indicates whether two-factor authentication is enforced by the tenant admin.
func (u UserTFA) TFAEnforced() bool {
	return u.Get("tfaEnforced").Bool()
}

// Strategy returns the two-factor authentication strategy (SMS or TOTP).
func (u UserTFA) Strategy() string {
	return u.Get("strategy").String()
}

// LastTFARequestTime returns the latest date and time when the user last used TFA to log in.
func (u UserTFA) LastTFARequestTime() time.Time {
	return u.Get("lastTfaRequestTime").Time()
}
