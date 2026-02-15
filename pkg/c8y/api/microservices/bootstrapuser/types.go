package bootstrapuser

// BootstrapUser is the representation of the bootstrap user for microservices
type BootstrapUser struct {
	Username string `json:"name,omitempty"`
	Password string `json:"password,omitempty"`
	Tenant   string `json:"tenant,omitempty"`
}
