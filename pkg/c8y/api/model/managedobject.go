package model

import (
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
)

// DeviceFragment marks a managed object which are device representations
type DeviceFragment struct {
	DeviceFragment map[string]any `json:"c8y_IsDevice"`
}

// AgentFragment is the special agent fragment used to identify managed objects which are representations of an Agent.
type AgentFragment struct {
	AgentFragment map[string]any `json:"com_cumulocity_model_Agent"`
}

// ManagedObject is the general Inventory Managed Object data structure
type ManagedObject struct {
	ID           string    `json:"id,omitempty"`
	Name         string    `json:"name,omitempty"`
	Type         string    `json:"type,omitempty"`
	Self         string    `json:"self,omitempty"`
	Owner        string    `json:"owner,omitempty"`
	CreationTime time.Time `json:"creationTime,omitzero"`
	LastUpdated  time.Time `json:"lastUpdated,omitzero"`

	DeviceParents   *ManagedObjectReferenceCollection `json:"deviceParents,omitempty"`
	ChildDevices    *ManagedObjectReferenceCollection `json:"childDevices,omitempty"`
	AdditionParents *ManagedObjectReferenceCollection `json:"additionParents,omitempty"`
	AssetParents    *ManagedObjectReferenceCollection `json:"assetParents,omitempty"`
	ChildAdditions  *ManagedObjectReferenceCollection `json:"childAdditions,omitempty"`
	ChildAssets     *ManagedObjectReferenceCollection `json:"childAssets,omitempty"`
}

type ManagedObjectCollection struct {
	*BaseResponse

	ManagedObjects []ManagedObject `json:"managedObjects"`
}

type ManagedObjectReferenceCollection struct {
	Self       string                   `json:"self,omitempty"`
	References []ManagedObjectReference `json:"references,omitempty"`
}

type ManagedObjectReference struct {
	Self          string        `json:"self,omitempty"`
	ManagedObject ManagedObject `json:"managedObject,omitempty"`
}

func NewManagedObjectReference(ID string) *ManagedObjectReference {
	return &ManagedObjectReference{
		ManagedObject: ManagedObject{
			ID: ID,
		},
	}
}

func FromManagedObjectChildReferences(value any) (string, any) {
	switch v := value.(type) {
	case string:
		return types.MimeTypeManagedObjectReference, NewManagedObjectReference(v)
	case []string:
		data := &ManagedObjectReferenceCollection{
			References: []ManagedObjectReference{},
		}
		for _, ID := range v {
			data.References = append(data.References, *NewManagedObjectReference(ID))
		}
		return types.MimeTypeManagedObjectCollection, data
	default:
		return types.MimeTypeManagedObject, v
	}
}

func ToManagedObjectChildReferences(value any) *ManagedObjectReferenceCollection {
	data := &ManagedObjectReferenceCollection{
		References: []ManagedObjectReference{},
	}

	switch v := value.(type) {
	case string:
		data.References = append(data.References, *NewManagedObjectReference(v))
	case []string:
		for _, ID := range v {
			data.References = append(data.References, *NewManagedObjectReference(ID))
		}
	case ManagedObjectReference:
		data.References = append(data.References, v)
	case []ManagedObjectReference:
		data.References = append(data.References, v...)
	}
	return data
}
