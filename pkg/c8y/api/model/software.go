package model

import "time"

type Software struct {
	ID           string    `json:"id,omitempty"`
	Name         string    `json:"name,omitempty"`
	Type         string    `json:"type,omitempty"`
	Self         string    `json:"self,omitempty"`
	Owner        string    `json:"owner,omitempty"`
	CreationTime time.Time `json:"creationTime,omitzero"`
	LastUpdated  time.Time `json:"lastUpdated,omitzero"`

	C8y_Global   *GlobalFragment `json:"c8y_Global,omitempty"`
	SoftwareType string          `json:"softwareType,omitempty"`
	Description  string          `json:"description,omitempty"`

	*ManagedObject
}

func NewSoftware(name string, softwareType string) *Software {
	return &Software{
		Type:         "c8y_Software",
		Name:         name,
		SoftwareType: softwareType,
	}
}

type SoftwareBinary struct {
	ManagedObject
	C8Y_Software SoftwareVersion `json:"c8y_Software"`
	C8Y_Global   *GlobalFragment `json:"c8y_Global,omitempty"`
}

func NewSoftwareBinary() *SoftwareBinary {
	return &SoftwareBinary{
		ManagedObject: ManagedObject{
			Type: "c8y_SoftwareBinary",
		},
		C8Y_Global: &GlobalFragment{},
	}
}

type SoftwareCollection struct {
	*BaseResponse

	ManagedObjects []Software `json:"managedObjects"`
}

type SoftwareBinaryCollection struct {
	*BaseResponse

	ManagedObjects []SoftwareBinary `json:"managedObjects"`
}

// func (i Image) MarshalJSON() ([]byte, error) {}

type SoftwareVersion struct {
	Version string `json:"version,omitempty"`
	URL     string `json:"url,omitempty"`
}

type GlobalFragment struct{}
