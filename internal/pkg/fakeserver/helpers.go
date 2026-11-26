package fakeserver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, body json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

// writeNoContent writes a 204 No Content response.
func writeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// platformNotFound maps a resource to the error type and message format the
// platform answers for a missing item (the format takes the requested id).
var platformNotFound = map[string]struct{ errorType, format string }{
	"inventory/managedObject": {"inventory/Not Found", "Finding device data from database failed : No managedObject for id '%s'!"},
	"alarm":                   {"alarm/Not Found", "Finding alarm data from database failed : No alarm for id '%s'!"},
	"event":                   {"event/Not Found", "Finding event data from database failed : No event for id '%s'!"},
	"measurement":             {"measurement/Not Found", "Finding measurement data from database failed : No measurement for id '%s'!"},
	"devicecontrol/operation": {"devicecontrol/Not Found", "Finding operation data from database failed : No operation for id '%s'!"},
}

// writeNotFound writes a Cumulocity 404 error response. For the resources in
// platformNotFound the error type and message follow the platform (e.g.
// inventory/Not Found, "... No managedObject for id '0'!") when an id is
// given; any other resource gets a generic not found error.
func writeNotFound(w http.ResponseWriter, resource string, id ...string) {
	if pn, ok := platformNotFound[resource]; ok && len(id) > 0 {
		writeError(w, http.StatusNotFound, pn.errorType, fmt.Sprintf(pn.format, id[0]))
		return
	}
	writeError(w, http.StatusNotFound, resource+"/notFound", "Not found")
}

// unroutedHeader marks the 404 of a request which no handler routes (as opposed
// to the 404 of a resource that does not exist). The coverage matrix test
// (coverage_test.go) uses it to find the endpoints the fake server lacks.
const unroutedHeader = "X-Fakeserver-Unrouted"

// writeUnrouted answers a request to an endpoint the handler does not
// implement: the same not found error as writeNotFound, plus unroutedHeader.
func writeUnrouted(w http.ResponseWriter, resource string) {
	w.Header().Set(unroutedHeader, "true")
	writeNotFound(w, resource)
}

// writeError writes a Cumulocity-style error response.
func writeError(w http.ResponseWriter, status int, errorType, message string) {
	body, _ := json.Marshal(map[string]string{
		"error":   errorType,
		"message": message,
		"info":    "https://cumulocity.com/api/core/",
	})
	writeJSON(w, status, body)
}

// readBody reads the full request body as json.RawMessage. Fractional and
// exponent numbers of a valid JSON body are normalised like the platform does
// (see normalizeJSONNumbers); any other body is returned unchanged.
func readBody(r *http.Request) (json.RawMessage, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return json.RawMessage(`{}`), nil
	}
	if !json.Valid(b) {
		return json.RawMessage(b), nil
	}
	return json.RawMessage(normalizeJSONNumbers(b)), nil
}

// extractID extracts the resource ID from a URL path.
// Given a prefix like "/alarm/alarms/", it returns the next path segment.
func extractID(path, prefix string) string {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimPrefix(rest, "/")
	if idx := strings.Index(rest, "/"); idx != -1 {
		return rest[:idx]
	}
	return rest
}

// extractPathSegments returns all path segments after the prefix.
// e.g. for path="/inventory/managedObjects/123/childDevices" and prefix="/inventory/managedObjects/"
// returns ["123", "childDevices"]
func extractPathSegments(path, prefix string) []string {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return nil
	}
	return strings.Split(rest, "/")
}

// getJSONString extracts a top-level string field from raw JSON.
func getJSONString(doc json.RawMessage, key string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(doc, &m); err != nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	var s string
	if err := json.Unmarshal(v, &s); err != nil {
		// Might be a number
		return strings.Trim(string(v), `"`)
	}
	return s
}

// buildSourceSelfURL builds a self URL for a source object reference.
func buildSourceSelfURL(baseURL, sourceID string) string {
	return fmt.Sprintf("%s/inventory/managedObjects/%s", baseURL, sourceID)
}

// enrichSource ensures source objects have "self" links populated.
func enrichSource(doc json.RawMessage, baseURL string) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(doc, &m); err != nil {
		return doc
	}
	srcRaw, ok := m["source"]
	if !ok {
		return doc
	}
	var src map[string]json.RawMessage
	if err := json.Unmarshal(srcRaw, &src); err != nil {
		return doc
	}
	if idRaw, ok := src["id"]; ok {
		id := strings.Trim(string(idRaw), `"`)
		src["self"], _ = json.Marshal(buildSourceSelfURL(baseURL, id))
		srcJSON, _ := json.Marshal(src)
		m["source"] = srcJSON
		out, _ := json.Marshal(m)
		return out
	}
	return doc
}

// writeCount writes a plain text count response (for /count endpoints).
func writeCount(w http.ResponseWriter, count int) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(strconv.Itoa(count)))
}
