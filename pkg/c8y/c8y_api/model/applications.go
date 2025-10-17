package model

// Binary ManagedObject representation
type Application struct {
	ID                string            `json:"id,omitempty"`
	Key               string            `json:"key,omitempty"`
	Name              string            `json:"name,omitempty"`
	Type              string            `json:"type,omitempty"`
	Availability      string            `json:"availability,omitempty"`
	Self              string            `json:"self,omitempty"`
	ContextPath       string            `json:"contextPath,omitempty"`
	ExternalURL       string            `json:"externalUrl,omitempty"`
	ResourcesURL      string            `json:"resourcesUrl,omitempty"`
	ResourcesUsername string            `json:"resourcesUsername,omitempty"`
	ResourcesPassword string            `json:"resourcesPassword,omitempty"`
	Owner             *ApplicationOwner `json:"owner,omitempty"`

	// Hosted application
	ActiveVersionID string `json:"activeVersionId,omitempty"`

	// Microservice roles
	RequiredRoles []string `json:"requiredRoles,omitempty"`
	Roles         []string `json:"roles,omitempty"`

	// Application versions
	ApplicationVersions []ApplicationVersion `json:"applicationVersions,omitempty"`
}

// ApplicationOwner application owner
type ApplicationOwner struct {
	Self   string                      `json:"self,omitempty"`
	Tenant *ApplicationTenantReference `json:"tenant,omitempty"`
}

// ApplicationTenantReference tenant reference information about the application
type ApplicationTenantReference struct {
	ID string `json:"id,omitempty"`
}

// ApplicationCollection contains information about a list of applications
type ApplicationCollection struct {
	*BaseResponse

	Applications []Application `json:"applications"`
}

// Application version
type ApplicationVersion struct {
	Version  string   `json:"version,omitempty"`
	BinaryID string   `json:"binaryId,omitempty"`
	Tags     []string `json:"tags,omitempty"`

	Application *Application `json:"-,omitempty"`
}

// ApplicationVersionsCollection a list of versions related to an application
type ApplicationVersionsCollection struct {
	*BaseResponse

	Versions []ApplicationVersion `json:"applicationVersions"`
}
