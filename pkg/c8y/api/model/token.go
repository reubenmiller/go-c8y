package model

import "github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"

type OAIToken struct {
	AccessToken string `json:"access_token,omitempty"`
}

func (o *OAIToken) GetXSRFToken() (value string) {
	if claim, err := authentication.ParseToken(o.AccessToken); err == nil {
		value = claim.XSRFToken
	}
	return
}

func (o *OAIToken) GetUsername() (value string) {
	if claim, err := authentication.ParseToken(o.AccessToken); err == nil {
		value = claim.User
	}
	return
}

type DeviceAccessToken struct {
	AccessToken  string `json:"accessToken,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

func (o *DeviceAccessToken) GetXSRFToken() (value string) {
	if claim, err := authentication.ParseToken(o.AccessToken); err == nil {
		value = claim.XSRFToken
	}
	return
}

func (o *DeviceAccessToken) GetUsername() (value string) {
	if claim, err := authentication.ParseToken(o.AccessToken); err == nil {
		value = claim.User
	}
	return
}
