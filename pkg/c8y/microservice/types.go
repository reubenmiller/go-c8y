package microservice

import (
	"encoding/json"
	"io"
	"os"
)

// Manifest file name
var ManifestFile = "cumulocity.json"

// Manifest Version
type APIVersion string

const (
	APIVersion1 APIVersion = "v1"
	APIVersion2 APIVersion = "v2"
)

// Billing Mode
type BillingMode string

const (
	BillingModeResources    BillingMode = "RESOURCES"
	BillingModeSubscription BillingMode = "SUBSCRIPTION"
)

// Microservice isolation
type Isolation string

const (
	IsolationMultiTenant Isolation = "MULTI_TENANT"
	IsolationPerTenant   Isolation = "PER_TENANT"
)

// Scaling policy
type Scale string

const (
	ScaleAuto Scale = "AUTO"
	ScaleNone Scale = "NONE"
)

// Microservice manifest
type Manifest struct {
	// Document type format discriminator. The accepted values are positive integer numbers proceeded by an optional "v", such as "v2" and "2". Values which do not conform to this convention are considered "v2"
	APIVersion APIVersion `json:"apiVersion,omitempty"`

	// Application name. The accepted letters are lowercase characters (a-z), digits (0-9), or hyphens (-). The maximum length for the name is 23 characters.
	Name string `json:"name,omitempty"`

	// Microservice contextPath is used to define extension points. The accepted letters are lowercase (a-z) and uppercase (A-Z) characters, digits, hyphens (-), dots (.), underscores (_), or tildes (~).
	// Default: Microservice name
	ContextPath string `json:"contextPath,omitempty"`

	// Application version. Must be a correct SemVer value but the "+" sign is disallowed.
	Version string `json:"version,omitempty"`

	// Application provider information. Simple name allowed for predefined providers, for example, c8y. Detailed object for external provider
	Provider Provider `json:"provider,omitempty"`

	// Deployment isolation. In case of PER_TENANT, there is a separate instance for each tenant; otherwise, there is one single instance for all subscribed tenants. Should be overridable on subscription and should affect billing.
	Isolation Isolation `json:"isolation,omitempty"`

	// In case of RESOURCES, the number of resources used is exposed for billing calculation per usage. In case of SUBSCRIPTION, all resources usage is counted for the microservice owner and the subtenant is charged for subscription.
	BillingMode BillingMode `json:"billingMode,omitempty"`

	// Enables scaling policy
	Scale Scale `json:"scale,omitempty"`

	// Defines the number of microservice instances. For auto-scaled microservices, the value represents the minimum number of microservices instances
	Replicas int `json:"replicas,omitempty,omitzero"`

	// Configuration for resources limits
	// Different default values may be configured by the system administrator.
	Resources *Resources `json:"resources,omitempty"`

	// Intended configuration for minimal required resources.
	// The values may be over-written based on system settings.
	RequestedResources *RequestedResources `json:"requestedResources,omitempty"`

	// List of permissions required by a microservice to work
	RequiredRoles []string `json:"requiredRoles"`

	// Roles provided by the microservice
	Roles []string `json:"roles,omitempty"`

	// Set of tenant options available to define the configuration of a microservice
	Settings []Option `json:"settings,omitempty"`

	// Allows to specify custom category for microservice settings. By default contextPath is used
	SettingsCategory string `json:"settingsCategory,omitempty"`

	// Defines a set of extensions that should be enabled for a microservice
	Extensions []map[string]any `json:"extensions,omitempty"`

	// Defines the strategy used to verify if a microservice is alive or requires a restart. If no probe is specified, the microservice is assumed to be always healthy. We recommend that you implement liveness probes for production microservices.
	LivenessProbe *Probe `json:"livenessProbe,omitempty"`

	// Defines the strategy used to verify if a microservice is ready to accept traffic. If no probe is specified, the microservice is assumed to be always able to accept traffic immediately after it was started. Omitting the readinessProbe in production microservices will lead to clients of the microservice being exposed to startup errors
	ReadinessProbe *Probe `json:"readinessProbe,omitempty"`
}

// Create a new manifest
func NewManifest(in *Manifest, opts ...ManifestOption) (*Manifest, error) {
	if in == nil {
		in = &Manifest{}
	}
	for _, opt := range opts {
		if err := opt(in); err != nil {
			return nil, err
		}
	}
	return in, nil
}

// Manifest options
type ManifestOption func(*Manifest) error

func FromJSON(r io.Reader) ManifestOption {
	return func(m *Manifest) error {
		dec := json.NewDecoder(r)
		err := dec.Decode(m)
		return err
	}
}

