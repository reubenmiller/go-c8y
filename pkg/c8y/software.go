package c8y

import (
	"context"
	"fmt"
	"mime"
	"path/filepath"
)

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

// InventorySoftwareService responsible for all inventory api calls
type InventorySoftwareService service

// SoftwareOptions managed object options which can be given with the managed object request
type SoftwareOptions struct {
	WithParents bool `url:"withParents,omitempty"`

	Query string `url:"query,omitempty"`

	PaginationOptions
}

// AgentFragment is the special agent fragment used to identify managed objects which are representations of an Agent.
type SoftwareFragment struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

type InventoryDefaults struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	Self  string `json:"self,omitempty"`
	Owner string `json:"owner,omitempty"`
}

// Software is the general Inventory Managed Object data structure
type Software struct {
	ManagedObject
}

type SoftwareVersion struct {
	ManagedObject

	Software *SoftwareFragment `json:"c8y_Software,omitempty"`
}

// NewSoftware returns a simple software managed object
func NewSoftware(name string) *Software {
	return &Software{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentSoftware,
		},
	}
}

func NewSoftwareVersion(name string) *SoftwareVersion {
	return &SoftwareVersion{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentSoftwareBinary,
		},
	}
}

func GetProperties(filename string, global bool) map[string]interface{} {
	props := make(map[string]interface{})
	if global {

		props["c8y_Global"] = map[string]interface{}{}
	}
	props["name"] = filepath.Base(filename)
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return props
}

// CreateVersion upload a binary and creates a software version referencing it
// THe URL can be left blank in the software version as it will be automatically set if a filename is provided
func (s *InventorySoftwareService) CreateVersion(ctx context.Context, softwareID, filename string, version SoftwareVersion) (*ManagedObject, *Response, error) {
	// Upload file
	binaryProps := GetProperties(filename, true)
	binary, resp, err := s.client.Inventory.CreateBinary(ctx, filename, binaryProps)
	if err != nil {
		return binary, resp, err
	}

	// Create software version (as child addition of software)
	if binary != nil {
		version.Software.URL = binary.Self
	}
	mo, resp, err := s.client.Inventory.CreateChildAddition(ctx, softwareID, version)

	if err != nil {
		return mo, resp, err
	}

	// Add binary as child addition to software version managed object
	if childMO, childResp, childErr := s.client.Inventory.AddChildAddition(ctx, mo.ID, binary.ID); err != nil {
		return childMO, childResp, childErr
	}
	return mo, resp, err
}

// GetSoftwareByName returns software packages by name
func (s *InventorySoftwareService) GetSoftwareByName(ctx context.Context, name string, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("(name eq '%s') and type eq '%s'", name, FragmentSoftware),
		PaginationOptions: *paging,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}

// GetSoftwareVersionsByName returns software package versions by name
func (s *InventorySoftwareService) GetSoftwareVersionsByName(ctx context.Context, softwareID string, name string, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("(name eq '%s') and bygroupid(%s)", name, softwareID),
		PaginationOptions: *paging,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}
