package dtm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// LinkedSeries is a linked series to add or update. Fragment and Series
// identify it on the asset.
type LinkedSeries struct {
	Fragment string        `json:"fragment"`
	Series   string        `json:"series"`
	Label    string        `json:"label,omitempty"`
	Source   *SeriesSource `json:"source,omitempty"`
}

// SeriesSource is the device series a linked series reads from.
type SeriesSource struct {
	// ID is the device (managed object) id.
	ID       string `json:"id,omitempty"`
	Fragment string `json:"fragment,omitempty"`
	Series   string `json:"series,omitempty"`
	// Type is the measurement type. By default the server needs it: set it
	// here, or set FetchTypeFromSource to read it from an existing
	// measurement of the device.
	Type string `json:"type,omitempty"`
}

// Set adds new linked series to asset and updates existing ones, matched by
// fragment and series, in one request (POST /assets/{id}/linkedSeries). An
// update keeps the existing source id unless the item sets one.
// result.Data is the asset.
func (s *LinkedSeriesService) Set(ctx context.Context, asset AssetRef, series []LinkedSeries, opt WriteOptions) op.Result[jsonmodels.DigitalTwinAsset] {
	if len(series) == 0 {
		return op.Failed[jsonmodels.DigitalTwinAsset](fmt.Errorf("at least one linked series is required"), false)
	}
	for _, ls := range series {
		if ls.Fragment == "" || ls.Series == "" {
			return op.Failed[jsonmodels.DigitalTwinAsset](fmt.Errorf("linked series need a fragment and a series"), false)
		}
	}
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinAsset](err, false)
	}
	return httpcore.Execute(ctx, s.setB(id, series, opt), jsonmodels.NewDigitalTwinAsset, meta)
}

func (s *LinkedSeriesService) setB(assetID string, series []LinkedSeries, opt WriteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamAssetID, assetID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(series).
		SetURL(ApiAssetLinkedSeries)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type seriesKey struct {
	Fragment string `url:"fragment"`
	Series   string `url:"series"`
}

// Delete removes the linked series fragment.series from asset. A linked
// series or asset that does not exist returns StatusSkipped.
func (s *LinkedSeriesService) Delete(ctx context.Context, asset AssetRef, fragment, series string) op.Result[core.NoContent] {
	if fragment == "" || series == "" {
		return op.Failed[core.NoContent](fmt.Errorf("fragment and series are required"), false)
	}
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamAssetID, id).
		SetQueryParamsFromValues(httpcore.QueryParameters(seriesKey{fragment, series})).
		SetURL(ApiAssetLinkedSeries)
	return skipIfNotFound(httpcore.ExecuteNoContent(ctx, httpcore.NewTryRequest(s.svc.Client, req), meta))
}

// SetSource points the linked series fragment.series of asset at a device
// series (POST .../source). source.ID accepts the device resolver formats
// (id, "name:", "ext:<type>:<id>", "query:").
func (s *LinkedSeriesService) SetSource(ctx context.Context, asset AssetRef, fragment, series string, source SeriesSource, opt WriteOptions) op.Result[jsonmodels.DigitalTwinSeriesSource] {
	if fragment == "" || series == "" {
		return op.Failed[jsonmodels.DigitalTwinSeriesSource](fmt.Errorf("fragment and series are required"), false)
	}
	meta := map[string]any{}
	assetID, deviceID, err := s.assets.resolveAssetAndDevice(ctx, asset, source.ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinSeriesSource](err, false)
	}
	source.ID = deviceID
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamAssetID, assetID).
		SetPathParam(ParamFragment, fragment).
		SetPathParam(ParamSeries, series).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(source).
		SetURL(ApiLinkedSeriesSource)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), jsonmodels.NewDigitalTwinSeriesSource, meta)
}

// DeleteSource removes the source of the linked series fragment.series,
// keeping the linked series itself. A missing source returns StatusSkipped.
func (s *LinkedSeriesService) DeleteSource(ctx context.Context, asset AssetRef, fragment, series string) op.Result[core.NoContent] {
	if fragment == "" || series == "" {
		return op.Failed[core.NoContent](fmt.Errorf("fragment and series are required"), false)
	}
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamAssetID, id).
		SetPathParam(ParamFragment, fragment).
		SetPathParam(ParamSeries, series).
		SetURL(ApiLinkedSeriesSource)
	return skipIfNotFound(httpcore.ExecuteNoContent(ctx, httpcore.NewTryRequest(s.svc.Client, req), meta))
}