func FromFile(path string) ManifestOption {
	return func(m *Manifest) error {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		return FromJSON(file)(m)
	}
}

type Probe struct {
	// Commands to be executed on a container to probe the service
	Exec *ExecAction `json:"exec,omitempty"`

	// TCP socket connection attempt as a probe
	TCPSocket *TCPSocketAction `json:"tcpSocket,omitempty"`

	// TCP socket connection attempt as a probe
	HTTPGet *HTTPGetAction `json:"httpGet,omitempty"`

	// Tells the platform for how long it should wait before performing
	// the first probe
	InitialDelaySeconds int `json:"initialDelaySeconds,omitempty,omitzero"`

	// Defines in which interval the probe should be executed
	PeriodSeconds int `json:"periodSeconds,omitempty,omitzero"`

	// Minimum consecutive successes for the probe to be considered
	// successful after having failed
	SuccessThreshold int `json:"successThreshold,omitempty,omitzero"`

	// Number of seconds after which the probe times out
	TimeoutSeconds int `json:"timeoutSeconds,omitempty,omitzero"`

	// Number of failed probes after which an action should be taken
	FailureThreshold int `json:"failureThreshold,omitempty,omitzero"`
}

type ExecAction struct {
	// Commands to be executed on a container to probe the service
	Command []string `json:"command,omitempty"`
}

type TCPSocketAction struct {
	// Host to verify
	Host string `json:"host,omitempty"`
	// Port to verify
	Port int `json:"port,omitempty"`
}
type HTTPGetAction struct {
	// Host name to connect to
	Host string `json:"host,omitempty"`

	// Path to access on the HTTP server
	Path string `json:"path,omitempty"`

	// Port to verify
	Port int `json:"port,omitempty"`

	// Scheme to use for connecting to the host (HTTP or HTTPS)
	Scheme string `json:"scheme,omitempty"`

	// HTTP headers to be added to a request
	Headers []HTTPHeader `json:"headers,omitempty"`
}

type HTTPHeader struct {
	// Header name
	Name string `json:"name,omitempty"`
	// Header value
	Value string `json:"value,omitempty"`
}

type Resources struct {
	// Limit for number of CPUs or CPU time
	// A different default value may be configured by the system administrator.
	CPU string `json:"cpu,omitempty"`

	// Limit for microservice memory usage
	// Possible units are: E, P, T, G, M, K, Ei, Pi, Ti, Gi, Mi, Ki
	// A different default value may be configured by the system administrator
	Memory string `json:"memory,omitempty"`
}

type RequestedResources struct {
	// Intended minimal requirements for number of CPUs or CPU time
	// The value may be over-written based on system settings
	CPU string `json:"cpu,omitempty"`

	// Intended minimal requirements for microservice memory usage
	// The value may be over-written based on system settings.
	// Possible units are: E, P, T, G, M, K, Ei, Pi, Ti, Gi, Mi, Ki
	// A different default value may be configured by the system administrator
	Memory string `json:"memory,omitempty"`
}

// Microservice option
type Option struct {
	// Key of the option
	Key string `json:"key,omitempty"`

	// Default value
	DefaultValue string `json:"defaultValue,omitempty"`

	// Defines if the option can be changed by a subscribed tenant on runtime
	// Default: false
	Editable *bool `json:"editable,omitempty"`

	// Defines if an editable option is reset upon microservice update
	// Default: true
	OverwriteOnUpdate *bool `json:"overwriteOnUpdate,omitempty"`

	// Specifies if an option should be inherited from the owner
	// Default: true
	InheritFromOwner *bool `json:"inheritFromOwner,omitempty"`
}

// Create a new manifest
func NewOption(in *Option, opts ...OptionFunc) *Option {
	if in == nil {
		in = &Option{}
	}
	for _, opt := range opts {
		opt(in)
	}
	return in
}

// Manifest options
type OptionFunc func(*Option)

// WithOverwriteOnUpdate set the overwrite on update value
func WithOverwriteOnUpdate(v bool) OptionFunc {
	return func(in *Option) {
		in.OverwriteOnUpdate = &v
	}
}

// WithInheritFromOwner set the inherit from owner value
func WithInheritFromOwner(v bool) OptionFunc {
	return func(in *Option) {
		in.InheritFromOwner = &v
	}
}

// Microservice provider
type Provider struct {
	// Company name of the provider
	Name string `json:"name,omitempty"`

	// Website of the provider
	Domain string `json:"domain,omitempty"`

	// Email of the support person
	Support string `json:"support,omitempty"`
}
