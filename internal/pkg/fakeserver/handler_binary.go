package fakeserver

import (
	"io"
	"net/http"
	"strings"
)

// handleBinaries routes /inventory/binaries requests.
func (fs *FakeServer) handleBinaries(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	id := extractID(path, "/inventory/binaries")
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Binaries.Get(id)
			if !ok {
				writeNotFound(w, "inventory/binary")
				return
			}
			// Check if we have actual binary data
			fs.ManagedObjects.mu.RLock()
			data, hasData := fs.BinaryData[id]
			fs.ManagedObjects.mu.RUnlock()

			if hasData {
				contentType := "application/octet-stream"
				if ct := getJSONString(doc, "contentType"); ct != "" {
					contentType = ct
				}
				w.Header().Set("Content-Type", contentType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			} else {
				writeJSON(w, http.StatusOK, doc)
			}

		case http.MethodPut:
			// Replace binary content
			data, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			fs.ManagedObjects.mu.Lock()
			fs.BinaryData[id] = data
			fs.ManagedObjects.mu.Unlock()

			doc, ok := fs.Binaries.Get(id)
			if !ok {
				writeNotFound(w, "inventory/binary")
				return
			}
			doc = mergeFields(doc, map[string]any{"length": len(data)})
			fs.Binaries.Update(id, doc)
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			fs.ManagedObjects.mu.Lock()
			delete(fs.BinaryData, id)
			fs.ManagedObjects.mu.Unlock()

			if !fs.Binaries.Delete(id) {
				writeNotFound(w, "inventory/binary")
				return
			}
			// Also remove from ManagedObjects so GET /inventory/managedObjects/{id} returns 404
			fs.ManagedObjects.Delete(id)
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /inventory/binaries (collection)
	switch r.Method {
	case http.MethodGet:
		items := FilterItems(r, fs.Binaries.List())
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "managedObjects", page))

	case http.MethodPost:
		// Multipart file upload
		ct := r.Header.Get("Content-Type")
		var data []byte
		var filename, fileContentType string

		if strings.HasPrefix(ct, "multipart/") {
			err := r.ParseMultipartForm(32 << 20)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			file, header, err := r.FormFile("file")
			if err != nil {
				// Try reading "object" field for metadata, and raw body for single file
				writeError(w, http.StatusBadRequest, "general/badRequest", "Missing file in multipart form")
				return
			}
			defer file.Close()
			data, _ = io.ReadAll(file)
			filename = header.Filename
			fileContentType = header.Header.Get("Content-Type")
			if fileContentType == "" {
				fileContentType = "application/octet-stream"
			}
		} else {
			var err error
			data, err = io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			filename = "upload"
			fileContentType = ct
		}

		binaryMO := marshalJSON(map[string]any{
			"name":         filename,
			"type":         fileContentType,
			"contentType":  fileContentType,
			"length":       len(data),
			"c8y_IsBinary": map[string]any{},
		})

		id, doc := fs.Binaries.Create(binaryMO, fs.URL()+"/inventory/binaries")

		// Also store in ManagedObjects so GET /inventory/managedObjects/{id} works
		moDoc := mergeFields(doc, map[string]any{
			"self": fs.URL() + "/inventory/managedObjects/" + id,
		})
		fs.ManagedObjects.CreateWithID(id, moDoc)

		fs.ManagedObjects.mu.Lock()
		fs.BinaryData[id] = data
		fs.ManagedObjects.mu.Unlock()

		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}
