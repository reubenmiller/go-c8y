package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

type DeviceAccessToken struct {
	jsondoc.JSONDoc
}

func NewDeviceAccessToken(b []byte) DeviceAccessToken {
	return DeviceAccessToken{jsondoc.New(b)}
}

// AccessToken returns the device access token
func (t DeviceAccessToken) AccessToken() string {
	return t.Get("accessToken").String()
}

// RefreshToken returns the refresh token
func (t DeviceAccessToken) RefreshToken() string {
	return t.Get("refreshToken").String()
}

func (t DeviceAccessToken) GetXSRFToken() (value string) {
	if claim, err := authentication.ParseToken(t.AccessToken()); err == nil {
		value = claim.XSRFToken
	}
	return
}

func (t DeviceAccessToken) GetUsername() (value string) {
	if claim, err := authentication.ParseToken(t.AccessToken()); err == nil {
		value = claim.User
	}
	return
}
