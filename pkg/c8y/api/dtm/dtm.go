// Package dtm provides access to the Cumulocity Digital Twin Manager (DTM)
// API (/service/dtm): assets, their sub-assets, devices and linked series
// (the Asset API), and property, asset, measurement, event and alarm
// definitions (the Definition API).
//
// Writes follow the v2 idempotency rules: Delete and Unassign return
// StatusSkipped when the target is already gone, GetOrCreate returns
// StatusOK for an existing resource, and Upsert returns StatusSkipped when
// the existing resource already matches the desired state. See
// docs/proposals/DATAPREP_AND_DTM.md.
package dtm

import (
	"strconv"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
)

// API paths
const (
	ApiAssets                 = "/service/dtm/assets"
	ApiAssetsCount            = "/service/dtm/assets/count"
	ApiAsset                  = "/service/dtm/assets/{assetId}"
	ApiAssetByExternalID      = "/service/dtm/assets/externalIds/{externalId}"
	ApiSubAssets              = "/service/dtm/assets/{assetId}/subAssets"
	ApiSubAssetsByExternalID  = "/service/dtm/assets/externalIds/{externalId}/subAssets"
	ApiSubAssetsByParents     = "/service/dtm/assets/subAssets"
	ApiSubAsset               = "/service/dtm/assets/{assetId}/subAssets/{subAssetId}"
	ApiAssetDevices           = "/service/dtm/assets/{assetId}/devices"
	ApiAssetDevice            = "/service/dtm/assets/{assetId}/devices/{deviceId}"
	ApiLinkedSeriesLabel      = "/service/dtm/assets/{assetId}/linkedSeries/{fragment}/{series}/label"
	ApiAssetLinkedSeries      = "/service/dtm/assets/{assetId}/linkedSeries"
	ApiAssetLinkedSeriesCount = "/service/dtm/assets/{assetId}/linkedSeries/count"
	ApiLinkedSeriesSource     = "/service/dtm/assets/{assetId}/linkedSeries/{fragment}/{series}/source"
	ApiAllLinkedSeries        = "/service/dtm/assets/linkedSeries"
	ApiLinkedSeriesOpposites  = "/service/dtm/assets/linkedSeries/opposites/{deviceId}"

	ApiDefinitionsCompose = "/service/dtm/definitions/properties/compose"

	ApiDefinitions      = "/service/dtm/definitions/{kind}"
	ApiDefinition       = "/service/dtm/definitions/{kind}/{identifier}"
	ApiDefinitionsCount = "/service/dtm/definitions/{kind}/count"
	ApiDefinitionsTags  = "/service/dtm/definitions/{kind}/tags"
)

// Path parameters
const (
	ParamAssetID    = "assetId"
	ParamSubAssetID = "subAssetId"
	ParamExternalID = "externalId"
	ParamDeviceID   = "deviceId"
	ParamFragment   = "fragment"
	ParamSeries     = "series"
	ParamKind       = "kind"
	ParamIdentifier = "identifier"
)

// Collection property names in DTM responses
const (
	ResultPropertyAssets       = "assets"
	ResultPropertyDefinitions  = "definitions"
	ResultPropertyLinkedSeries = "linkedSeries"
	ResultPropertyDevices      = "devices"
)

// MimeTypeLinkedAssetsCollection selects the LinkedAsset representation of
// GET /assets/linkedSeries/opposites/{deviceId}.
const MimeTypeLinkedAssetsCollection = "application/vnd.com.nsn.cumulocity.linkedassetscollection+json"

// Service is the entry point to the Digital Twin Manager API.
type Service struct {
	// Assets covers the Asset API, including sub-assets, devices and linked
	// series.
	Assets *AssetService

	// Definitions covers the Definition API.
	Definitions *DefinitionsService
}

// NewService creates the DTM service. mo is used to resolve device
// references (for example "name:pump-01") in device arguments.
func NewService(s *httpcore.Service, mo *managedobjects.Service) *Service {
	assets := newAssetService(s, mo)
	return &Service{
		Assets:      assets,
		Definitions: newDefinitionsService(s),
	}
}

// parseCount parses the bare integer body of the DTM count endpoints.
func parseCount(b []byte) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(string(b)), 10, 64)
	return n
}
