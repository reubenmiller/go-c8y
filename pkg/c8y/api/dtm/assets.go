package dtm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// AssetIterator iterates DTM assets across pages.
type AssetIterator = pagination.Iterator[jsonmodels.DigitalTwinAsset]

// ErrAmbiguousAssetRef is returned when a name: or query: reference matches
// more than one asset.
var ErrAmbiguousAssetRef = errors.New("asset reference matches more than one asset")

// AssetRef references an asset. It is one of:
//
//   - "12345": an asset id, used as-is
//   - "ext:<externalId>": the asset's c8y_ExternalId (identity type c8y_Asset)
//   - "name:<name>": the one asset with this exact name
//   - "query:<filter>": the one asset matching a DTM $filter expression
//
// Use the constructors (AssetByID, AssetByExternalID, AssetByName,
// AssetByQuery) or cast a string.
type AssetRef string

func AssetByID(id string) AssetRef                 { return AssetRef(id) }
func AssetByExternalID(externalID string) AssetRef { return AssetRef(schemeExternalID + externalID) }
func AssetByName(name string) AssetRef             { return AssetRef(schemeName + name) }
func AssetByQuery(filter string) AssetRef          { return AssetRef(schemeQuery + filter) }

const (
	schemeExternalID = "ext:"
	schemeName       = "name:"
	schemeQuery      = "query:"
)

// externalID returns the external id when ref uses the ext: scheme.
func (r AssetRef) externalID() (string, bool) {
	return strings.CutPrefix(string(r), schemeExternalID)
}

// AssetService reads DTM assets.
type AssetService struct {
	svc            httpcore.Service
	deviceResolver *managedobjects.DeviceResolver

	// SubAssets reads the sub-assets of an asset.
	SubAssets *SubAssetService
	// Devices reads the devices assigned to an asset.
	Devices *AssetDeviceService
	// LinkedSeries reads the linked series of assets.
	LinkedSeries *LinkedSeriesService
}

func newAssetService(s *httpcore.Service, mo *managedobjects.Service) *AssetService {
	a := &AssetService{svc: *s}
	if mo != nil {
		a.deviceResolver = managedobjects.NewDeviceResolver(mo)
	}
	a.SubAssets = &SubAssetService{svc: *s, assets: a}
	a.Devices = &AssetDeviceService{svc: *s, assets: a}
	a.LinkedSeries = &LinkedSeriesService{svc: *s, assets: a}
	return a
}

// GetOptions selects what an asset response includes.
type GetOptions struct {
	// WithLinkedSeries adds the c8y_LinkedSeries fragment.
	WithLinkedSeries bool `url:"withLinkedSeries,omitempty"`
	// WithSubAssets adds the subAssets array. Deprecated by the API in favour
	// of SubAssets.List.
	WithSubAssets bool `url:"withSubAssets,omitempty"`
	// WithLatestValues adds c8y_LatestMeasurements for the linked series.
	// Only supported when getting a single asset by id.
	WithLatestValues bool `url:"withLatestValues,omitempty"`
	// WithParents adds the assetParents array.
	WithParents bool `url:"withParents,omitempty"`
	// WithChildrenCount adds assignedChildrenCount (sub-assets and devices).
	WithChildrenCount bool `url:"withChildrenCount,omitempty"`
}

// Get returns one asset. An ext: reference is fetched directly from
// /assets/externalIds/{externalId}; name: and query: references are resolved
// to an id first.
func (s *AssetService) Get(ctx context.Context, ref AssetRef, opt GetOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	if ext, ok := ref.externalID(); ok {
		return s.GetByExternalID(ctx, ext, opt)
	}
	meta := map[string]any{}
	id, err := s.ResolveID(ctx, ref, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
	}
	return httpcore.Execute(ctx, s.getB(id, opt), jsonmodels.NewDigitalTwinAsset, meta)
}

func (s *AssetService) getB(id string, opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(paramAssetID, id).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiAsset)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// GetByExternalID returns the asset whose c8y_ExternalId is externalID.
// WithLatestValues is not supported by this endpoint and is not sent.
func (s *AssetService) GetByExternalID(ctx context.Context, externalID string, opt GetOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	if externalID == "" {
		return op.Failed[jsonmodels.DigitalTwinAsset](fmt.Errorf("empty asset external id"), false)
	}
	opt.WithLatestValues = false
	return httpcore.Execute(ctx, s.getByExternalIDB(externalID, opt), jsonmodels.NewDigitalTwinAsset)
}

