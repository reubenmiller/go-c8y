package softwareversions

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamId = "id"

const ResultProperty = "managedObjects"

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:   *s,
		software:  softwareitems.NewService(s),
		inventory: managedobjects.NewService(s),
		binaries:  binaries.NewService(s),
	}
}

// Service api to interact with software items
// type Service core.Service
type Service struct {
	core.Service
	software  *softwareitems.Service
	inventory *managedobjects.Service
	binaries  *binaries.Service
}

// ListOptions filter software
type ListOptions struct {
	SoftwareID   string `url:"-"`
	SoftwareName string `url:"-"`

	Version string `url:"-"`

	// Pagination options
	pagination.PaginationOptions
}

func (lo ListOptions) Resolve(ctx context.Context, s *Service) ListOptions {
	if lo.SoftwareID == "" && lo.SoftwareName != "" {
		mo, found, _ := pagination.First[model.Software](ctx, s.software.ListB(softwareitems.ListOptions{
			Name: lo.SoftwareName,
		}), pagination.PagerOptions{
			PageSize: 2000,
		})
		if found {
			lo.SoftwareName = mo.ID
		}
	}
	return lo
}

// List software versions
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.SoftwareBinaryCollection, error) {
	return core.ExecuteResultOnly[model.SoftwareBinaryCollection](ctx, s.ListB(opt.Resolve(ctx, s)))
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	return s.inventory.ListB(managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddFilterEqStr("c8y_Software.version", opt.Version).
			ByGroupID(opt.SoftwareID).
			AddOrderBy("c8y_Software.version").
			AddOrderBy("creationTime").
			Build(),
		PaginationOptions: opt.PaginationOptions,
	})
}

type GetOptions struct {
	ID                string `url:"-"`
	WithParents       bool   `url:"withParents,omitempty"`
	WithChildren      bool   `url:"withChildren,omitempty"`
	withChildrenCount bool   `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool   `url:"skipChildrenNames,omitempty"`
}

type UploadFileOptions = core.UploadFileOptions

type CreateOptions struct {
	Software model.Software

	Version model.SoftwareVersion
	File    UploadFileOptions
}

// Create a software version
func (s *Service) Create(ctx context.Context, opt CreateOptions) (*model.SoftwareBinary, error) {
	item, _, err := s.software.GetOrCreate(ctx, softwareitems.GetOrCreateOptions{
		Software: opt.Software,
	})
	if err != nil {
		return nil, err
	}
	opt.Software = *item

	if opt.Version.URL == "" {
		// URL isn't provided, so it must be a binary upload
		binary, err := s.binaries.Create(ctx, binaries.UploadFileOptions(opt.File))
		if err != nil {
			return nil, err
		}
		opt.Version.URL = binary.Self
	}
	return core.ExecuteResultOnly[model.SoftwareBinary](ctx, s.CreateB(opt))
}

func (s *Service) CreateB(opt CreateOptions) *core.TryRequest {
	version := model.NewSoftwareBinary()
	version.C8Y_Software = opt.Version
	return s.inventory.ChildAdditions.CreateB(opt.Software.ID, version)
}

type GetOrCreateOptions struct {
	Software model.Software
	Version  model.SoftwareVersion

	File UploadFileOptions
}

func (s *Service) GetOrCreate(ctx context.Context, opt GetOrCreateOptions) (*model.SoftwareBinary, bool, error) {
	software, _, err := s.software.GetOrCreate(ctx, softwareitems.GetOrCreateOptions{
		Software: opt.Software,
	})
	if err != nil {
		return nil, false, err
	}
	return pagination.FindOrCreate[model.SoftwareBinary](
		ctx,
		s.ListB(ListOptions{
			SoftwareID: software.ID,
			Version:    opt.Version.Version,
		}),
		s.CreateB(CreateOptions{
			Software: *software,
			Version:  opt.Version,
			File:     opt.File,
		}),
		pagination.DefaultSearch(),
	)
}

// Get a software item
func (s *Service) Get(ctx context.Context, opt GetOptions) (*model.SoftwareBinary, error) {
	return core.ExecuteResultOnly[model.SoftwareBinary](ctx, s.GetB(opt))
}

func (s *Service) GetB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, opt.ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// Update a software item
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.SoftwareBinary, error) {
	return core.ExecuteResultOnly[model.SoftwareBinary](ctx, s.UpdateB(ID, body))
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

// DeleteOptions options to delete a software item
type DeleteOptions struct {
	ID string `url:"-"`

	// When set to true and the managed object is a device or group, all the hierarchy will be deleted
	Cascade bool `url:"cascade,omitempty"`

	// When set to true all the hierarchy will be deleted without checking the type of managed object. It takes precedence over the parameter cascade
	ForceCascade bool `url:"forceCascade,omitempty"`
}

// Delete a software item
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(opt))
}

func (s *Service) DeleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, opt.ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}
