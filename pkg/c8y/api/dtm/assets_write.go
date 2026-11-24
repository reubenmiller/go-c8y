package dtm

import (
	"context"
	"fmt"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// FragmentExternalID is the asset fragment holding the DTM external id.
const FragmentExternalID = "c8y_ExternalId"

// HeaderUpsertMode makes POST /assets update an asset with the same
// c8y_ExternalId instead of failing with 409.
const HeaderUpsertMode = "X-Upsert-Mode"

// WriteOptions are the query options shared by asset writes.
type WriteOptions struct {
	// FetchTypeFromSource fetches the measurement type of linked series
	// sources from their recorded measurements. The tenant option
	// assets.linkedSeries.source.measurementType.mode can override it.
	FetchTypeFromSource bool `url:"fetchTypeFromSource,omitempty"`
}

// Create creates an asset (POST /assets). If the body has a c8y_ExternalId
// that is already used, the server returns 409; use GetOrCreate or Upsert to
// handle that case.
func (s *AssetService) Create(ctx context.Context, body any, opt WriteOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	return httpcore.Execute(ctx, s.createB(body, opt, false), jsonmodels.NewDigitalTwinAsset)
}

func (s *AssetService) createB(body any, opt WriteOptions, upsert bool) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAssets)
	if upsert {
		req.SetHeader(HeaderUpsertMode, "true")
	}
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update updates an asset (PUT /assets/{id}).
func (s *AssetService) Update(ctx context.Context, ref AssetRef, body any, opt WriteOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	meta := map[string]any{}
	id, err := s.ResolveID(ctx, ref, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
	}
	return httpcore.Execute(ctx, s.updateB(id, body, opt), jsonmodels.NewDigitalTwinAsset, meta)
}

func (s *AssetService) updateB(id string, body any, opt WriteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamAssetID, id).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiAsset)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Cascade selects what Delete removes besides the asset. Combine values with
// |. The zero value removes only the asset; sub-assets and devices are
// unassigned.
type Cascade int

const (
	// CascadeSubAssets also deletes the sub-asset hierarchy.
	CascadeSubAssets Cascade = 1 << iota
	// CascadeDevices also deletes devices whose parents are all being
	// deleted.
	CascadeDevices
)

// DeleteOptions controls an asset delete.
type DeleteOptions struct {
	// Cascade selects what else is deleted. There is no default, so devices
	// are never deleted by accident. The API keeps sub-assets and devices
	// that also belong to a parent outside the deleted hierarchy.
	Cascade Cascade
}

type deleteQuery struct {
	DeleteSubAssets bool `url:"deleteSubAssets,omitempty"`
	DeleteDevices   bool `url:"deleteDevices,omitempty"`
}

