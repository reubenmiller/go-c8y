package model

import "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"

type OAIToken struct {
	AccessToken string `json:"access_token,omitempty"`
}

func (o *OAIToken) GetXSRFToken() (value string) {
	if claim, err := authentication.ParseToken(o.AccessToken); err == nil {
		value = claim.XSRFToken
	}
	return
}
