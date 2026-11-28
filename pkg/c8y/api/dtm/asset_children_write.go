package dtm

import (
	"context"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// Create creates an asset and assigns it to parent in one request
// (POST /assets/{id}/subAssets). The response, and so result.Data, is the
// parent asset.
func (s *SubAssetService) Create(ctx context.Context, parent AssetRef, body any) op.Result[jsonmodels.DigitalTwinAsset] {
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, parent, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
	}
	return httpcore.Execute(ctx, s.createB(id, body), jsonmodels.NewDigitalTwinAsset, meta)
}

func (s *SubAssetService) createB(parentID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(paramAssetID, parentID).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiSubAssets)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// GetOrCreate returns the asset with opt.ExternalID and makes sure it is a
// sub-asset of parent. A missing asset is created from opt.Body and assigned
// in one request; an existing asset is assigned (the server ignores a repeat
// assignment) but not updated. result.Data is the sub-asset.
//
// Status: StatusCreated (new) or StatusOK (existed).
func (s *SubAssetService) GetOrCreate(ctx context.Context, parent AssetRef, opt AssetCreateOptions, annotations ...model.Fragment) op.Result[jsonmodels.DigitalTwinAsset] {
	return deferrable(ctx, "getOrCreateSubAsset", func(ctx context.Context) op.Result[jsonmodels.DigitalTwinAsset] {
		body, err := opt.body(annotations...)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
		}
		parentID, err := s.assets.ResolveID(ctx, parent, nil)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
		}

		existing := s.assets.GetByExternalID(ctx, opt.ExternalID, GetOptions{})
		switch {
		case existing.Err == nil:
			if res := s.Assign(ctx, AssetByID(parentID), AssetByID(existing.Data.ID())); res.Err != nil {
				return op.Failed[jsonmodels.DigitalTwinAsset](res.Err, false)
			}
			return op.WithMeta(op.WithMeta(existing, "existing", true), "parentId", parentID)
		case !core.IsNotFound(existing.Err):
			return existing
		}

		if res := s.Create(ctx, AssetByID(parentID), body); res.Err != nil {
			return res
		}
		created := s.assets.GetByExternalID(ctx, opt.ExternalID, GetOptions{})
		if created.Err != nil {
			return created
		}
		created.Status = op.StatusCreated
		return op.WithMeta(created, "parentId", parentID)
	})
}

// Assign makes child a sub-asset of parent. Both must exist. Assigning
// again is safe: the server answers 201 again and adds no duplicate.
// result.Data is the parent asset.
func (s *SubAssetService) Assign(ctx context.Context, parent, child AssetRef) op.Result[jsonmodels.DigitalTwinAsset] {
	meta := map[string]any{}
	parentID, childID, err := s.assets.resolvePair(ctx, parent, child, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
	}
	return httpcore.Execute(ctx, s.assets.linkB(resty.MethodPost, apiSubAsset, paramSubAssetID, parentID, childID), jsonmodels.NewDigitalTwinAsset, meta)
}

// Unassign removes child from parent without deleting it. A child that is
// not assigned, or a parent or child that does not exist, returns
// StatusSkipped without an error.
func (s *SubAssetService) Unassign(ctx context.Context, parent, child AssetRef) op.Result[core.NoContent] {
	meta := map[string]any{}
	parentID, childID, err := s.assets.resolvePair(ctx, parent, child, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	return unassignResult(httpcore.ExecuteNoContent(ctx, s.assets.linkB(resty.MethodDelete, apiSubAsset, paramSubAssetID, parentID, childID), meta))
}

// Assign assigns device to asset. device accepts the device resolver
// formats (id, "name:", "ext:<type>:<id>", "query:"). Assigning again is
// safe. result.Data is the asset.
func (s *AssetDeviceService) Assign(ctx context.Context, asset AssetRef, device string) op.Result[jsonmodels.DigitalTwinAsset] {
	meta := map[string]any{}
	assetID, deviceID, err := s.assets.resolveAssetAndDevice(ctx, asset, device, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
	}
	return httpcore.Execute(ctx, s.assets.linkB(resty.MethodPost, apiAssetDevice, paramDeviceID, assetID, deviceID), jsonmodels.NewDigitalTwinAsset, meta)
}

// Unassign removes device from asset without deleting it. A device that is
// not assigned, or an asset or device that does not exist, returns
// StatusSkipped without an error.
func (s *AssetDeviceService) Unassign(ctx context.Context, asset AssetRef, device string) op.Result[core.NoContent] {
	meta := map[string]any{}
	assetID, deviceID, err := s.assets.resolveAssetAndDevice(ctx, asset, device, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	return unassignResult(httpcore.ExecuteNoContent(ctx, s.assets.linkB(resty.MethodDelete, apiAssetDevice, paramDeviceID, assetID, deviceID), meta))
}

// unassignResult maps "already unassigned" responses to StatusSkipped: the
// 422 "is not a child of object" the server returns for a child that is not
// assigned, and a 404 for a parent or child that does not exist.
func unassignResult(r op.Result[core.NoContent]) op.Result[core.NoContent] {
	if isNotAChild(r.Err) {
		r.HTTPStatus = http.StatusUnprocessableEntity
		return skippedFrom(r, "not assigned")
	}
	return skipIfNotFound(r)
}

func (s *AssetService) linkB(method, path, childParam, assetID, childID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(method).
		SetPathParam(paramAssetID, assetID).
		SetPathParam(childParam, childID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(path)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *AssetService) resolvePair(ctx context.Context, parent, child AssetRef, meta map[string]any) (string, string, error) {
	parentID, err := s.ResolveID(ctx, parent, nil)
	if err != nil {
		return "", "", err
	}
	childID, err := s.ResolveID(ctx, child, nil)
	if err != nil {
		return "", "", err
	}
	meta["parentId"] = parentID
	meta["childId"] = childID
	return parentID, childID, nil
}

func (s *AssetService) resolveAssetAndDevice(ctx context.Context, asset AssetRef, device string, meta map[string]any) (string, string, error) {
	assetID, err := s.ResolveID(ctx, asset, nil)
	if err != nil {
		return "", "", err
	}
	deviceID, err := s.resolveDevice(ctx, device, nil)
	if err != nil {
		return "", "", err
	}
	meta["assetId"] = assetID
	meta["deviceId"] = deviceID
	return assetID, deviceID, nil
}
