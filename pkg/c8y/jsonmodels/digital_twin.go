package jsonmodels

import (
	"iter"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// seqOf yields one model per element of a JSON array.
func seqOf[T any](arr gjson.Result, ctor func([]byte) T) iter.Seq[T] {
	return func(yield func(T) bool) {
		if !arr.IsArray() {
			return
		}
		for _, item := range arr.Array() {
			if !yield(ctor([]byte(item.Raw))) {
				return
			}
		}
	}
}

// stringsOf returns the string elements of a JSON array (nil when absent).
func stringsOf(arr gjson.Result) []string {
	if !arr.IsArray() {
		return nil
	}
	items := arr.Array()
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.String())
	}
	return out
}

// DigitalTwinAsset is an asset returned by the Digital Twin Manager (DTM)
// Asset API (/service/dtm/assets).
type DigitalTwinAsset struct {
	jsondoc.Facade
}

func NewDigitalTwinAsset(b []byte) DigitalTwinAsset {
	return DigitalTwinAsset{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (a DigitalTwinAsset) ID() string              { return a.Get("id").String() }
func (a DigitalTwinAsset) Name() string            { return a.Get("name").String() }
func (a DigitalTwinAsset) Type() string            { return a.Get("type").String() }
func (a DigitalTwinAsset) Owner() string           { return a.Get("owner").String() }
func (a DigitalTwinAsset) CreationTime() time.Time { return a.Get("creationTime").Time() }
func (a DigitalTwinAsset) LastUpdated() time.Time  { return a.Get("lastUpdated").Time() }

// ExternalID returns the c8y_ExternalId fragment (identity type c8y_Asset).
func (a DigitalTwinAsset) ExternalID() string { return a.Get("c8y_ExternalId").String() }

// Notes returns the asset description (c8y_Notes).
func (a DigitalTwinAsset) Notes() string { return a.Get("c8y_Notes").String() }

// IsDeviceGroup reports whether the asset is also a device group
// (c8y_IsDeviceGroup), which is the case for assets created by the DTM.
func (a DigitalTwinAsset) IsDeviceGroup() bool { return a.Get("c8y_IsDeviceGroup").Exists() }

// ChildrenCount returns assignedChildrenCount, which is only present when the
// request set WithChildrenCount. The bool is false when it is absent.
func (a DigitalTwinAsset) ChildrenCount() (int64, bool) {
	v := a.Get("assignedChildrenCount")
	return v.Int(), v.Exists()
}

// SubAssets yields the sub-assets included in the response (WithSubAssets).
func (a DigitalTwinAsset) SubAssets() iter.Seq[DigitalTwinAsset] {
	return seqOf(a.Get("subAssets"), NewDigitalTwinAsset)
}

// Parents yields the parent assets included in the response (WithParents).
func (a DigitalTwinAsset) Parents() iter.Seq[DigitalTwinAsset] {
	return seqOf(a.Get("assetParents"), NewDigitalTwinAsset)
}

// LinkedSeries yields the linked series included in the response
// (WithLinkedSeries).
func (a DigitalTwinAsset) LinkedSeries() iter.Seq[DigitalTwinLinkedSeries] {
	return seqOf(a.Get("c8y_LinkedSeries"), NewDigitalTwinLinkedSeries)
}

// LatestMeasurement returns the latest value of fragment.series from the
// c8y_LatestMeasurements fragment (WithLatestValues). The bool is false when
// no value is present.
func (a DigitalTwinAsset) LatestMeasurement(fragment, series string) (DigitalTwinMeasurementValue, bool) {
	v := a.Get("c8y_LatestMeasurements." + gjsonEscape(fragment) + "." + gjsonEscape(series))
	if !v.Exists() {
		return DigitalTwinMeasurementValue{}, false
	}
	return DigitalTwinMeasurementValue{
		Time:     v.Get("time").Time(),
		Value:    v.Get("value").Float(),
		Unit:     v.Get("unit").String(),
		SourceID: v.Get("source.id").String(),
	}, true
}

// DigitalTwinMeasurementValue is one entry of c8y_LatestMeasurements.
type DigitalTwinMeasurementValue struct {
	Time     time.Time
	Value    float64
	Unit     string
	SourceID string
}

// DigitalTwinLinkedSeries links an asset series to a device measurement
// series. Linked series are keyed by Fragment and Series.
type DigitalTwinLinkedSeries struct {
	jsondoc.Facade
}

func NewDigitalTwinLinkedSeries(b []byte) DigitalTwinLinkedSeries {
	return DigitalTwinLinkedSeries{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (s DigitalTwinLinkedSeries) Fragment() string { return s.Get("fragment").String() }
func (s DigitalTwinLinkedSeries) Series() string   { return s.Get("series").String() }
func (s DigitalTwinLinkedSeries) Label() string    { return s.Get("label").String() }

// HasSource reports whether the linked series points to a device series.
func (s DigitalTwinLinkedSeries) HasSource() bool { return s.Get("source.id").Exists() }

// Source returns the device series the linked series reads from.
func (s DigitalTwinLinkedSeries) Source() DigitalTwinSeriesSource {
	return NewDigitalTwinSeriesSource([]byte(s.Get("source").Raw))
}

// DigitalTwinSeriesSource is the device side of a linked series.
type DigitalTwinSeriesSource struct {
	jsondoc.Facade
}

func NewDigitalTwinSeriesSource(b []byte) DigitalTwinSeriesSource {
	return DigitalTwinSeriesSource{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// ID returns the id of the device (managed object) providing the series.
func (s DigitalTwinSeriesSource) ID() string       { return s.Get("id").String() }
func (s DigitalTwinSeriesSource) Name() string     { return s.Get("name").String() }
func (s DigitalTwinSeriesSource) Fragment() string { return s.Get("fragment").String() }
func (s DigitalTwinSeriesSource) Series() string   { return s.Get("series").String() }

// Type returns the measurement type of the source series, when known.
func (s DigitalTwinSeriesSource) Type() string { return s.Get("type").String() }

// DigitalTwinLinkedAsset is an asset series that reads from a device series,
// as returned when looking up the assets a device feeds.
type DigitalTwinLinkedAsset struct {
	jsondoc.Facade
}

func NewDigitalTwinLinkedAsset(b []byte) DigitalTwinLinkedAsset {
	return DigitalTwinLinkedAsset{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

// Fragment and Series name the device series.
func (l DigitalTwinLinkedAsset) Fragment() string { return l.Get("fragment").String() }
func (l DigitalTwinLinkedAsset) Series() string   { return l.Get("series").String() }

// Type is the measurement type of the device series.
func (l DigitalTwinLinkedAsset) Type() string { return l.Get("type").String() }
func (l DigitalTwinLinkedAsset) Name() string { return l.Get("name").String() }

// AssetID returns the id of the asset that links to the device series.
func (l DigitalTwinLinkedAsset) AssetID() string { return l.Get("asset.id").String() }

// AssetFragment and AssetSeries name the series on the asset side.
func (l DigitalTwinLinkedAsset) AssetFragment() string { return l.Get("asset.fragment").String() }
func (l DigitalTwinLinkedAsset) AssetSeries() string   { return l.Get("asset.series").String() }
func (l DigitalTwinLinkedAsset) AssetLabel() string    { return l.Get("asset.label").String() }

// DigitalTwinDefinition is a DTM definition: a property, asset, measurement,
// event or alarm definition. All kinds share identifier, jsonSchema and tags;
// the other accessors return zero values for kinds that do not have them.
type DigitalTwinDefinition struct {
	jsondoc.Facade
}

func NewDigitalTwinDefinition(b []byte) DigitalTwinDefinition {
	return DigitalTwinDefinition{jsondoc.Facade{JSONDoc: jsondoc.New(b)}}
}

func (d DigitalTwinDefinition) Identifier() string      { return d.Get("identifier").String() }
func (d DigitalTwinDefinition) Title() string           { return d.Get("jsonSchema.title").String() }
func (d DigitalTwinDefinition) Description() string     { return d.Get("jsonSchema.description").String() }
func (d DigitalTwinDefinition) Tags() []string          { return stringsOf(d.Get("tags")) }
func (d DigitalTwinDefinition) CreationTime() time.Time { return d.Get("creationTime").Time() }
func (d DigitalTwinDefinition) LastUpdated() time.Time  { return d.Get("lastUpdated").Time() }

// JSONSchema returns the stored JSON schema, including the fields the server
// adds ($schema, definitions, required, additionalProperties).
func (d DigitalTwinDefinition) JSONSchema() jsondoc.JSONDoc {
	return jsondoc.New([]byte(d.Get("jsonSchema").Raw))
}

// Contexts returns the domain entities a property definition applies to
// (for example "asset" or "measurement"). Empty for other kinds.
func (d DigitalTwinDefinition) Contexts() []string { return stringsOf(d.Get("contexts")) }

// Origin returns c8y_Origin, the component that owns a property definition.
// Empty for definitions created by users.
func (d DigitalTwinDefinition) Origin() string { return d.Get("c8y_Origin").String() }

// IsShared reports whether the definition is shared from the enterprise
// tenant (c8y_SharedDefinition). Shared definitions are read-only in
// subtenants.
func (d DigitalTwinDefinition) IsShared() bool { return d.Get("c8y_SharedDefinition").Exists() }

// CanEdit, CanEditMetadata and CanDelete read c8y_AvailableActions, which
// tells the current user what they may change.
func (d DigitalTwinDefinition) CanEdit() bool {
	return d.Get("c8y_AvailableActions.editAllowed").Bool()
}
func (d DigitalTwinDefinition) CanEditMetadata() bool {
	return d.Get("c8y_AvailableActions.editMetadataAllowed").Bool()
}
func (d DigitalTwinDefinition) CanDelete() bool {
	return d.Get("c8y_AvailableActions.deleteAllowed").Bool()
}

// AllowedProperties returns composition.allowedProperties (asset definitions).
func (d DigitalTwinDefinition) AllowedProperties() []DigitalTwinAllowedItem {
	return allowedItems(d.Get("composition.allowedProperties"))
}

// AllowedSubAssets returns composition.allowedSubAssets (asset definitions).
func (d DigitalTwinDefinition) AllowedSubAssets() []DigitalTwinAllowedItem {
	return allowedItems(d.Get("composition.allowedSubAssets"))
}

// AdditionalProperties and AdditionalSubAssets read the composition flags
// that allow properties or sub-assets beyond the allowed lists.
func (d DigitalTwinDefinition) AdditionalProperties() bool {
	return d.Get("composition.additionalProperties").Bool()
}
func (d DigitalTwinDefinition) AdditionalSubAssets() bool {
	return d.Get("composition.additionalSubAssets").Bool()
}

// DigitalTwinAllowedItem is an entry of an asset definition's composition.
// MinOccurs and MaxOccurs are strings in the API (for example "0", "1", "*").
type DigitalTwinAllowedItem struct {
	Identifier string
	MinOccurs  string
	MaxOccurs  string
}

func allowedItems(arr gjson.Result) []DigitalTwinAllowedItem {
	if !arr.IsArray() {
		return nil
	}
	items := arr.Array()
	out := make([]DigitalTwinAllowedItem, 0, len(items))
	for _, item := range items {
		out = append(out, DigitalTwinAllowedItem{
			Identifier: item.Get("identifier").String(),
			MinOccurs:  item.Get("minOccurs").String(),
			MaxOccurs:  item.Get("maxOccurs").String(),
		})
	}
	return out
}

// gjsonEscape escapes characters with a special meaning in gjson paths, so
// fragment and series names are matched literally.
func gjsonEscape(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.', '*', '?', '|', '#', '@', '\\', '!', '=', '<', '>', '%':
			out = append(out, '\\')
		}
		out = append(out, s[i])
	}
	return string(out)
}
