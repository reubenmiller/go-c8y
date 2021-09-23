package c8y

import (
	"context"
	"fmt"
	"net/url"
)

const FragmentFirmware = "c8y_Firmware"
const FragmentFirmwareBinary = "c8y_FirmwareBinary"

// InventoryFirmwareService responsible for all inventory api calls
type InventoryFirmwareService service

// FirmwareOptions managed object options which can be given with the managed object request
type FirmwareOptions struct {
	WithParents bool `url:"withParents,omitempty"`

	Query string `url:"query,omitempty"`

	PaginationOptions
}

// AgentFragment is the special agent fragment used to identify managed objects which are representations of an Agent.
type FirmwareFragment struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

// Firmware is the general Inventory Managed Object data structure
type Firmware struct {
	ManagedObject
}

type FirmwareVersion struct {
	ManagedObject

	Firmware *FirmwareFragment `json:"c8y_Firmware,omitempty"`
}

// NewFirmware returns a simple firmware managed object
func NewFirmware(name string) *Firmware {
	return &Firmware{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentFirmware,
		},
	}
}

func NewFirmwareVersion(name string) *FirmwareVersion {
	return &FirmwareVersion{
		ManagedObject: ManagedObject{
			Name: name,
			Type: FragmentFirmwareBinary,
		},
	}
}

// CreateVersion upload a binary and creates a firmware version referencing it
// THe URL can be left blank in the firmware version as it will be automatically set if a filename is provided
func (s *InventoryFirmwareService) CreateVersion(ctx context.Context, firmwareID, filename string, version FirmwareVersion) (*ManagedObject, *Response, error) {
	return s.client.Inventory.CreateChildAdditionWithBinary(ctx, firmwareID, filename, func(binaryURL string) interface{} {
		version.Firmware.URL = binaryURL
		return version
	})
}

// GetFirmwareByName returns firmware packages by name
func (s *InventoryFirmwareService) GetFirmwareByName(ctx context.Context, name string, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("(name eq '%s') and type eq '%s' $orderby=creationTime,name", url.QueryEscape(name), FragmentFirmware),
		PaginationOptions: *paging,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}

// GetFirmwareVersionsByName returns firmware package versions by name
func (s *InventoryFirmwareService) GetFirmwareVersionsByName(ctx context.Context, firmwareID string, name string, withParents bool, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("(c8y_Firmware.version eq '%s') and bygroupid(%s) $orderby=creationTime,c8y_Firmware.version", url.QueryEscape(name), firmwareID),
		PaginationOptions: *paging,
		WithParents:       withParents,
	}
	return s.client.Inventory.GetManagedObjects(ctx, opt)
}