// Delete deletes an asset. An asset that does not exist (including a name:
// or query: reference that matches nothing) returns StatusSkipped without
// an error, so Delete is safe to call in defer.
func (s *AssetService) Delete(ctx context.Context, ref AssetRef, opt DeleteOptions) op.Result[core.NoContent] {
	meta := map[string]any{"cascade": opt.Cascade}
	id, err := s.ResolveID(ctx, ref, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	return skipIfNotFound(httpcore.ExecuteNoContent(ctx, s.deleteB(id, opt), meta))
}

func (s *AssetService) deleteB(id string, opt DeleteOptions) *httpcore.TryRequest {
	q := deleteQuery{
		DeleteSubAssets: opt.Cascade&CascadeSubAssets != 0,
		DeleteDevices:   opt.Cascade&CascadeDevices != 0,
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamAssetID, id).
		SetQueryParamsFromValues(httpcore.QueryParameters(q)).
		SetURL(ApiAsset)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// AssetCreateOptions describes an asset identified by its external id.
type AssetCreateOptions struct {
	// ExternalID is written to c8y_ExternalId and identifies the asset.
	// Required.
	ExternalID string
	// Body is the asset (a map or a struct), without the external id.
	Body any

	WriteOptions
}

// body returns the request body with c8y_ExternalId set.
func (o AssetCreateOptions) body(annotations ...model.Fragment) (map[string]any, error) {
	if o.ExternalID == "" {
		return nil, fmt.Errorf("ExternalID is required")
	}
	m, err := toMap(o.Body, annotations...)
	if err != nil {
		return nil, err
	}
	if v, ok := m[FragmentExternalID]; ok && v != o.ExternalID {
		return nil, fmt.Errorf("body %s %v does not match ExternalID %q", FragmentExternalID, v, o.ExternalID)
	}
	m[FragmentExternalID] = o.ExternalID
	return m, nil
}

// GetOrCreate returns the asset with opt.ExternalID, creating it from
// opt.Body when it does not exist. It never updates an existing asset.
//
// Status: StatusCreated (new) or StatusOK (existed, no write).
//
// annotations are merged into the body when the asset is created.
func (s *AssetService) GetOrCreate(ctx context.Context, opt AssetCreateOptions, annotations ...model.Fragment) op.Result[jsonmodels.DigitalTwinAsset] {
	return deferrable(ctx, "getOrCreate", func(ctx context.Context) op.Result[jsonmodels.DigitalTwinAsset] {
		body, err := opt.body(annotations...)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
		}
		res := s.Create(ctx, body, opt.WriteOptions)
		if !core.ErrHasStatus(res.Err, http.StatusConflict) {
			return res
		}
		// The external id is taken: return the existing asset.
		existing := s.GetByExternalID(ctx, opt.ExternalID, GetOptions{})
		if existing.Err != nil {
			return existing
		}
		return op.WithMeta(existing, "existing", true)
	})
}

// AssetUpsertOptions describes the desired state of an asset identified by
// its external id.
type AssetUpsertOptions struct {
	AssetCreateOptions

	// SkipCompare sends the server-side upsert directly, without reading
	// the asset first. It is one request instead of two, but always writes
	// and never returns StatusSkipped.
	SkipCompare bool
}

// Upsert creates the asset with opt.ExternalID or updates it to match
// opt.Body, using the server-side upsert (POST /assets with X-Upsert-Mode).
//
// The server upsert always writes, even when nothing changed. Unless
// SkipCompare is set, Upsert therefore reads the asset first and returns
// StatusSkipped, with no write, when every field in the body already has the
// same value (a subset match: fields the body does not mention are ignored).
//
// Status: StatusCreated, StatusUpdated or StatusSkipped.
//
// annotations (for example a sync timestamp) are written on create and on
// every real update, but are not compared, so they never cause an update on
// their own.
func (s *AssetService) Upsert(ctx context.Context, opt AssetUpsertOptions, annotations ...model.Fragment) op.Result[jsonmodels.DigitalTwinAsset] {
	return deferrable(ctx, "upsert", func(ctx context.Context) op.Result[jsonmodels.DigitalTwinAsset] {
		desired, err := opt.body()
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
		}
		body, err := opt.body(annotations...)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
		}

		if !opt.SkipCompare {
			existing := s.GetByExternalID(ctx, opt.ExternalID, GetOptions{})
			switch {
			case existing.Err == nil:
				if op.DesiredStateMatches(desired, existing.Data.Bytes()) {
					return skippedFrom(existing, "no changes detected")
				}
			case !core.IsNotFound(existing.Err):
				return existing
			}
		}
		return s.serverUpsert(ctx, body, opt.WriteOptions)
	})
}

// serverUpsert sends POST /assets with X-Upsert-Mode: true. The server
// answers 201 for a new asset and 200 for an updated one.
func (s *AssetService) serverUpsert(ctx context.Context, body any, opt WriteOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	res := httpcore.Execute(ctx, s.createB(body, opt, true), jsonmodels.NewDigitalTwinAsset)
	if res.Err == nil && res.HTTPStatus == http.StatusOK {
		res.Status = op.StatusUpdated
	}
	return res
}
