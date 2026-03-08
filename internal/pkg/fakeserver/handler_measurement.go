package fakeserver

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleMeasurements routes /measurement/measurements requests.
func (fs *FakeServer) handleMeasurements(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// /measurement/measurements/series
	if strings.HasSuffix(path, "/series") || strings.Contains(path, "/series?") {
		fs.handleMeasurementSeries(w, r)
		return
	}

	// /measurement/measurements/{id}
	id := extractID(path, "/measurement/measurements")
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Measurements.Get(id)
			if !ok {
				writeNotFound(w, "measurement")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.Measurements.Delete(id) {
				writeNotFound(w, "measurement")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /measurement/measurements (collection)
	switch r.Method {
	case http.MethodGet:
		items := ReverseItems(FilterItems(r, fs.Measurements.List()))
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "measurements", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		body = enrichSource(body, fs.URL())
		_, doc := fs.Measurements.Create(body, fs.URL()+"/measurement/measurements")
		writeJSON(w, http.StatusCreated, doc)

	case http.MethodDelete:
		// Bulk delete measurements by filter
		items := FilterItems(r, fs.Measurements.List())
		for _, item := range items {
			id := getJSONString(item, "id")
			if id != "" {
				fs.Measurements.Delete(id)
			}
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleMeasurementSeries handles /measurement/measurements/series.
// Returns a simplified series structure.
func (fs *FakeServer) handleMeasurementSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}

	// Build a simplified series response from stored measurements
	items := FilterItems(r, fs.Measurements.List())

	// Collect unique series and timestamps
	type seriesInfo struct {
		Name string
		Type string
	}
	seriesSet := make(map[string]seriesInfo)
	var timestamps []string

	for _, item := range items {
		var m map[string]any
		if err := json.Unmarshal(item, &m); err != nil {
			continue
		}
		timeVal, _ := m["time"].(string)
		if timeVal != "" {
			timestamps = append(timestamps, timeVal)
		}
		// Find measurement fragment (any key that's a map with nested series data)
		for k, v := range m {
			if k == "id" || k == "self" || k == "source" || k == "type" || k == "time" ||
				k == "creationTime" || k == "lastUpdated" {
				continue
			}
			if fragment, ok := v.(map[string]any); ok {
				for seriesKey := range fragment {
					key := k + "." + seriesKey
					seriesSet[key] = seriesInfo{Name: seriesKey, Type: k}
				}
			}
		}
	}

	// Build series array
	var seriesArr []map[string]string
	for key, info := range seriesSet {
		seriesArr = append(seriesArr, map[string]string{
			"name": info.Name,
			"type": info.Type,
			"unit": key,
		})
	}
	if seriesArr == nil {
		seriesArr = []map[string]string{}
	}

	// Build values array (simplified - one entry per timestamp)
	values := make(map[string][]map[string]any)
	for key := range seriesSet {
		values[key] = []map[string]any{}
	}

	resp := map[string]any{
		"series":    seriesArr,
		"values":    values,
		"truncated": false,
	}

	out, _ := json.Marshal(resp)
	writeJSON(w, http.StatusOK, out)
}
