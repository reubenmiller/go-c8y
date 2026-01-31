package softwareversions

import (
	"context"
	"fmt"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamId = "id"

const ResultProperty = "managedObjects"

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		software:       softwareitems.NewService(s),
		managedObjects: managedobjects.NewService(s),
		binaries:       binaries.NewService(s),
	}
}

// Service api to interact with software versions
type Service struct {
	core.Service
	software       *softwareitems.Service
	managedObjects *managedobjects.Service
	binaries       *binaries.Service
}

// Create a software version
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.SoftwareVersion] {
	return core.ExecuteReturnResult(ctx, s.managedObjects.CreateB(body), jsonmodels.NewSoftwareVersion)
}

// ListOptions filter software versions
type ListOptions struct {
	SoftwareID   string `url:"-"`
	SoftwareName string `url:"-"`
	Version      string `url:"-"`

	// Pagination options
	CurrentPage int `url:"-"`
	PageSize    int `url:"-"`
}

// List software versions
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.SoftwareVersion] {
	// Resolve software name to ID if needed
	softwareID := opt.SoftwareID
	if softwareID == "" && opt.SoftwareName != "" {
		softwareResult := s.software.Get(ctx, softwareitems.GetOptions{
			Name: opt.SoftwareName,
		})
		if softwareResult.Err != nil {
			return op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("failed to resolve software name: %w", softwareResult.Err),
				true,
			)
		}
		softwareID = softwareResult.Data.ID()
	}

	return core.ExecuteReturnCollection(ctx, s.ListB(softwareID, opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSoftwareVersion)
}

func (s *Service) ListB(softwareID string, opt ListOptions) *core.TryRequest {
	return s.managedObjects.ListB(managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddFilterEqStr("c8y_Software.version", opt.Version).
			ByGroupID(softwareID).
			AddOrderBy("c8y_Software.version").
			AddOrderBy("creationTime").
			Build(),
		PaginationOptions: pagination.PaginationOptions{
			CurrentPage: opt.CurrentPage,
			PageSize:    opt.PageSize,
		},
	})
}

// SoftwareVersionIterator provides iteration over software versions
type SoftwareVersionIterator struct {
	items iter.Seq[jsonmodels.SoftwareVersion]
	err   error
}

func (it *SoftwareVersionIterator) Items() iter.Seq[jsonmodels.SoftwareVersion] {
	return it.items
}

func (it *SoftwareVersionIterator) Err() error {
	return it.err
}

