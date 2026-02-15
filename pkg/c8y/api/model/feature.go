package model

// Cumulocity feature phases
const (
	FeaturePhaseInDevelopment      = "IN_DEVELOPMENT"
	FeaturePhasePrivatePreview     = "PRIVATE_PREVIEW"
	FeaturePhasePublicPreview      = "PUBLIC_PREVIEW"
	FeaturePhaseGenerallyAvailable = "GENERALLY_AVAILABLE"
)

// Cumulocity feature strategy
const (
	FeatureStrategyDefault = "DEFAULT"
	FeatureStrategyTenant  = "TENANT"
)

// Feature representation
type Feature struct {
	// A unique key of the feature toggle
	Key string `json:"key,omitempty"`

	// Current phase of feature toggle rollout.
	Phase string `json:"phase,omitempty"`

	// Current value of the feature toggle marking whether the feature is active or not.
	Active bool `json:"active"`

	// The source of the feature toggle value - either it's feature toggle definition provided default, or per tenant provided override.
	Strategy string `json:"strategy,omitempty"`

	// Tenant id where the feature is active (only set when using the by-tenant api)
	TenantId string `json:"tenantId,omitempty"`
}

// Check if the feature toggle is set to the default for the tenant
func (f *Feature) IsDefault() bool {
	return f.Strategy == FeatureStrategyDefault
}
