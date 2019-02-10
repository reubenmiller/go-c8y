package c8y

import (
	"context"
	"fmt"

	"github.com/tidwall/gjson"
)

// DeviceFragmentName name of the c8yDevice Fragment property
const DeviceFragmentName = "c8y_IsDevice"

// InventoryService responsible for all inventory api calls
type InventoryService service

// ManagedObjectOptions managed object options which can be given with the managed object request
type ManagedObjectOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	WithParents bool `url:"withParents,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"query,omitempty"`

	PaginationOptions
}

// DeviceFragment Special device fragment used by Cumulocity to mark the managed objects as devices
type DeviceFragment struct{}

// EmptyFragment fragment used for special c8y fragments, i.e. c8y_IsDevice etc.
type EmptyFragment struct{}

// ConfigurationFragment fragment containing the agent's configuration information
type ConfigurationFragment struct {
	C8yConfiguration AgentConfiguration `json:"c8y_Configuration,omitempty"`
}

// SupportedOperationsFragment list of supported operations which can be sent to device/agent which has this fragment
type SupportedOperationsFragment struct {
	SupportedOperations []string `json:"c8y_SupportedOperations,omitempty"`
}

// AgentConfiguration agent configuration fragment
type AgentConfiguration struct {
	Configuration string `json:"config"`
}

// AgentFragment is the special agent fragment used to identify managed objects which are representations of an Agent.
type AgentFragment struct {
	AgentFragment struct{} `json:"com_cumulocity_model_Agent"`
}

// DeviceFragmentType marks a managed object which are device representations
type DeviceFragmentType struct {
	DeviceFragmentType struct{} `json:"c8y_IsDevice"`
}

// ManagedObject is the general Inventory Managed Object data structure
type ManagedObject struct {
	ID               string              `json:"id,omitempty"`
	Name             string              `json:"name,omitempty"`
	Type             string              `json:"type,omitempty"`
	Self             string              `json:"self,omitempty"`
	Owner            string              `json:"owner,omitempty"`
	DeviceParents    *ParentDevices      `json:"deviceParents,omitempty"`
	ChildDevices     *ChildDevices       `json:"childDevices,omitempty"`
	Kpi              *Kpi                `json:"c8y_Kpi,omitempty"`
	C8yIsDevice      *DeviceFragment     `json:"c8y_IsDevice,omitempty"`
	C8yConfiguration *AgentConfiguration `json:"c8y_Configuration,omitempty"`

	Item gjson.Result `json:"-"`
}

// NewDevice returns a simple device managed object
func NewDevice(name string) *ManagedObject {
	return &ManagedObject{
		Name:        name,
		C8yIsDevice: &DeviceFragment{},
	}
}

// Kpi is the Data Point Library fragment
type Kpi struct {
	Series   string `json:"series"`
	Fragment string `json:"fragment"`
}

