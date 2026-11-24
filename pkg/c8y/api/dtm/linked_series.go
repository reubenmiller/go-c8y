package dtm

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// LinkedSeriesService reads linked series, which connect an asset series
// (fragment and series) to a device measurement series.
type LinkedSeriesService struct {
	svc    httpcore.Service
	assets *AssetService
}

// LinkedSeriesListOptions filters the linked series of one asset.
type LinkedSeriesListOptions struct {
	Fragment string `url:"fragment,omitempty"`
	Series   string `url:"series,omitempty"`
	// WithMeasurementType adds the measurement type of each source series.
	WithMeasurementType bool `url:"withMeasurementType,omitempty"`
	// WithJSONSchema adds a JSON schema describing all linked series of the
	// asset to the response envelope (see op.Result.Response).
	WithJSONSchema bool `url:"withJsonSchema,omitempty"`

	pagination.PaginationOptions
}

// List returns the linked series of asset.
//
// The OpenAPI spec describes a paged {"linkedSeries": [...]} envelope, but
// the server returns a bare JSON array (checked 2026-11-24), so List accepts
// both. With a bare array the paging options have no effect and every linked
// series is returned at once.
func (s *LinkedSeriesService) List(ctx context.Context, asset AssetRef, opt LinkedSeriesListOptions) op.Result[jsonmodels.DigitalTwinLinkedSeries] {
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinLinkedSeries](err, false)
	}
	return httpcore.Execute(ctx, s.listB(id, opt), linkedSeriesItems, meta)
}

// linkedSeriesItems returns the array of linked series from either response
// shape.
func linkedSeriesItems(b []byte) jsonmodels.DigitalTwinLinkedSeries {
	if v := gjson.GetBytes(b, ResultPropertyLinkedSeries); v.IsArray() {
		return jsonmodels.NewDigitalTwinLinkedSeries([]byte(v.Raw))
	}
	return jsonmodels.NewDigitalTwinLinkedSeries(b)
}

func (s *LinkedSeriesService) listB(assetID string, opt LinkedSeriesListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamAssetID, assetID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiAssetLinkedSeries)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultPropertyLinkedSeries)
}

// ListAll returns an iterator over all linked series of asset.
func (s *LinkedSeriesService) ListAll(ctx context.Context, asset AssetRef, opt LinkedSeriesListOptions) *pagination.Iterator[jsonmodels.DigitalTwinLinkedSeries] {
	id, err := s.assets.ResolveID(ctx, asset, nil)
	if err != nil {
		return pagination.NewErrorIterator[jsonmodels.DigitalTwinLinkedSeries](err)
	}
	return pagination.Paginate(ctx, opt.PaginationOptions, func(p pagination.PaginationOptions) op.Result[jsonmodels.DigitalTwinLinkedSeries] {
		o := opt
		o.PaginationOptions = p
		return s.List(ctx, AssetByID(id), o)
	}, jsonmodels.NewDigitalTwinLinkedSeries)
}

// LinkedSeriesCountOptions filters a linked series count.
type LinkedSeriesCountOptions struct {
	Fragment string `url:"fragment,omitempty"`
	Series   string `url:"series,omitempty"`
}

// Count returns the number of linked series of asset.
func (s *LinkedSeriesService) Count(ctx context.Context, asset AssetRef, opt LinkedSeriesCountOptions) op.Result[int64] {
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		return op.Failed[int64](err, false)
	}
	return httpcore.Execute(ctx, s.countB(id, opt), parseCount, meta)
}

func (s *LinkedSeriesService) countB(assetID string, opt LinkedSeriesCountOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamAssetID, assetID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiAssetLinkedSeriesCount)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// GetSource returns the device series that the asset series
// fragment.series reads from.
func (s *LinkedSeriesService) GetSource(ctx context.Context, asset AssetRef, fragment, series string) op.Result[jsonmodels.DigitalTwinSeriesSource] {
	if fragment == "" || series == "" {
		return op.Failed[jsonmodels.DigitalTwinSeriesSource](fmt.Errorf("fragment and series are required"), false)
	}
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinSeriesSource](err, false)
	}
	return httpcore.Execute(ctx, s.getSourceB(id, fragment, series), jsonmodels.NewDigitalTwinSeriesSource, meta)
}

func (s *LinkedSeriesService) getSourceB(assetID, fragment, series string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamAssetID, assetID).
		SetPathParam(ParamFragment, fragment).
		SetPathParam(ParamSeries, series).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiLinkedSeriesSource)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListAcrossAssetsOptions filters linked series across many assets.
type ListAcrossAssetsOptions struct {
	// AssetIDs limits the search to these assets (ids only).
	AssetIDs []string `url:"assetIds,comma,omitempty"`
	// Type filters by measurement type.
	Type     string `url:"type,omitempty"`
	Fragment string `url:"fragment,omitempty"`
	Series   string `url:"series,omitempty"`

	pagination.PaginationOptions
}

// ListAcrossAssets returns one page of assets that have linked series
// matching opt (GET /assets/linkedSeries). Each asset includes its
// c8y_LinkedSeries.
func (s *LinkedSeriesService) ListAcrossAssets(ctx context.Context, opt ListAcrossAssetsOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	return httpcore.ExecuteCollection(ctx, s.listAcrossAssetsB(opt), ResultPropertyAssets, types.ResponseFieldStatistics, jsonmodels.NewDigitalTwinAsset)
}

func (s *LinkedSeriesService) listAcrossAssetsB(opt ListAcrossAssetsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiAllLinkedSeries)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultPropertyAssets)
}

// OppositesOptions filters the assets linked to a device.
type OppositesOptions struct {
	// AssetIDs limits the result to these assets (ids only).
	AssetIDs []string `url:"assetIds,comma,omitempty"`
	Fragment string   `url:"fragment,omitempty"`
	Series   string   `url:"series,omitempty"`
}

// Opposites returns the asset series that read from device, one entry per
// linked series. device accepts the device resolver formats: an id,
// "name:<name>", "ext:<type>:<externalId>" or "query:<query>". Use it to
// check which assets depend on a device before changing or deleting it.
func (s *LinkedSeriesService) Opposites(ctx context.Context, device string, opt OppositesOptions) op.Result[jsonmodels.DigitalTwinLinkedAsset] {
	meta := map[string]any{}
	id, err := s.assets.resolveDevice(ctx, device, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinLinkedAsset](err, false)
	}
	// The response is a bare JSON array, so the whole body is the collection.
	return httpcore.ExecuteCollection(ctx, s.oppositesB(id, opt), "", "", jsonmodels.NewDigitalTwinLinkedAsset, meta)
}

func (s *LinkedSeriesService) oppositesB(deviceID string, opt OppositesOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamDeviceID, deviceID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", MimeTypeLinkedAssetsCollection).
		SetURL(ApiLinkedSeriesOpposites)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
