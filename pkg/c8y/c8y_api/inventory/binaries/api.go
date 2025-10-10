package binaries

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"resty.dev/v3"
)

var ApiManagedObjectBinaries = "/inventory/binaries"
var ApiManagedObjectBinary = "/inventory/binaries/{id}"

const ParamId = "id"

// ManagedObjectsService inventory api to interact with managed objects
type ManagedObjectBinaryService core.Service

// ListOptions filter managed object
type ListOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"query,omitempty"`

	*GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// GetMeasurements return a measurement collection (multiple measurements)
func (s *ManagedObjectBinaryService) List(ctx context.Context, opt ListOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjectBinaries)
}

type GetOptions struct {
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	withChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
	WithLatestValues  bool `url:"withLatestValues,omitempty"`
}

/*
	{
	  "additionParents": {
	    "references": []
	  },
	  "assetParents": {
	    "references": []
	  },
	  "c8y_IsBinary": "",
	  "childAdditions": {
	    "references": [],
	    "self": "https://t493319102.eu-latest.cumulocity.com/inventory/managedObjects/104332/childAdditions"
	  },
	  "childAssets": {
	    "references": [],
	    "self": "https://t493319102.eu-latest.cumulocity.com/inventory/managedObjects/104332/childAssets"
	  },
	  "childDevices": {
	    "references": [],
	    "self": "https://t493319102.eu-latest.cumulocity.com/inventory/managedObjects/104332/childDevices"
	  },
	  "contentType": "text/csv",
	  "creationTime": "2021-03-19T15:06:28.427Z",
	  "deviceParents": {
	    "references": []
	  },
	  "id": "104332",
	  "lastUpdated": "2021-03-19T15:06:38.171Z",
	  "length": 7788940,
	  "name": "series_2021-03-19T15:06:28.427Z.csv",
	  "owner": "octocat-testuser",
	  "self": "https://t493319102.eu-latest.cumulocity.com/inventory/managedObjects/104332",
	  "type": "text/csv"
	}
*/
type BinaryManagedObject struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	// CreationTime time.Time `json:"creationTime,omitempty,omitzero"`
	LastUpdated time.Time `json:"lastUpdated,omitempty,omitzero"`
	Type        string    `json:"type,omitempty"`
	Self        string    `json:"self,omitempty"`
	Owner       string    `json:"owner,omitempty"`
	Length      int64     `json:"length,omitempty,omitzero"`
}

type Binary struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`

	AdditionalProperties map[string]any `json:"-"`
}

type CreateOptions struct {
	Object Binary
	File   *resty.MultipartField
}

func (c *CreateOptions) GetObject() Binary {
	out := Binary{
		Name:                 c.Object.Name,
		Type:                 c.Object.Type,
		AdditionalProperties: c.Object.AdditionalProperties,
	}
	if out.Name == "" {
		out.Name = "default"
	}
	if out.Type == "" {
		out.Type = "application/octet-stream"
	}
	return out
}

func (s *ManagedObjectBinaryService) Create(ctx context.Context, opts CreateOptions) *resty.Request {
	formFields := make([]*resty.MultipartField, 0, 2)

	object, _ := json.Marshal(opts.GetObject())
	formFields = append(formFields, &resty.MultipartField{
		Name:   "object",
		Reader: bytes.NewReader(object),
	})

	if opts.File != nil {
		// ensure the field name is set correctly
		opts.File.Name = "file"
		formFields = append(formFields, opts.File)
	}

	return s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", "application/json").
		SetMultipartFields(formFields...).
		SetURL(ApiManagedObjectBinaries)
}

func (s *ManagedObjectBinaryService) Get(ctx context.Context, ID string, opt GetOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjectBinary)
}

func (s *ManagedObjectBinaryService) GetMeta(ctx context.Context, ID string, opt GetOptions) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObjectBinary)
}

func (s *ManagedObjectBinaryService) Update(ctx context.Context, ID string, data any) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(data).
		SetURL(ApiManagedObjectBinary)
}

func (s *ManagedObjectBinaryService) Delete(ctx context.Context, ID string) *resty.Request {
	return s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiManagedObjectBinary)
}