// ChildDevices todo
type ChildDevices struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`
}

// ParentDevices todo
type ParentDevices struct {
	Self       string                   `json:"self"`
	References []ManagedObjectReference `json:"references"`
}

// ManagedObjectCollection todo
type ManagedObjectCollection struct {
	*BaseResponse

	ManagedObjects []ManagedObject `json:"managedObjects"`
	Items          []gjson.Result
}

// SupportedSeries is a list of the supported series in the format of <fragment>.<series>
type SupportedSeries struct {
	SupportedSeries []string `json:"c8y_SupportedSeries"`
}

// SupportedMeasurements is a list of measurement fragments for the given device
type SupportedMeasurements struct {
	SupportedMeasurements []string `json:"c8y_SupportedMeasurements"`
}

// ManagedObjectReferencesCollection Managed object references
type ManagedObjectReferencesCollection struct {
	*BaseResponse
	References []ManagedObjectReference `json:"references"`
}

// ManagedObjectReference Managed object reference
type ManagedObjectReference struct {
	Self          string        `json:"self"`
	ManagedObject ManagedObject `json:"managedObject"`
}

// GetDevicesByName returns managed object devices by filter by a name
func (s *InventoryService) GetDevicesByName(ctx context.Context, name string, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	opt := &ManagedObjectOptions{
		Query:             fmt.Sprintf("(name eq '%s') and has(%s)", name, DeviceFragmentName),
		PaginationOptions: *paging,
	}
	return s.GetManagedObjectCollection(ctx, opt)
}

// GetDevices returns the c8y device managed objects. These are the objects with the fragment "c8y_IsDevice"
func (s *InventoryService) GetDevices(ctx context.Context, paging *PaginationOptions) (*ManagedObjectCollection, *Response, error) {
	u := fmt.Sprintf("inventory/managedObjects")

	opt := &ManagedObjectOptions{
		FragmentType:      "c8y_IsDevice",
		PaginationOptions: *paging,
	}

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)

	if err != nil {
		return nil, nil, err
	}

	data := new(ManagedObjectCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Items = resp.JSON.Get("managedObjects").Array()

	return data, resp, nil
}

// All todo
func (s *ManagedObjectCollection) All() error {
	// TODO: Get All results
	return nil
}

// GetManagedObject returns a managed object by its id
func (s *InventoryService) GetManagedObject(ctx context.Context, ID string, opt *ManagedObjectOptions) (*ManagedObject, *Response, error) {
	u := fmt.Sprintf("inventory/managedObjects/%s", ID)

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(ManagedObject)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Item = *resp.JSON

	return data, resp, nil
}

// GetManagedObjectCollection todo
func (s *InventoryService) GetManagedObjectCollection(ctx context.Context, opt *ManagedObjectOptions) (*ManagedObjectCollection, *Response, error) {
	u := fmt.Sprintf("inventory/managedObjects")

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(ManagedObjectCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	data.Items = resp.JSON.Get("managedObjects").Array()

	return data, resp, nil
}

// GetSupportedSeries returns the supported series for a give device
func (s *InventoryService) GetSupportedSeries(ctx context.Context, id string) (*SupportedSeries, *Response, error) {
	u := fmt.Sprintf("/inventory/managedObjects/%s/supportedSeries", id)

	req, err := s.client.NewRequest("GET", u, "", nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(SupportedSeries)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

// GetSupportedMeasurements returns the supported measurements for a given device
func (s *InventoryService) GetSupportedMeasurements(ctx context.Context, id string) (*SupportedMeasurements, *Response, error) {
	u := fmt.Sprintf("/inventory/managedObjects/%s/supportedMeasurements", id)

	req, err := s.client.NewRequest("GET", u, "", nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(SupportedMeasurements)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

// GetManagedObjectChildDevices Get the child devices of a given managed object
func (s *InventoryService) GetManagedObjectChildDevices(ctx context.Context, id string, opt *PaginationOptions) (*ManagedObjectReferencesCollection, *Response, error) {
	u := fmt.Sprintf("inventory/managedObjects/%s/childDevices", id)

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	data := new(ManagedObjectReferencesCollection)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

// UpdateManagedObject updates a managed object
// Link: http://cumulocity.com/guides/reference/inventory
func (s *InventoryService) UpdateManagedObject(ctx context.Context, ID string, body interface{}) (*ManagedObject, *Response, error) {
	u := fmt.Sprintf("inventory/managedObjects/%s", ID)

	req, err := s.client.NewRequest("PUT", u, "", body)
	if err != nil {
		return nil, nil, err
	}

	data := new(ManagedObject)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

// CreateDevice creates a device in the Cumulocity platform with the required Device Fragment
func (s *InventoryService) CreateDevice(ctx context.Context, name string) (*ManagedObject, *Response, error) {
	return s.CreateManagedObject(ctx, NewDevice(name))
}

// CreateManagedObject create a new managed object
func (s *InventoryService) CreateManagedObject(ctx context.Context, body interface{}) (*ManagedObject, *Response, error) {
	u := fmt.Sprintf("inventory/managedObjects")

	req, err := s.client.NewRequest("POST", u, "", body)
	if err != nil {
		return nil, nil, err
	}

	data := new(ManagedObject)

	resp, err := s.client.Do(ctx, req, data)
	if err != nil {
		return nil, resp, err
	}

	return data, resp, nil
}

// Delete removes a managed object by ID
func (s *InventoryService) Delete(ctx context.Context, ID string) (*Response, error) {
	u := fmt.Sprintf("inventory/managedObjects/%s", ID)

	req, err := s.client.NewRequest("DELETE", u, "", nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(ctx, req, nil)
	if err != nil {
		return resp, err
	}

	return resp, nil
}
