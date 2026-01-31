package binaries

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiBinaries = "/inventory/binaries"
var ApiBinary = "/inventory/binaries/{id}"
var ApiManagedObject = "/inventory/managedObject/{id}"

var ParamId = "id"

const ResultProperty = "managedObjects"

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
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
type BinaryIterator struct {
	items iter.Seq[jsonmodels.Binary]
	err   error
}

func (it *BinaryIterator) Items() iter.Seq[jsonmodels.Binary] {
	return it.items
}

func (it *BinaryIterator) Err() error {
	return it.err
}

func paginateBinaries(ctx context.Context, fetch func(page int) op.Result[jsonmodels.Binary], maxItems int64) *BinaryIterator {
	iterator := &BinaryIterator{}

	iterator.items = func(yield func(jsonmodels.Binary) bool) {
		page := 1
		count := int64(0)
		for {
			result := fetch(page)
			if result.Err != nil {
				iterator.err = result.Err
				return
			}
			countBeforeResults := count
			for doc := range result.Data.Iter() {
				if maxItems > 0 && count >= maxItems {
					return
				}
				item := jsonmodels.NewBinary(doc.Bytes())
				if !yield(item) {
					return
				}
				count++
			}
			if countBeforeResults == count {
				slog.Info("Stopping pagination as results array is empty")
				return
			}

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}

// List binaries
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Binary] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewBinary)
}

// ListAll returns an iterator for all binaries
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *BinaryIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateBinaries(ctx, func(page int) op.Result[jsonmodels.Binary] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, opts.GetMaxItems())
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiBinaries)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get a binary
// TODO: How to wrap the a binary type response in op.Result? A io.Reader or io.ReadCloser might make the most sense
func (s *Service) Get(ctx context.Context, ID string) op.Result[core.BinaryResponse] {
	return core.ExecuteBinaryResponse(ctx, s.GetB(ID))
}

// TODO: For binaries the response shouldn't be read by default as this would
// result in large memory usage, however then the error handling does not work correctly
func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiBinary)
	return core.NewTryRequest(s.Client, req)
}

type UploadFileOptions = core.UploadFileOptions

// Create/Upload a binary
func (s *Service) Create(ctx context.Context, opt UploadFileOptions) op.Result[jsonmodels.Binary] {
	return core.ExecuteReturnResult(ctx, s.CreateB(opt), jsonmodels.NewBinary)
}

func (s *Service) CreateB(opt UploadFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetMultipartFields(core.NewMultiPartFileFields(opt)...).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiBinaries)
	return core.NewTryRequest(s.Client, req)
}

// Update/replace a binary
func (s *Service) Update(ctx context.Context, ID string, opt UploadFileOptions) op.Result[jsonmodels.Binary] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(ID, opt), jsonmodels.NewBinary)
}

func (s *Service) UpdateB(eventID string, opt UploadFileOptions) *core.TryRequest {
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
func (s *Service) Delete(ctx context.Context, ID string) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(ID))
}

func (s *Service) DeleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiBinary)
	return core.NewTryRequest(s.Client, req)
}
