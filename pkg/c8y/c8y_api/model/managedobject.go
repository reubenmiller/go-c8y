package model

var MimeTypeManagedObjectCollection = "application/vnd.com.nsn.cumulocity.managedobjectreferencecollection+json"
var MimeTypeManagedObject = "application/vnd.com.nsn.cumulocity.managedobject+json"
var MimeTypeManagedObjectReference = "application/vnd.com.nsn.cumulocity.managedobjectreference+json"

// ManagedObject is the general Inventory Managed Object data structure
type ManagedObject struct {
	ID              string                            `json:"id,omitempty"`
	Name            string                            `json:"name,omitempty"`
	Type            string                            `json:"type,omitempty"`
	Self            string                            `json:"self,omitempty"`
	Owner           string                            `json:"owner,omitempty"`
	DeviceParents   *ManagedObjectReferenceCollection `json:"deviceParents,omitempty"`
	ChildDevices    *ManagedObjectReferenceCollection `json:"childDevices,omitempty"`
	AdditionParents *ManagedObjectReferenceCollection `json:"additionParents,omitempty"`
	AssetParents    *ManagedObjectReferenceCollection `json:"assetParents,omitempty"`
	ChildAdditions  *ManagedObjectReferenceCollection `json:"childAdditions,omitempty"`
	ChildAssets     *ManagedObjectReferenceCollection `json:"childAssets,omitempty"`
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
		return MimeTypeManagedObjectReference, NewManagedObjectReference(v)
	case []string:
		data := &ManagedObjectReferenceCollection{
			References: []ManagedObjectReference{},
		}
		for _, ID := range v {
			data.References = append(data.References, *NewManagedObjectReference(ID))
		}
		return MimeTypeManagedObjectCollection, data
	default:
		return MimeTypeManagedObject, v
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
