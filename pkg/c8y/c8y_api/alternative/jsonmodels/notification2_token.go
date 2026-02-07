package jsonmodels

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
)

// Notification2Token notification2 token which can be used by client to subscribe to notifications
type Notification2Token struct {
	jsondoc.Facade
}

func NewNotification2Token(data []byte) Notification2Token {
	return Notification2Token{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (n Notification2Token) Token() string {
	return n.Get("token").String()
}
