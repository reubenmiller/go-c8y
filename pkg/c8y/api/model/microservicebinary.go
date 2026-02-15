package model

// MicroserviceBinary representation
type MicroserviceBinary struct {
	ManagedObject

	Manifest MicroserviceManifest `json:"com_cumulocity_model_application_MicroserviceManifest"`
}
