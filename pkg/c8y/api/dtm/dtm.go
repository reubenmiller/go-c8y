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
	apiAssets                 = "/service/dtm/assets"
	apiAssetsCount            = "/service/dtm/assets/count"
	apiAsset                  = "/service/dtm/assets/{assetId}"
	apiAssetByExternalID      = "/service/dtm/assets/externalIds/{externalId}"
	apiSubAssets              = "/service/dtm/assets/{assetId}/subAssets"
	apiSubAssetsByExternalID  = "/service/dtm/assets/externalIds/{externalId}/subAssets"
	apiSubAssetsByParents     = "/service/dtm/assets/subAssets"
	apiSubAsset               = "/service/dtm/assets/{assetId}/subAssets/{subAssetId}"
	apiAssetDevices           = "/service/dtm/assets/{assetId}/devices"
	apiAssetDevice            = "/service/dtm/assets/{assetId}/devices/{deviceId}"
	apiLinkedSeriesLabel      = "/service/dtm/assets/{assetId}/linkedSeries/{fragment}/{series}/label"
	apiAssetLinkedSeries      = "/service/dtm/assets/{assetId}/linkedSeries"
	apiAssetLinkedSeriesCount = "/service/dtm/assets/{assetId}/linkedSeries/count"
	apiLinkedSeriesSource     = "/service/dtm/assets/{assetId}/linkedSeries/{fragment}/{series}/source"
	apiAllLinkedSeries        = "/service/dtm/assets/linkedSeries"
	apiLinkedSeriesOpposites  = "/service/dtm/assets/linkedSeries/opposites/{deviceId}"

	apiDefinitionsCompose = "/service/dtm/definitions/properties/compose"

	apiDefinitions      = "/service/dtm/definitions/{kind}"
	apiDefinition       = "/service/dtm/definitions/{kind}/{identifier}"
	apiDefinitionsCount = "/service/dtm/definitions/{kind}/count"
	apiDefinitionsTags  = "/service/dtm/definitions/{kind}/tags"
)

// Path parameters
const (
	paramAssetID    = "assetId"
	paramSubAssetID = "subAssetId"
	paramExternalID = "externalId"
	paramDeviceID   = "deviceId"
	paramFragment   = "fragment"
	paramSeries     = "series"
	paramKind       = "kind"
	paramIdentifier = "identifier"
)

// Collection property names in DTM responses
const (
	resultPropertyAssets       = "assets"
	resultPropertyDefinitions  = "definitions"
	resultPropertyLinkedSeries = "linkedSeries"
	resultPropertyDevices      = "devices"
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
