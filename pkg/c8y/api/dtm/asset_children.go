package dtm

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// SubAssetService reads the sub-assets of assets.
type SubAssetService struct {
	svc    httpcore.Service
	assets *AssetService
}

// SubAssetListOptions filters and shapes a sub-asset list.
type SubAssetListOptions struct {
	// Query is a DTM query applied to the sub-assets.
	Query string `url:"query,omitempty"`
	// IncludeGroups also returns device groups that are not assets.
	IncludeGroups bool `url:"includeGroups,omitempty"`

	WithLinkedSeries  bool `url:"withLinkedSeries,omitempty"`
	WithParents       bool `url:"withParents,omitempty"`
	WithChildrenCount bool `url:"withChildrenCount,omitempty"`

	pagination.PaginationOptions
}

// List returns one page of the sub-assets of parent. An ext: reference uses
// /assets/externalIds/{externalId}/subAssets, so it needs no lookup.
func (s *SubAssetService) List(ctx context.Context, parent AssetRef, opt SubAssetListOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	opt.Query = normalizeQuery(opt.Query)
	if ext, ok := parent.externalID(); ok {
		return httpcore.ExecuteCollection(ctx, s.listB(ApiSubAssetsByExternalID, ParamExternalID, ext, opt), ResultPropertyAssets, types.ResponseFieldStatistics, jsonmodels.NewDigitalTwinAsset)
	}
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, parent, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
	}
	return httpcore.ExecuteCollection(ctx, s.listB(ApiSubAssets, ParamAssetID, id, opt), ResultPropertyAssets, types.ResponseFieldStatistics, jsonmodels.NewDigitalTwinAsset, meta)
}

func (s *SubAssetService) listB(path, param, value string, opt SubAssetListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(param, value).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(path)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultPropertyAssets)
}

// ListAll returns an iterator over all sub-assets of parent. The parent is
// resolved once, before the first page.
func (s *SubAssetService) ListAll(ctx context.Context, parent AssetRef, opt SubAssetListOptions) *AssetIterator {
	parent, err := s.assets.pinRef(ctx, parent)
	if err != nil {
		return pagination.NewErrorIterator[jsonmodels.DigitalTwinAsset](err)
	}
	return pagination.Paginate(ctx, opt.PaginationOptions, func(p pagination.PaginationOptions) op.Result[jsonmodels.DigitalTwinAsset] {
		o := opt
		o.PaginationOptions = p
		return s.List(ctx, parent, o)
	}, jsonmodels.NewDigitalTwinAsset)
}

// ListByParentsOptions selects the parents and shapes the result of
// ListByParents.
type ListByParentsOptions struct {
	// Parents are asset ids. Resolver references are not supported here;
	// resolve them with AssetService.ResolveID first.
	Parents []string `url:"parents,comma"`

	Query             string `url:"query,omitempty"`
	IncludeGroups     bool   `url:"includeGroups,omitempty"`
	WithLinkedSeries  bool   `url:"withLinkedSeries,omitempty"`
	WithParents       bool   `url:"withParents,omitempty"`
	WithChildrenCount bool   `url:"withChildrenCount,omitempty"`

	pagination.PaginationOptions
}

// ListByParents returns one page of the sub-assets of several parents in one
// request (GET /assets/subAssets?parents=...).
func (s *SubAssetService) ListByParents(ctx context.Context, opt ListByParentsOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	if len(opt.Parents) == 0 {
		return op.Failed[jsonmodels.DigitalTwinAsset](fmt.Errorf("at least one parent id is required"), false)
	}
	opt.Query = normalizeQuery(opt.Query)
	return httpcore.ExecuteCollection(ctx, s.listByParentsB(opt), ResultPropertyAssets, types.ResponseFieldStatistics, jsonmodels.NewDigitalTwinAsset)
}

func (s *SubAssetService) listByParentsB(opt ListByParentsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiSubAssetsByParents)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultPropertyAssets)
}

// AssetDeviceService reads the devices assigned to assets.
type AssetDeviceService struct {
	svc    httpcore.Service
	assets *AssetService
}

// DeviceListOptions filters a device list.
type DeviceListOptions struct {
	// Query is a DTM query applied to the devices.
	Query string `url:"query,omitempty"`

	pagination.PaginationOptions
}

// List returns one page of the devices assigned to asset. Devices are plain
// managed objects.
func (s *AssetDeviceService) List(ctx context.Context, asset AssetRef, opt DeviceListOptions) op.Result[jsonmodels.ManagedObject] {
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	opt.Query = normalizeQuery(opt.Query)
	return httpcore.ExecuteCollection(ctx, s.listB(id, opt), ResultPropertyDevices, types.ResponseFieldStatistics, jsonmodels.NewManagedObject, meta)
}

func (s *AssetDeviceService) listB(assetID string, opt DeviceListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamAssetID, assetID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiAssetDevices)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultPropertyDevices)
}

// ListAll returns an iterator over all devices assigned to asset.
func (s *AssetDeviceService) ListAll(ctx context.Context, asset AssetRef, opt DeviceListOptions) *pagination.Iterator[jsonmodels.ManagedObject] {
	id, err := s.assets.ResolveID(ctx, asset, nil)
	if err != nil {
		return pagination.NewErrorIterator[jsonmodels.ManagedObject](err)
	}
	return pagination.Paginate(ctx, opt.PaginationOptions, func(p pagination.PaginationOptions) op.Result[jsonmodels.ManagedObject] {
		o := opt
		o.PaginationOptions = p
		return s.List(ctx, AssetByID(id), o)
	}, jsonmodels.NewManagedObject)
}

// pinRef resolves name: and query: references once, so iterators don't
// repeat the lookup for every page. ext: references are kept, because the
// endpoints that accept them need no lookup.
func (s *AssetService) pinRef(ctx context.Context, ref AssetRef) (AssetRef, error) {
	if _, ok := ref.externalID(); ok {
		return ref, nil
	}
	id, err := s.ResolveID(ctx, ref, nil)
	if err != nil {
		return "", err
	}
	return AssetByID(id), nil
}

// resolveDevice resolves a device reference (id, name:, ext:<type>:<id>,
// query:) with the managed object device resolver.
func (s *AssetService) resolveDevice(ctx context.Context, device string, meta map[string]any) (string, error) {
	if device == "" {
		return "", fmt.Errorf("empty device reference")
	}
	if s.deviceResolver == nil {
		return device, nil
	}
	return s.deviceResolver.ResolveID(ctxhelpers.ResolutionContext(ctx), managedobjects.DeviceRef(device), meta)
}
