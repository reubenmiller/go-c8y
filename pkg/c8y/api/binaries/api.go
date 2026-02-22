package binaries

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiBinaries = "/inventory/binaries"
var ApiBinary = "/inventory/binaries/{id}"
var ApiManagedObject = "/inventory/managedObject/{id}"

var ParamId = "id"

const ResultProperty = "managedObjects"

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service struct{ core.Service }

func NewService(common *core.Service) *Service {
	return &Service{Service: *common}
}

// ListOptions to filter for specific binaries
type ListOptions struct {
	// Search for a specific child addition and list all the groups to which it belongs
	ChildAdditionID string `url:"childAdditionId,omitempty"`

	// Search for a specific child asset and list all the groups to which it belongs
	ChildAssetId string `url:"childAssetId,omitempty"`

	// Search for a specific child device and list all the groups to which it belongs
	ChildDeviceId string `url:"childDeviceId,omitempty"`

	// The managed object IDs to search for
	Ids []string `url:"ids,omitempty"`

	// Username of the owner of the managed objects
	Owner string `url:"owner,omitempty"`

	// Search for managed objects where a property value is equal to the given one. The following properties are examined: id, type, name, owner, externalIds
	Text string `url:"text,omitempty"`

	// The type of managed object to search for
	Type string `url:"type,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}

// BinaryIterator provides iteration over binaries
type BinaryIterator = pagination.Iterator[jsonmodels.Binary]

// List binaries
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Binary] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewBinary)
}

// ListAll returns an iterator for all binaries
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *BinaryIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Binary] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewBinary,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiBinaries)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get a binary
// TODO: How to wrap the a binary type response in op.Result? A io.Reader or io.ReadCloser might make the most sense
func (s *Service) Get(ctx context.Context, ID string) op.Result[core.BinaryResponse] {
	return core.ExecuteBinary(ctx, s.getB(ID))
}

// TODO: For binaries the response shouldn't be read by default as this would
// result in large memory usage, however then the error handling does not work correctly
func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiBinary)
	return core.NewTryRequest(s.Client, req)
}

type UploadFileOptions = core.UploadFileOptions

// Create/Upload a binary
func (s *Service) Create(ctx context.Context, opt UploadFileOptions) op.Result[jsonmodels.Binary] {
	return core.Execute(ctx, s.createB(opt), jsonmodels.NewBinary)
}

func (s *Service) createB(opt UploadFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetMultipartFields(core.NewMultiPartFileFields(opt)...).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiBinaries)
	return core.NewTryRequest(s.Client, req)
}

// Update/replace a binary
func (s *Service) Update(ctx context.Context, ID string, opt UploadFileOptions) op.Result[jsonmodels.Binary] {
	return core.Execute(ctx, s.updateB(ID, opt), jsonmodels.NewBinary)
}

func (s *Service) updateB(eventID string, opt UploadFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, eventID).
		SetBody(opt.GetReader()).
		SetContentType(types.MimeTypeApplicationOctetStream).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiBinary)
	return core.NewTryRequest(s.Client, req)
}

// Delete a binary
func (s *Service) Delete(ctx context.Context, ID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(ID))
}

func (s *Service) deleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiBinary)
	return core.NewTryRequest(s.Client, req)
}
