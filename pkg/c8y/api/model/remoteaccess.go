package model

import "encoding/json"

type RemoteAccessProtocol string

const (
	RemoteAccessProtocolPassthrough RemoteAccessProtocol = "PASSTHROUGH"
	RemoteAccessProtocolSSH         RemoteAccessProtocol = "SSH"
	RemoteAccessProtocolVNC         RemoteAccessProtocol = "VNC"
	RemoteAccessProtocolTelnet      RemoteAccessProtocol = "TELNET"
)

type RemoteAccessCredentialsType string

const (
	RemoteAccessCredentialsTypeNone         RemoteAccessCredentialsType = "NONE"
	RemoteAccessCredentialsTypeUserPassword RemoteAccessCredentialsType = "USER_PASS"
	RemoteAccessCredentialsTypeKeyPair      RemoteAccessCredentialsType = "KEY_PAIR"
	RemoteAccessCredentialsTypeCertificate  RemoteAccessCredentialsType = "CERTIFICATE"
)

type RemoteAccessConfiguration struct {
	ID              string                      `json:"id,omitempty"`
	Name            string                      `json:"name,omitempty"`
	Hostname        string                      `json:"hostname,omitempty"`
	Port            int                         `json:"port,omitempty"`
	Protocol        RemoteAccessProtocol        `json:"protocol,omitempty"`
	CredentialsType RemoteAccessCredentialsType `json:"credentialsType,omitempty"`
	Username        string                      `json:"username,omitempty"`
	Password        string                      `json:"password,omitempty"`
	PublicKey       string                      `json:"publicKey,omitempty"`
	PrivateKey      string                      `json:"privateKey,omitempty"`
	HostKey         string                      `json:"hostKey,omitempty"`
}

// remoteAccessBase contains fields common to all remote access configuration types.
type remoteAccessBase struct {
	ID       string `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Port     int    `json:"port,omitempty"`
}

// RemoteAccessPassthrough is a passthrough remote access configuration.
// Protocol is always PASSTHROUGH and CredentialsType is always NONE.
type RemoteAccessPassthrough struct {
	remoteAccessBase
}

func (r RemoteAccessPassthrough) MarshalJSON() ([]byte, error) {
	type alias RemoteAccessPassthrough
	return json.Marshal(struct {
		alias
		Protocol        RemoteAccessProtocol        `json:"protocol"`
		CredentialsType RemoteAccessCredentialsType `json:"credentialsType"`
	}{
		alias:           alias(r),
		Protocol:        RemoteAccessProtocolPassthrough,
		CredentialsType: RemoteAccessCredentialsTypeNone,
	})
}

// RemoteAccessSSHUserPass is an SSH remote access configuration using username and password credentials.
type RemoteAccessSSHUserPass struct {
	remoteAccessBase
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func (r RemoteAccessSSHUserPass) MarshalJSON() ([]byte, error) {
	type alias RemoteAccessSSHUserPass
	return json.Marshal(struct {
		alias
		Protocol        RemoteAccessProtocol        `json:"protocol"`
		CredentialsType RemoteAccessCredentialsType `json:"credentialsType"`
	}{
		alias:           alias(r),
		Protocol:        RemoteAccessProtocolSSH,
		CredentialsType: RemoteAccessCredentialsTypeUserPassword,
	})
}

// RemoteAccessSSHKeyPair is an SSH remote access configuration using a key pair.
type RemoteAccessSSHKeyPair struct {
	remoteAccessBase
	Username   string `json:"username,omitempty"`
	PublicKey  string `json:"publicKey,omitempty"`
	PrivateKey string `json:"privateKey,omitempty"`
	HostKey    string `json:"hostKey,omitempty"`
}

func (r RemoteAccessSSHKeyPair) MarshalJSON() ([]byte, error) {
	type alias RemoteAccessSSHKeyPair
	return json.Marshal(struct {
		alias
		Protocol        RemoteAccessProtocol        `json:"protocol"`
		CredentialsType RemoteAccessCredentialsType `json:"credentialsType"`
	}{
		alias:           alias(r),
		Protocol:        RemoteAccessProtocolSSH,
		CredentialsType: RemoteAccessCredentialsTypeKeyPair,
	})
}

// RemoteAccessVNC is a VNC remote access configuration.
// CredentialsType can be NONE or USER_PASS; set Username and Password when using USER_PASS.
type RemoteAccessVNC struct {
	remoteAccessBase
	CredentialsType RemoteAccessCredentialsType `json:"credentialsType,omitempty"`
	Username        string                      `json:"username,omitempty"`
	Password        string                      `json:"password,omitempty"`
}

func (r RemoteAccessVNC) MarshalJSON() ([]byte, error) {
	type alias RemoteAccessVNC
	return json.Marshal(struct {
		alias
		Protocol RemoteAccessProtocol `json:"protocol"`
	}{
		alias:    alias(r),
		Protocol: RemoteAccessProtocolVNC,
	})
}

// RemoteAccessTelnet is a Telnet remote access configuration.
// CredentialsType can be NONE or USER_PASS; set Username and Password when using USER_PASS.
type RemoteAccessTelnet struct {
	remoteAccessBase
	CredentialsType RemoteAccessCredentialsType `json:"credentialsType,omitempty"`
	Username        string                      `json:"username,omitempty"`
	Password        string                      `json:"password,omitempty"`
}

func (r RemoteAccessTelnet) MarshalJSON() ([]byte, error) {
	type alias RemoteAccessTelnet
	return json.Marshal(struct {
		alias
		Protocol RemoteAccessProtocol `json:"protocol"`
	}{
		alias:    alias(r),
		Protocol: RemoteAccessProtocolTelnet,
	})
}
