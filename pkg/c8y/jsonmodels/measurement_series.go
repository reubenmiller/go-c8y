package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// MeasurementSeries represents the raw API response for measurement series
// The API returns data in a column-based format
type MeasurementSeries struct {
	jsondoc.Facade
	DeviceID   string // ID of the source device
	DeviceName string // Name of the source device (if resolved)
}

func NewMeasurementSeries(b []byte) MeasurementSeries {
	return MeasurementSeries{
		Facade:     jsondoc.Facade{JSONDoc: jsondoc.New(b)},
		DeviceID:   "",
		DeviceName: "",
	}
}

// WithDeviceInfo creates a MeasurementSeries with device metadata
func NewMeasurementSeriesWithDevice(b []byte, deviceID, deviceName string) MeasurementSeries {
	return MeasurementSeries{
		Facade:     jsondoc.Facade{JSONDoc: jsondoc.New(b)},
		DeviceID:   deviceID,
		DeviceName: deviceName,
	}
}

// Series represents a single measurement series with its values
type Series struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Unit string `json:"unit"`
}

// SeriesValue represents aggregated values at a specific timestamp
// Uses pointers to distinguish between zero values and unset fields
type SeriesValue struct {
	Min        *float64 `json:"min,omitempty"`
	Max        *float64 `json:"max,omitempty"`
	Count      *int     `json:"count,omitempty"`
	Sum        *float64 `json:"sum,omitempty"`
	Avg        *float64 `json:"avg,omitempty"`
	StdDevPop  *float64 `json:"stdDevPop,omitempty"`
	StdDevSamp *float64 `json:"stdDevSamp,omitempty"`
}

// GetMin returns the min value or 0 if not set
func (s SeriesValue) GetMin() float64 {
	if s.Min != nil {
		return *s.Min
	}
	return 0
}

// GetMax returns the max value or 0 if not set
func (s SeriesValue) GetMax() float64 {
	if s.Max != nil {
		return *s.Max
	}
	return 0
}

// GetCount returns the count value or 0 if not set
func (s SeriesValue) GetCount() int {
	if s.Count != nil {
		return *s.Count
	}
	return 0
}

// GetSum returns the sum value or 0 if not set
func (s SeriesValue) GetSum() float64 {
	if s.Sum != nil {
		return *s.Sum
	}
	return 0
}

// GetAvg returns the average value or 0 if not set
func (s SeriesValue) GetAvg() float64 {
	if s.Avg != nil {
		return *s.Avg
	}
	return 0
}

// GetStdDevPop returns the population standard deviation or 0 if not set
func (s SeriesValue) GetStdDevPop() float64 {
	if s.StdDevPop != nil {
		return *s.StdDevPop
	}
	return 0
}

// GetStdDevSamp returns the sample standard deviation or 0 if not set
func (s SeriesValue) GetStdDevSamp() float64 {
	if s.StdDevSamp != nil {
		return *s.StdDevSamp
	}
	return 0
}

// HasMin returns true if min value is set
func (s SeriesValue) HasMin() bool {
	return s.Min != nil
}

// HasMax returns true if max value is set
func (s SeriesValue) HasMax() bool {
	return s.Max != nil
}

// HasCount returns true if count value is set
func (s SeriesValue) HasCount() bool {
	return s.Count != nil
}

// HasSum returns true if sum value is set
func (s SeriesValue) HasSum() bool {
	return s.Sum != nil
}

// HasAvg returns true if average value is set
func (s SeriesValue) HasAvg() bool {
	return s.Avg != nil
}

// HasStdDevPop returns true if population standard deviation is set
func (s SeriesValue) HasStdDevPop() bool {
	return s.StdDevPop != nil
}

// HasStdDevSamp returns true if sample standard deviation is set
func (s SeriesValue) HasStdDevSamp() bool {
	return s.StdDevSamp != nil
}

// SeriesRow represents a row in the tabular format
// Each row contains a timestamp and aggregated values for all series at that timestamp
type SeriesRow struct {
	Timestamp time.Time
	Values    []SeriesValue // Values for each series (order matches series array)
}

// GetSeries returns all series from the response
func (m MeasurementSeries) GetSeries() []Series {
	seriesArray := m.Get("series").Array()
	series := make([]Series, 0, len(seriesArray))

	for _, s := range seriesArray {
		var ser Series
		if err := json.Unmarshal([]byte(s.Raw), &ser); err == nil {
			series = append(series, ser)
		}
	}

	return series
}

// IsTruncated returns whether the result set was truncated
func (m MeasurementSeries) IsTruncated() bool {
	return m.Get("truncated").Bool()
}

// ToTabular transforms the column-based series data into a row-based tabular format
// This makes it easier to work with the data and convert to CSV or other formats
func (m MeasurementSeries) ToTabular() []SeriesRow {
	// Get the values map: timestamp -> array of aggregation objects
	valuesMap := m.Get("values")
	if !valuesMap.Exists() {
		return []SeriesRow{}
	}

	rows := make([]SeriesRow, 0)

	// Iterate over each timestamp in the values map
	valuesMap.ForEach(func(timestampStr, seriesValues gjson.Result) bool {
		// Parse the timestamp
		timestamp, err := time.Parse(time.RFC3339, timestampStr.String())
		if err != nil {
			return true // Continue iteration
		}

		// Parse the array of series values for this timestamp
		valueArray := seriesValues.Array()
		values := make([]SeriesValue, 0, len(valueArray))

		for _, v := range valueArray {
			var seriesValue SeriesValue
			if err := json.Unmarshal([]byte(v.Raw), &seriesValue); err == nil {
				values = append(values, seriesValue)
			}
		}

		rows = append(rows, SeriesRow{
			Timestamp: timestamp,
			Values:    values,
		})

		return true // Continue iteration
	})

	// Sort rows by timestamp
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[i].Timestamp.After(rows[j].Timestamp) {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}

	return rows
}

// GetSeriesNames returns the names of all series in the response
func (m MeasurementSeries) GetSeriesNames() []string {
	series := m.GetSeries()
	names := make([]string, 0, len(series))
	for _, s := range series {
		names = append(names, s.Name)
	}
	return names
}

// GetValuesForSeries returns all aggregated values for a specific series by index
// The index corresponds to the position in the series array
func (m MeasurementSeries) GetValuesForSeries(seriesIndex int) map[time.Time]SeriesValue {
	result := make(map[time.Time]SeriesValue)

	valuesMap := m.Get("values")
	if !valuesMap.Exists() {
		return result
	}

	// Iterate over each timestamp
	valuesMap.ForEach(func(timestampStr, seriesValues gjson.Result) bool {
		timestamp, err := time.Parse(time.RFC3339, timestampStr.String())
		if err != nil {
			return true
		}

		valueArray := seriesValues.Array()
		if seriesIndex < len(valueArray) {
			var seriesValue SeriesValue
			if err := json.Unmarshal([]byte(valueArray[seriesIndex].Raw), &seriesValue); err == nil {
				result[timestamp] = seriesValue
			}
		}

		return true
	})

	return result
}

// GetValuesForSeriesByName returns all aggregated values for a specific series by name
func (m MeasurementSeries) GetValuesForSeriesByName(seriesName string) map[time.Time]SeriesValue {
	series := m.GetSeries()

	// Find the index of the series
	for i, s := range series {
		if s.Name == seriesName {
			return m.GetValuesForSeries(i)
		}
	}

	return make(map[time.Time]SeriesValue)
}
