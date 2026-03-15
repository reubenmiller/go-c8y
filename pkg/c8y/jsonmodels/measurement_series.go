package jsonmodels

import (
	"encoding/json"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
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

func (s *Series) GetSeries() string {
	return s.Type + "." + s.Name
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
	Time       time.Time
	DeviceID   string        // ID of the source device
	DeviceName string        // Name of the source device (if resolved)
	Series     []Series      // Series metadata (order matches Values)
	Values     []SeriesValue // Values for each series (order matches Series)
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

	series := m.GetSeries()
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
			Time:       timestamp,
			DeviceID:   m.DeviceID,
			DeviceName: m.DeviceName,
			Series:     series,
			Values:     values,
		})

		return true // Continue iteration
	})

	// Sort rows by timestamp
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			if rows[i].Time.After(rows[j].Time) {
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

// FlatRow represents a single timestamp row where each element of Values corresponds
// to a column returned by ToFlatRows. A nil pointer means that stat was not present
// for that series at that timestamp.
type FlatRow struct {
	Time       time.Time
	DeviceID   string // ID of the source device
	DeviceName string // Name of the source device (if resolved)
	Values     []*float64
}

// ToFlatRows returns a column header slice and a flat row slice suitable for CSV or
// tabular export. Columns are named "<type>.<name>.<stat>" (e.g.
// "c8y_Temperature.T.avg"). Only stats that appear in the data produce columns —
// stats that were never requested will not appear. Count is widened to float64.
// Source device context is available on each row via DeviceID and DeviceName.
//
// Example:
//
//	columns, rows := m.ToFlatRows()
//	fmt.Println("time,source.id,source.name," + strings.Join(columns, ","))
//	for _, row := range rows {
//	    vals := make([]string, len(row.Values))
//	    for i, v := range row.Values {
//	        if v != nil { vals[i] = strconv.FormatFloat(*v, 'f', -1, 64) }
//	    }
//	    fmt.Printf("%s,%s,%s,%s\n", row.Time.Format(time.RFC3339), row.DeviceID, row.DeviceName, strings.Join(vals, ","))
//	}
func (m MeasurementSeries) ToFlatRows() (columns []string, rows []FlatRow) {
	tabular := m.ToTabular()
	if len(tabular) == 0 {
		return nil, nil
	}

	series := m.GetSeries()

	// Determine which stats are present by scanning all data so that columns
	// only appear when they carry actual data.
	type statPresence struct {
		min, max, avg, count, sum, stdDevPop, stdDevSamp bool
	}
	presence := make([]statPresence, len(series))
	for _, row := range tabular {
		for i, v := range row.Values {
			if i >= len(series) {
				break
			}
			p := &presence[i]
			p.min = p.min || v.Min != nil
			p.max = p.max || v.Max != nil
			p.avg = p.avg || v.Avg != nil
			p.count = p.count || v.Count != nil
			p.sum = p.sum || v.Sum != nil
			p.stdDevPop = p.stdDevPop || v.StdDevPop != nil
			p.stdDevSamp = p.stdDevSamp || v.StdDevSamp != nil
		}
	}

	// Build an ordered column definition list. Within each series the stat order
	// is: min, max, avg, count, sum, stdDevPop, stdDevSamp.
	type colDef struct {
		seriesIdx int
		stat      string
	}
	var defs []colDef
	for i, s := range series {
		p := presence[i]
		name := s.GetSeries()
		add := func(stat string, present bool) {
			if present {
				columns = append(columns, name+"."+stat)
				defs = append(defs, colDef{i, stat})
			}
		}
		add("min", p.min)
		add("max", p.max)
		add("avg", p.avg)
		add("count", p.count)
		add("sum", p.sum)
		add("stdDevPop", p.stdDevPop)
		add("stdDevSamp", p.stdDevSamp)
	}

	// Build flat rows.
	rows = make([]FlatRow, len(tabular))
	for ri, row := range tabular {
		flat := FlatRow{
			Time:       row.Time,
			DeviceID:   row.DeviceID,
			DeviceName: row.DeviceName,
			Values:     make([]*float64, len(defs)),
		}
		for ci, col := range defs {
			if col.seriesIdx >= len(row.Values) {
				continue
			}
			v := row.Values[col.seriesIdx]
			switch col.stat {
			case "min":
				flat.Values[ci] = v.Min
			case "max":
				flat.Values[ci] = v.Max
			case "avg":
				flat.Values[ci] = v.Avg
			case "count":
				if v.Count != nil {
					f := float64(*v.Count)
					flat.Values[ci] = &f
				}
			case "sum":
				flat.Values[ci] = v.Sum
			case "stdDevPop":
				flat.Values[ci] = v.StdDevPop
			case "stdDevSamp":
				flat.Values[ci] = v.StdDevSamp
			}
		}
		rows[ri] = flat
	}

	return columns, rows
}

// ToJSONRows returns one map per timestamp, suitable for JSON Lines encoding.
// Each map contains "timestamp" (time.Time) plus one key per stat that is
// present, named "<type>.<name>.<stat>" (e.g. "c8y_Temperature.T.avg").
// Stats with a nil value at a given timestamp are omitted from that map, so
// each line only carries the data that actually exists.
//
// Example:
//
//	enc := json.NewEncoder(os.Stdout)
//	for _, obj := range m.ToJSONRows() {
//	    enc.Encode(obj)
//	}
func (m MeasurementSeries) ToJSONRows() []map[string]any {
	tabular := m.ToTabular()
	if len(tabular) == 0 {
		return nil
	}

	result := make([]map[string]any, len(tabular))
	for ri, row := range tabular {
		obj := map[string]any{
			"time": row.Time,
			"source": map[string]string{
				"id": row.DeviceID,
				// "name": row.DeviceName,
			},
		}
		if row.DeviceName != "" {
			obj["source"].(map[string]string)["name"] = row.DeviceName
		}
		for i, v := range row.Values {
			if i >= len(row.Series) {
				break
			}
			prefix := row.Series[i].GetSeries()
			if v.Min != nil {
				obj[prefix+".min"] = *v.Min
			}
			if v.Max != nil {
				obj[prefix+".max"] = *v.Max
			}
			if v.Avg != nil {
				obj[prefix+".avg"] = *v.Avg
			}
			if v.Count != nil {
				obj[prefix+".count"] = *v.Count
			}
			if v.Sum != nil {
				obj[prefix+".sum"] = *v.Sum
			}
			if v.StdDevPop != nil {
				obj[prefix+".stdDevPop"] = *v.StdDevPop
			}
			if v.StdDevSamp != nil {
				obj[prefix+".stdDevSamp"] = *v.StdDevSamp
			}
		}
		result[ri] = obj
	}

	return result
}