// SetLabel changes the label of the linked series fragment.series.
func (s *LinkedSeriesService) SetLabel(ctx context.Context, asset AssetRef, fragment, series, label string) op.Result[jsonmodels.DigitalTwinLinkedSeries] {
	if fragment == "" || series == "" {
		return op.Failed[jsonmodels.DigitalTwinLinkedSeries](fmt.Errorf("fragment and series are required"), false)
	}
	meta := map[string]any{}
	id, err := s.assets.ResolveID(ctx, asset, meta)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinLinkedSeries](err, false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamAssetID, id).
		SetPathParam(ParamFragment, fragment).
		SetPathParam(ParamSeries, series).
		SetContentType(types.MimeTypeTextPlain).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(label).
		SetURL(ApiLinkedSeriesLabel)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), jsonmodels.NewDigitalTwinLinkedSeries, meta)
}

// LinkOptions describes a link from an asset series to a device series.
type LinkOptions struct {
	// Fragment and Series name the series on the asset. Required.
	Fragment string
	Series   string
	// Label is the display name. Empty keeps the existing label.
	Label string

	// Device is the device providing the data, in any device resolver
	// format (id, "name:", "ext:<type>:<id>", "query:"). Required.
	Device string
	// SourceFragment and SourceSeries name the series on the device.
	// They default to Fragment and Series.
	SourceFragment string
	SourceSeries   string
	// SourceType is the measurement type of the device series. Unless the
	// tenant option assets.linkedSeries.source.measurementType.mode says
	// otherwise, the server needs it: either set it here, or let the
	// server read it from an existing measurement (FetchTypeFromSource).
	// Without either, the server answers 400 "Failed to fetch the
	// measurement type".
	SourceType string

	WriteOptions
}

// Link makes the asset series opt.Fragment.opt.Series read from a device
// series. It resolves the asset and the device, and returns StatusSkipped
// without a write when the link already points to the same device series
// (and has the requested label). Otherwise it adds or updates the linked
// series in one request.
//
// Status: StatusCreated (new linked series), StatusUpdated or StatusSkipped.
// result.Data is the linked series as read back from the server.
func (s *LinkedSeriesService) Link(ctx context.Context, asset AssetRef, opt LinkOptions) op.Result[jsonmodels.DigitalTwinLinkedSeries] {
	return deferrable(ctx, "link", func(ctx context.Context) op.Result[jsonmodels.DigitalTwinLinkedSeries] {
		if opt.Fragment == "" || opt.Series == "" || opt.Device == "" {
			return op.Failed[jsonmodels.DigitalTwinLinkedSeries](fmt.Errorf("fragment, series and device are required"), false)
		}
		meta := map[string]any{}
		assetID, deviceID, err := s.assets.resolveAssetAndDevice(ctx, asset, opt.Device, meta)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinLinkedSeries](err, false)
		}
		desired := desiredLink(opt, deviceID)

		current, found, res := s.find(ctx, assetID, opt.Fragment, opt.Series)
		if res.Err != nil {
			return res
		}
		if found && linkMatches(desired, current) {
			return skippedFrom(op.OK(current, meta), "no changes detected")
		}

		set := s.Set(ctx, AssetByID(assetID), []LinkedSeries{desired}, opt.WriteOptions)
		if set.Err != nil {
			return op.Failed[jsonmodels.DigitalTwinLinkedSeries](set.Err, false)
		}
		after, ok, res := s.find(ctx, assetID, opt.Fragment, opt.Series)
		if res.Err != nil {
			return res
		}
		if !ok {
			return op.Failed[jsonmodels.DigitalTwinLinkedSeries](fmt.Errorf("linked series %s.%s not found after update", opt.Fragment, opt.Series), false)
		}
		if found {
			return op.Updated(after, meta)
		}
		return op.Created(after, meta)
	})
}

// find returns the linked series fragment.series of an asset.
func (s *LinkedSeriesService) find(ctx context.Context, assetID, fragment, series string) (jsonmodels.DigitalTwinLinkedSeries, bool, op.Result[jsonmodels.DigitalTwinLinkedSeries]) {
	list := s.List(ctx, AssetByID(assetID), LinkedSeriesListOptions{Fragment: fragment, Series: series})
	if list.Err != nil {
		return jsonmodels.DigitalTwinLinkedSeries{}, false, list
	}
	for item := range list.Seq() {
		if item.Fragment() == fragment && item.Series() == series {
			return item, true, list
		}
	}
	return jsonmodels.DigitalTwinLinkedSeries{}, false, list
}

// linkMatches reports whether current already has the desired source and
// label. An empty desired label matches any label.
func linkMatches(desired LinkedSeries, current jsonmodels.DigitalTwinLinkedSeries) bool {
	b, err := json.Marshal(desired)
	if err != nil {
		return false
	}
	m := jsonObject(b)
	if desired.Label == "" {
		delete(m, "label")
	}
	return op.DesiredStateMatches(m, current.Bytes())
}

func orDefault(v, fallback string) string {
	if v != "" {
		return v
	}
	return fallback
}