func paginateSoftwareVersions(ctx context.Context, fetch func(page int) op.Result[jsonmodels.SoftwareVersion], maxItems int) *SoftwareVersionIterator {
	iterator := &SoftwareVersionIterator{}

	iterator.items = func(yield func(jsonmodels.SoftwareVersion) bool) {
		page := 1
		count := 0
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
				item := jsonmodels.NewSoftwareVersion(doc.Bytes())
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

// ListAll returns an iterator for all software versions
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SoftwareVersionIterator {
	return paginateSoftwareVersions(ctx, func(page int) op.Result[jsonmodels.SoftwareVersion] {
		opts.CurrentPage = page
		opts.PageSize = 2000
		return s.List(ctx, opts)
	}, 0)
}

// ListLimit returns an iterator for up to maxItems software versions
func (s *Service) ListLimit(ctx context.Context, opts ListOptions, maxItems int) *SoftwareVersionIterator {
	return paginateSoftwareVersions(ctx, func(page int) op.Result[jsonmodels.SoftwareVersion] {
		opts.CurrentPage = page
		opts.PageSize = 2000
		return s.List(ctx, opts)
	}, maxItems)
}

type GetOptions struct {
	// Lookup strategies (at least one required)
	ID      string `url:"-"`
	Version string `url:"-"` // Requires SoftwareID or SoftwareName

	// For version-based lookup
	SoftwareID   string `url:"-"`
	SoftwareName string `url:"-"`

	// Query options
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	withChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// UpdateOptions options for updating a software version
type UpdateOptions struct {
	// Lookup strategies (at least one required)
	ID      string `url:"-"`
	Version string `url:"-"`

	// For version-based lookup
	SoftwareID   string `url:"-"`
	SoftwareName string `url:"-"`
}

// DeleteOptions options to delete a software version
type DeleteOptions struct {
	// Lookup strategies (at least one required)
	ID      string `url:"-"`
	Version string `url:"-"`

	// For version-based lookup
	SoftwareID   string `url:"-"`
	SoftwareName string `url:"-"`

	// Delete options
	ForceCascade bool `url:"forceCascade,omitempty"`
}

// resolveID resolves a software version ID from various lookup strategies
func (s *Service) resolveID(ctx context.Context, id, version, softwareID, softwareName string) (string, op.Result[jsonmodels.SoftwareVersion]) {
	// Direct ID provided
	if id != "" {
		return id, op.Result[jsonmodels.SoftwareVersion]{}
	}

	// Lookup by version (requires software ID or name)
	if version != "" {
		// Resolve software ID from name if needed
		resolvedSoftwareID := softwareID
		if resolvedSoftwareID == "" && softwareName != "" {
			softwareResult := s.software.Get(ctx, softwareitems.GetOptions{
				Name: softwareName,
			})
			if softwareResult.Err != nil {
				return "", op.Failed[jsonmodels.SoftwareVersion](
					fmt.Errorf("failed to resolve software name: %w", softwareResult.Err),
					true,
				)
			}
			resolvedSoftwareID = softwareResult.Data.ID()
		}

		if resolvedSoftwareID == "" {
			return "", op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("version lookup requires SoftwareID or SoftwareName"),
				false,
			)
		}

		listResult := s.List(ctx, ListOptions{
			SoftwareID: resolvedSoftwareID,
			Version:    version,
			PageSize:   1,
		})

		if listResult.Err != nil {
			return "", op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("failed to lookup version: %w", listResult.Err),
				true,
			)
		}

		// Check if any items were found
		for item := range listResult.Data.Iter() {
			return jsonmodels.NewSoftwareVersion(item.Bytes()).ID(), op.Result[jsonmodels.SoftwareVersion]{}
		}

		return "", op.Failed[jsonmodels.SoftwareVersion](
			fmt.Errorf("version not found: software=%s, version=%s", resolvedSoftwareID, version),
			false,
		)
	}

	// No lookup strategy provided
	return "", op.Failed[jsonmodels.SoftwareVersion](
		fmt.Errorf("no lookup strategy provided: must specify ID or Version (with SoftwareID/SoftwareName)"),
		false,
	)
}

// Get a software version
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.SoftwareVersion] {
	id, resolveResult := s.resolveID(ctx, opt.ID, opt.Version, opt.SoftwareID, opt.SoftwareName)
	if resolveResult.Err != nil {
		return resolveResult
	}

	result := core.ExecuteReturnResult(ctx, s.GetB(id, opt), jsonmodels.NewSoftwareVersion)

	// Add lookup metadata
	if opt.ID != "" {
		result.Meta["lookupMethod"] = "id"
	} else if opt.Version != "" {
		result.Meta["lookupMethod"] = "version"
		result.Meta["lookupVersion"] = opt.Version
		if opt.SoftwareName != "" {
			result.Meta["lookupSoftwareName"] = opt.SoftwareName
		} else {
			result.Meta["lookupSoftwareID"] = opt.SoftwareID
		}
	}

	return result
}

// Update a software version
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.SoftwareVersion] {
	id, resolveResult := s.resolveID(ctx, opt.ID, opt.Version, opt.SoftwareID, opt.SoftwareName)
	if resolveResult.Err != nil {
		return resolveResult
	}

	result := core.ExecuteReturnResult(ctx, s.UpdateB(id, body, opt), jsonmodels.NewSoftwareVersion)

	// Add lookup metadata
	if opt.ID != "" {
		result.Meta["lookupMethod"] = "id"
	} else if opt.Version != "" {
		result.Meta["lookupMethod"] = "version"
		result.Meta["lookupVersion"] = opt.Version
	}

	return result
}

// Delete a software version
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[jsonmodels.SoftwareVersion] {
	id, resolveResult := s.resolveID(ctx, opt.ID, opt.Version, opt.SoftwareID, opt.SoftwareName)
	if resolveResult.Err != nil {
		return resolveResult
	}

	result := core.ExecuteReturnResult(ctx, s.DeleteB(id, opt), jsonmodels.NewSoftwareVersion)

	// Add lookup metadata
	if opt.ID != "" {
		result.Meta["lookupMethod"] = "id"
	} else if opt.Version != "" {
		result.Meta["lookupMethod"] = "version"
		result.Meta["lookupVersion"] = opt.Version
	}

	return result
}

type UploadFileOptions = core.UploadFileOptions