func (s *AssetService) getByExternalIDB(externalID string, opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(paramExternalID, externalID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiAssetByExternalID)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListOptions filters and shapes an asset list.
type ListOptions struct {
	// Query is a DTM query, for example built with model.NewInventoryQuery
	// ("$filter=(name eq 'Pump 07')"). A bare filter expression without
	// "$filter=" is wrapped automatically.
	Query string `url:"query,omitempty"`

	// OnlyRoots returns only assets without a parent asset.
	OnlyRoots bool `url:"onlyRoots,omitempty"`
	// IncludeGroups also returns device groups that are not assets.
	IncludeGroups bool `url:"includeGroups,omitempty"`

	WithLinkedSeries  bool `url:"withLinkedSeries,omitempty"`
	WithSubAssets     bool `url:"withSubAssets,omitempty"`
	WithParents       bool `url:"withParents,omitempty"`
	WithChildrenCount bool `url:"withChildrenCount,omitempty"`

	pagination.PaginationOptions
}

// List returns one page of assets.
func (s *AssetService) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	opt.Query = normalizeQuery(opt.Query)
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultPropertyAssets, types.ResponseFieldStatistics, jsonmodels.NewDigitalTwinAsset)
}

func (s *AssetService) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiAssets)
	return httpcore.NewTryRequest(s.svc.Client, req, resultPropertyAssets)
}

// ListAll returns an iterator over all assets matching opt.
func (s *AssetService) ListAll(ctx context.Context, opt ListOptions) *AssetIterator {
	return pagination.Paginate(ctx, opt.PaginationOptions, func(p pagination.PaginationOptions) op.Result[jsonmodels.DigitalTwinAsset] {
		o := opt
		o.PaginationOptions = p
		return s.List(ctx, o)
	}, jsonmodels.NewDigitalTwinAsset)
}

// CountOptions filters an asset count.
type CountOptions struct {
	Query         string `url:"query,omitempty"`
	OnlyRoots     bool   `url:"onlyRoots,omitempty"`
	IncludeGroups bool   `url:"includeGroups,omitempty"`
}

// Count returns the number of assets matching opt.
func (s *AssetService) Count(ctx context.Context, opt CountOptions) op.Result[int64] {
	opt.Query = normalizeQuery(opt.Query)
	return httpcore.Execute(ctx, s.countB(opt), parseCount)
}

func (s *AssetService) countB(opt CountOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiAssetsCount)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ResolveID resolves an asset reference to an asset id. A plain id is
// returned without a request. The lookups run even under dry run and
// deferred execution, so a prepared request carries the real id. meta, when
// not nil, receives the resolved "id" and "name" and the "source" scheme.
//
// A name: or query: reference must match exactly one asset; no match returns
// a 404 error (core.IsNotFound) and several matches return
// ErrAmbiguousAssetRef.
func (s *AssetService) ResolveID(ctx context.Context, ref AssetRef, meta map[string]any) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup("asset", string(ref), resolveErr) }()
	value := string(ref)
	if value == "" {
		return "", fmt.Errorf("empty asset reference")
	}
	ctx = ctxhelpers.ResolutionContext(ctx)

	if ext, ok := ref.externalID(); ok {
		res := s.GetByExternalID(ctx, ext, GetOptions{})
		if res.Err != nil {
			return "", res.Err
		}
		setMeta(meta, res.Data, "externalId")
		return res.Data.ID(), nil
	}
	if name, ok := strings.CutPrefix(value, schemeName); ok {
		query := model.NewInventoryQuery().AddFilterEqStr("name", name).Build()
		return s.lookupOne(ctx, query, value, "name", meta)
	}
	if filter, ok := strings.CutPrefix(value, schemeQuery); ok {
		return s.lookupOne(ctx, normalizeQuery(filter), value, "query", meta)
	}
	return value, nil
}

func (s *AssetService) lookupOne(ctx context.Context, query, label, source string, meta map[string]any) (string, error) {
	opt := ListOptions{Query: query}
	opt.PageSize = 2
	res := s.List(ctx, opt)
	if res.Err != nil {
		return "", res.Err
	}
	var found []jsonmodels.DigitalTwinAsset
	for item := range res.Seq() {
		found = append(found, item)
	}
	switch len(found) {
	case 0:
		return "", core.ErrNotFound("asset not found: %s", label)
	case 1:
		setMeta(meta, found[0], source)
		return found[0].ID(), nil
	default:
		return "", fmt.Errorf("%w: %s", ErrAmbiguousAssetRef, label)
	}
}

func setMeta(meta map[string]any, asset jsonmodels.DigitalTwinAsset, source string) {
	if meta == nil {
		return
	}
	meta["id"] = asset.ID()
	meta["name"] = asset.Name()
	meta["source"] = source
}

// normalizeQuery prefixes a bare filter expression with "$filter=", so both
// "name eq 'x'" and "$filter=(name eq 'x')" are accepted.
func normalizeQuery(q string) string {
	q = strings.TrimSpace(q)
	if q == "" || strings.HasPrefix(q, "$") {
		return q
	}
	return "$filter=(" + q + ")"
}
