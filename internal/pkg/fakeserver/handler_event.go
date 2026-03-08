package fakeserver

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// handleEvents routes /event/events requests.
func (fs *FakeServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check for binary sub-resource: /event/events/{id}/binaries
	segments := extractPathSegments(path, "/event/events")
	if len(segments) >= 2 && segments[1] == "binaries" {
		fs.handleEventBinaries(w, r, segments[0])
		return
	}

	// /event/events/{id}
	id := extractID(path, "/event/events")
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Events.Get(id)
			if !ok {
				writeNotFound(w, "event")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.Events.Update(id, body)
			if !ok {
				writeNotFound(w, "event")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.Events.Delete(id) {
				writeNotFound(w, "event")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /event/events (collection)
	switch r.Method {
	case http.MethodGet:
		items := ReverseItems(FilterItems(r, fs.Events.List()))
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "events", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		// Validate source reference
		sourceID := getJSONString(body, "source.id")
		if sourceID == "" {
			// Try nested struct
			var payload struct {
				Source struct {
					ID string `json:"id"`
				} `json:"source"`
			}
			json.Unmarshal(body, &payload)
			sourceID = payload.Source.ID
		}
		if sourceID != "" {
			if _, ok := fs.ManagedObjects.Get(sourceID); !ok {
				writeError(w, http.StatusUnprocessableEntity, "event/Unprocessable Entity", "Source not found: "+sourceID)
				return
			}
		}
		body = enrichSource(body, fs.URL())
		_, doc := fs.Events.Create(body, fs.URL()+"/event/events")
		writeJSON(w, http.StatusCreated, doc)

	case http.MethodDelete:
		// Delete matching events
		items := FilterItems(r, fs.Events.List())
		for _, item := range items {
			id := getJSONString(item, "id")
			if id != "" {
				fs.Events.Delete(id)
			}
		}
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleEventBinaries handles /event/events/{id}/binaries
func (fs *FakeServer) handleEventBinaries(w http.ResponseWriter, r *http.Request, eventID string) {
	if _, ok := fs.Events.Get(eventID); !ok {
		writeNotFound(w, "event")
		return
	}

	switch r.Method {
	case http.MethodPost:
		// Read multipart or raw binary
		var data []byte
		var filename string

		ct := r.Header.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/") {
			if err := r.ParseMultipartForm(32 << 20); err == nil {
				for _, fhs := range r.MultipartForm.File {
					if len(fhs) > 0 {
						filename = fhs[0].Filename
						f, err := fhs[0].Open()
						if err == nil {
							data, _ = io.ReadAll(f)
							f.Close()
						}
						break
					}
				}
			}
		}
		if data == nil {
			var err error
			data, err = io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
		}
		if filename == "" {
			filename = "binary-" + eventID
		}

		fs.ManagedObjects.mu.Lock()
		fs.EventBinaries[eventID] = data
		fs.ManagedObjects.mu.Unlock()

		// Return the event with binary info
		doc, _ := fs.Events.Get(eventID)
		respCT := ct
		if respCT == "" {
			respCT = "application/octet-stream"
		}
		doc = mergeFields(doc, map[string]any{
			"c8y_IsBinary": map[string]any{},
			"name":         filename,
			"contentType":  respCT,
			"length":       len(data),
		})
		fs.Events.Update(eventID, doc)
		writeJSON(w, http.StatusCreated, doc)

	case http.MethodGet:
		fs.ManagedObjects.mu.RLock()
		data, ok := fs.EventBinaries[eventID]
		fs.ManagedObjects.mu.RUnlock()
		if !ok {
			writeNotFound(w, "event/binary")
			return
		}
		contentType := "application/octet-stream"
		if doc, ok := fs.Events.Get(eventID); ok {
			if ct := getJSONString(doc, "contentType"); ct != "" {
				contentType = ct
			}
		}
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)

	case http.MethodDelete:
		fs.ManagedObjects.mu.Lock()
		delete(fs.EventBinaries, eventID)
		fs.ManagedObjects.mu.Unlock()
		writeNoContent(w)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// isEventBinaryPath checks if the path includes a binary sub-resource.
func isEventBinaryPath(path string) bool {
	return strings.Contains(path, "/binaries")
}
