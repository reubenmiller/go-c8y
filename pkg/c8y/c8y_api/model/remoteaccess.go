package model

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

// TODO: Create remote access structs for the different types of configuration types as each type supports a subset of properties
// and this makes it annoying to use
