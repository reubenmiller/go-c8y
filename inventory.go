package go-c8y

import (
	"context"
	"fmt"
	"log"
)

// InventoryService does something
type InventoryService service

// ManagedObjectOptions todo
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

type EmptyFragment struct{}

// AgentConfiguration agent configuration fragment
type AgentConfiguration struct {
	Configuration string `json:"config"`
}

// ManagedObject todo
type ManagedObject struct {
	ID                   string               `json:"id"`
	Name                 string               `json:"name"`
	FarmName             string               `json:"farmName,omitempty"`
	Type                 string               `json:"type"`
	Self                 string               `json:"self"`
	Owner                string               `json:"owner"`
	DeviceParents        ParentDevices        `json:"deviceParents"`
	ChildDevices         ChildDevices         `json:"childDevices"`
	DeviceTypeDefinition DeviceTypeDefinition `json:"nx_DeviceTypeDefinition"`
	DeviceProperties     DeviceProperties     `json:"nx_DeviceProperties,omitempty"`
	NxUserSubscriptions  []UserSubscription   `json:"nx_userSubscriptions,omitempty"`
	C8yKpi               C8yKpi               `json:"c8y_Kpi,omitempty"`
	NxDevicetypeDetails  NxDevicetypeDetails  `json:"nx_devicetype_Details,omitempty"`
	C8yIsDevice          DeviceFragment       `json:"c8y_IsDevice,omitempty"`
	C8yConfiguration     AgentConfiguration   `json:"c8y_Configuration,omitempty"`
}

// C8yKpi todo
type C8yKpi struct {
	Series   string `json:"series"`
	Fragment string `json:"fragment"`
}

// NxDevicetypeDetails todo
type NxDevicetypeDetails struct {
	SampleTime int `json:"sampleTime"`
}

// UserSubscription Nx Customer User Subscriptions
type UserSubscription struct {
	PollingRate int    `json:"pollingRate"`
	KksID       string `json:"kksid"`
}

// DeviceTypeDefinition Device Type Definition containing the information model identifier
type DeviceTypeDefinition struct {
	MeasurementFragment string `json:"nx_measurementFragment"`
	Self                string `json:"self"`
}

// DeviceProperties todo
type DeviceProperties struct {
	RFCType          string `json:"RFCType"`
	TurbineType      string `json:"TurbineType"`
	SoftwareRevision string `json:"SoftwareRevision"`
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
	*C8yBaseResponse

	ManagedObjects []ManagedObject `json:"managedObjects"`
}

// SupportedSeries todo
type SupportedSeries struct {
	SupportedSeries []string `json:"c8y_SupportedSeries"`
}

// ManagedObjectReferencesCollection Managed object references
type ManagedObjectReferencesCollection struct {
	*C8yBaseResponse
	References []ManagedObjectReference `json:"references"`
}

// ManagedObjectReference Managed object reference
type ManagedObjectReference struct {
	Self          string        `json:"self"`
	ManagedObject ManagedObject `json:"managedObject"`
}

// GetDevices todo
func (s *InventoryService) GetDevices(ctx context.Context) (*ManagedObjectCollection, *Response, error) {
	u := fmt.Sprintf("inventory/managedObjects?fragment=c8y_IsDevice&pageSize=1000")

	opt := &ManagedObjectOptions{
		FragmentType: "c8y_IsDevice",
		PaginationOptions: PaginationOptions{
			PageSize:       1000,
			WithTotalPages: true,
		},
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

	return data, resp, nil
}

// All todo
func (s *ManagedObjectCollection) All() error {
	// TODO: Get All results
	return nil
}

// GetManagedObject todo
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

	return data, resp, nil
}

// GetSupportedSeries does something
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

	// println("jsonData: ", *resp.jsonData)
	log.Printf("Total count: %d\n", len(data.SupportedSeries))
	// log.Printf("Last time: %v\n", data.Measurements[0].Time)
	// log.Printf("Measurement Collection: currentPage=%d, pageSize=%v\n", *mcol.Statistics.CurrentPage, *mcol.Statistics.PageSize)

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

// UpdateManagedObject todo
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

// CreateManagedObject todo
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
