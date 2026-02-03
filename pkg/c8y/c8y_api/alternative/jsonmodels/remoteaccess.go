package jsonmodels

import "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"

type RemoteAccessConfiguration struct {
	jsondoc.JSONDoc
}

func NewRemoteAccessConfiguration(b []byte) RemoteAccessConfiguration {
	return RemoteAccessConfiguration{jsondoc.New(b)}
}

func (e RemoteAccessConfiguration) ID() string {
	return e.Get("id").String()
}

func (e RemoteAccessConfiguration) Name() string {
	return e.Get("name").String()
}

func (e RemoteAccessConfiguration) Hostname() string {
	return e.Get("hostname").String()
}

func (e RemoteAccessConfiguration) Port() int {
	return int(e.Get("port").Int())
}

func (e RemoteAccessConfiguration) Protocol() string {
	return e.Get("protocol").String()
}

func (e RemoteAccessConfiguration) Credentials() RemoteAccessCredentials {
	if node := e.Get("credentials"); node.Exists() {
		return RemoteAccessCredentials{jsondoc.New([]byte(node.Raw))}
	}
	return RemoteAccessCredentials{JSONDoc: jsondoc.JSONDoc{}}
}

type RemoteAccessCredentials struct {
	jsondoc.JSONDoc
}

func NewRemoteAccessCredentials(b []byte) RemoteAccessCredentials {
	return RemoteAccessCredentials{jsondoc.New(b)}
}

func (e RemoteAccessCredentials) Type() string {
	return e.Get("type").String()
}

func (e RemoteAccessCredentials) Username() string {
	return e.Get("username").String()
}

func (e RemoteAccessCredentials) Password() string {
	return e.Get("password").String()
}

func (e RemoteAccessCredentials) PublicKey() string {
	return e.Get("publicKey").String()
}

func (e RemoteAccessCredentials) PrivateKey() string {
	return e.Get("privateKey").String()
}

func (e RemoteAccessCredentials) HostKey() string {
	return e.Get("hostKey").String()
}