type CreateOptions struct {
	SoftwareName string
	SoftwareType string
	SoftwareID   string

	Version string
	URL     string
	File    UploadFileOptions
}

// CreateVersion creates a software version, automatically handling software item lookup/creation and binary upload
func (s *Service) CreateVersion(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.SoftwareVersion] {
	// Step 1: Get or create software item
	var softwareID string
	if opt.SoftwareID != "" {
		softwareID = opt.SoftwareID
	} else if opt.SoftwareName != "" {
		softwareResult := s.software.GetOrCreateByName(ctx, opt.SoftwareName, opt.SoftwareType, map[string]any{
			"name":         opt.SoftwareName,
			"type":         "c8y_Software",
			"softwareType": opt.SoftwareType,
		})
		if softwareResult.Err != nil {
			return op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("failed to get/create software item: %w", softwareResult.Err),
				true,
			)
		}
		softwareID = softwareResult.Data.ID()
	} else {
		return op.Failed[jsonmodels.SoftwareVersion](
			fmt.Errorf("must specify SoftwareID or SoftwareName"),
			false,
		)
	}

	// Step 2: Upload binary if file provided and URL not specified
	url := opt.URL
	if url == "" && opt.File.Name != "" {
		binaryResult := s.binaries.Create(ctx, opt.File)
		if binaryResult.IsError() {
			return op.Failed[jsonmodels.SoftwareVersion](
				fmt.Errorf("failed to upload binary: %w", binaryResult.Err),
				true,
			)
		}
		url = binaryResult.Data.Self()
	}

	// Step 3: Create software version
	versionBody := map[string]any{
		"type": "c8y_SoftwareBinary",
		"c8y_Software": map[string]any{
			"version": opt.Version,
		},
	}
	if url != "" {
		versionBody["c8y_Software"].(map[string]any)["url"] = url
	}

	return core.ExecuteReturnResult(ctx, s.CreateB(softwareID, versionBody), jsonmodels.NewSoftwareVersion)
}

// GetOrCreateVersion searches by software + version, creating if not found
func (s *Service) GetOrCreateVersion(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.SoftwareVersion] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.SoftwareVersion], bool) {
		// First resolve software ID
		softwareID := opt.SoftwareID
		if softwareID == "" && opt.SoftwareName != "" {
			softwareResult := s.software.Get(ctx, softwareitems.GetOptions{
				Name:         opt.SoftwareName,
				SoftwareType: opt.SoftwareType,
			})
			if softwareResult.Err != nil {
				return op.Result[jsonmodels.SoftwareVersion]{}, false
			}
			softwareID = softwareResult.Data.ID()
		}

		if softwareID == "" {
			return op.Result[jsonmodels.SoftwareVersion]{}, false
		}

		// Search for version
		listResult := s.List(ctx, ListOptions{
			SoftwareID: softwareID,
			Version:    opt.Version,
			PageSize:   1,
		})

		if listResult.Err != nil {
			return op.Result[jsonmodels.SoftwareVersion]{}, false
		}

		// Check if any items were found
		for item := range listResult.Data.Iter() {
			version := jsonmodels.NewSoftwareVersion(item.Bytes())
			result := op.OK(version)
			result.HTTPStatus = listResult.HTTPStatus
			result.Meta["found"] = true
			result.Meta["lookupMethod"] = "version"
			return result, true
		}

		return op.Result[jsonmodels.SoftwareVersion]{}, false
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.SoftwareVersion] {
		createResult := s.CreateVersion(ctx, opt)
		if createResult.Err != nil {
			return createResult
		}
		createResult.Meta["found"] = false
		return createResult
	}

	// Execute get-or-create pattern
	return op.GetOrCreateR(ctx, finder, creator)
}

// Builder methods

func (s *Service) CreateB(softwareID string, body any) *core.TryRequest {
	return s.managedObjects.ChildAdditions.CreateB(softwareID, body)
}

func (s *Service) GetB(ID string, opt GetOptions) *core.TryRequest {
	return s.managedObjects.GetB(ID, managedobjects.GetOptions{
		WithParents: opt.WithParents,
	})
}

func (s *Service) UpdateB(ID string, body any, opt UpdateOptions) *core.TryRequest {
	return s.managedObjects.UpdateB(ID, body)
}

func (s *Service) DeleteB(ID string, opt DeleteOptions) *core.TryRequest {
	return s.managedObjects.DeleteB(ID, managedobjects.DeleteOptions{
		ForceCascade: opt.ForceCascade,
	})
}
