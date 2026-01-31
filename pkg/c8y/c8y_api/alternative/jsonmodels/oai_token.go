package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"

type OAIToken struct {
	jsondoc.JSONDoc
}

func NewOAIToken(b []byte) OAIToken {
	return OAIToken{jsondoc.New(b)}
}

// AccessToken returns the OAI-Secure access token
func (t OAIToken) AccessToken() string {
	return t.Get("access_token").String()
}
