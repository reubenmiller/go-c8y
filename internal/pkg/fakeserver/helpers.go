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

// writeNotFound writes a standard Cumulocity 404 error response.
func writeNotFound(w http.ResponseWriter, resource string) {
	writeError(w, http.StatusNotFound, resource+"/notFound", "Not found")
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

// readBody reads the full request body as json.RawMessage.
func readBody(r *http.Request) (json.RawMessage, error) {
	defer r.Body.Close()
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return json.RawMessage(`{}`), nil
	}
	return json.RawMessage(b), nil
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

// getJSONNumber extracts a top-level number field from raw JSON as a string.
func getJSONNumber(doc json.RawMessage, key string) string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(doc, &m); err != nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	return strings.Trim(string(v), `"`)
}

// buildSelfURL builds a "self" URL for a given resource path.
func buildSelfURL(baseURL, path string) string {
	return baseURL + path
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
