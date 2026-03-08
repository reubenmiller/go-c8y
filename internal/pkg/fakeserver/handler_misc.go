package fakeserver

import (
	"encoding/json"
	"net/http"
)

// handleAuditRecords routes /audit/auditRecords requests.
func (fs *FakeServer) handleAuditRecords(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/audit/auditRecords")

	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.AuditRecords.Get(id)
			if !ok {
				writeNotFound(w, "auditRecord")
				return
			}
			writeJSON(w, http.StatusOK, doc)
		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		items := ReverseItems(FilterItems(r, fs.AuditRecords.List()))
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "auditRecords", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		_, doc := fs.AuditRecords.Create(body, fs.URL()+"/audit/auditRecords")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleRetentionRules routes /retention/retentions requests.
func (fs *FakeServer) handleRetentionRules(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/retention/retentions")

	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.RetentionRules.Get(id)
			if !ok {
				writeNotFound(w, "retentionRule")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.RetentionRules.Update(id, body)
			if !ok {
				writeNotFound(w, "retentionRule")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.RetentionRules.Delete(id) {
				writeNotFound(w, "retentionRule")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		items := fs.RetentionRules.List()
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "retentionRules", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		_, doc := fs.RetentionRules.Create(body, fs.URL()+"/retention/retentions")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleFeatures routes /features requests.
func (fs *FakeServer) handleFeatures(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/features")

	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Features.Get(id)
			if !ok {
				writeNotFound(w, "feature")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.Features.Update(id, body)
			if !ok {
				writeNotFound(w, "feature")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.Features.Delete(id) {
				writeNotFound(w, "feature")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		items := fs.Features.List()
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "features", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		_, doc := fs.Features.Create(body, fs.URL()+"/features")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleNotification2 routes /notification2/ requests.
func (fs *FakeServer) handleNotification2(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/notification2/subscriptions")

	// /notification2/token
	if extractID(r.URL.Path, "/notification2/token") != "" || r.URL.Path == "/notification2/token" {
		switch r.Method {
		case http.MethodPost:
			resp := marshalJSON(map[string]any{
				"token": "fake-notification2-token",
			})
			writeJSON(w, http.StatusOK, resp)
		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /notification2/unsubscribe
	if r.URL.Path == "/notification2/unsubscribe" {
		if r.Method == http.MethodPost {
			writeJSON(w, http.StatusOK, marshalJSON(map[string]any{
				"result": "DELETED",
			}))
			return
		}
	}

	// /notification2/subscriptions
	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.Notification2Subscriptions.Get(id)
			if !ok {
				writeNotFound(w, "subscription")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.Notification2Subscriptions.Delete(id) {
				writeNotFound(w, "subscription")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		items := fs.Notification2Subscriptions.List()
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "subscriptions", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		_, doc := fs.Notification2Subscriptions.Create(body, fs.URL()+"/notification2/subscriptions")
		writeJSON(w, http.StatusCreated, doc)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleDeviceRequests routes /devicecontrol/newDeviceRequests.
func (fs *FakeServer) handleDeviceRequests(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "/devicecontrol/newDeviceRequests")

	if id != "" {
		switch r.Method {
		case http.MethodGet:
			doc, ok := fs.DeviceRequests.Get(id)
			if !ok {
				writeNotFound(w, "newDeviceRequest")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodPut:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			doc, ok := fs.DeviceRequests.Update(id, body)
			if !ok {
				writeNotFound(w, "newDeviceRequest")
				return
			}
			writeJSON(w, http.StatusOK, doc)

		case http.MethodDelete:
			if !fs.DeviceRequests.Delete(id) {
				writeNotFound(w, "newDeviceRequest")
				return
			}
			writeNoContent(w)

		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	switch r.Method {
	case http.MethodGet:
		items := fs.DeviceRequests.List()
		page := Paginate(r, items)
		writeJSON(w, http.StatusOK, BuildCollectionResponse(r, fs.URL(), "newDeviceRequests", page))

	case http.MethodPost:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		reqID := getJSONString(body, "id")
		if reqID == "" {
			reqID = NextID()
		}
		body = mergeFields(body, map[string]any{
			"id":     reqID,
			"self":   fs.URL() + "/devicecontrol/newDeviceRequests/" + reqID,
			"status": "WAITING_FOR_CONNECTION",
		})
		fs.DeviceRequests.CreateWithID(reqID, body)
		writeJSON(w, http.StatusCreated, body)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// handleDeviceCredentials handles /devicecontrol/deviceCredentials.
func (fs *FakeServer) handleDeviceCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		body, _ := readBody(r)
		deviceID := getJSONString(body, "id")
		resp := marshalJSON(map[string]any{
			"id":       deviceID,
			"tenantId": "t12345",
			"username": "device_" + deviceID,
			"password": "***",
			"self":     fs.URL() + "/devicecontrol/deviceCredentials",
		})
		writeJSON(w, http.StatusCreated, resp)

	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}

// marshalJSON is defined in server.go — the following is for json.RawMessage convenience.
// This file uses the package-level marshalJSON from server.go.
var _ json.RawMessage // ensure import is used

// handleBulkNewDeviceRequests handles POST /devicecontrol/bulkNewDeviceRequests (file upload).
func (fs *FakeServer) handleBulkNewDeviceRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		return
	}

	// Count CSV lines (rows) in the uploaded file to simulate per-line responses.
	// The upload is typically multipart with a CSV body.
	// CSV has a header row + data rows. Each row ends with \n.
	// Total rows = count of \n characters. Data rows = total rows - 1 (header).
	lines := 0
	if err := r.ParseMultipartForm(32 << 20); err == nil {
		f, _, ferr := r.FormFile("file")
		if ferr == nil {
			buf := make([]byte, 0, 4096)
			tmp := make([]byte, 1024)
			for {
				n, rerr := f.Read(tmp)
				buf = append(buf, tmp[:n]...)
				if rerr != nil {
					break
				}
			}
			f.Close()
			for _, b := range buf {
				if b == '\n' {
					lines++
				}
			}
			// Subtract header row
			if lines > 0 {
				lines--
			}
		}
	}
	if lines == 0 {
		lines = 1
	}

	resp := marshalJSON(map[string]any{
		"numberOfAll":        lines,
		"numberOfCreated":    lines,
		"numberOfFailed":     0,
		"numberOfSuccessful": lines,
		"status":             "CREATED",
	})
	writeJSON(w, http.StatusCreated, resp)
}

// handleRemoteAccess routes /service/remoteaccess/ requests.
func (fs *FakeServer) handleRemoteAccess(w http.ResponseWriter, r *http.Request) {
	// /service/remoteaccess/devices/{moId}/configurations[/{id}]
	segments := extractPathSegments(r.URL.Path, "/service/remoteaccess/devices")
	if len(segments) < 2 {
		writeNotFound(w, "remoteaccess")
		return
	}
	// segments: [moId, "configurations", ...rest]

	if len(segments) == 2 {
		// /service/remoteaccess/devices/{moId}/configurations
		switch r.Method {
		case http.MethodGet:
			// Return all configs for this device (just return all — simple)
			items := fs.RemoteAccessConfigs.List()
			writeJSON(w, http.StatusOK, marshalJSON(items))
		case http.MethodPost:
			body, err := readBody(r)
			if err != nil {
				writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
				return
			}
			_, doc := fs.RemoteAccessConfigs.Create(body, fs.URL()+r.URL.Path)
			writeJSON(w, http.StatusCreated, doc)
		default:
			writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
		}
		return
	}

	// /service/remoteaccess/devices/{moId}/configurations/{id}
	configID := segments[2]
	switch r.Method {
	case http.MethodGet:
		doc, ok := fs.RemoteAccessConfigs.Get(configID)
		if !ok {
			writeNotFound(w, "remoteaccess/configuration")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	case http.MethodPut:
		body, err := readBody(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "general/badRequest", err.Error())
			return
		}
		doc, ok := fs.RemoteAccessConfigs.Update(configID, body)
		if !ok {
			writeNotFound(w, "remoteaccess/configuration")
			return
		}
		writeJSON(w, http.StatusOK, doc)
	case http.MethodDelete:
		if !fs.RemoteAccessConfigs.Delete(configID) {
			writeNotFound(w, "remoteaccess/configuration")
			return
		}
		writeNoContent(w)
	default:
		writeError(w, http.StatusMethodNotAllowed, "general/methodNotAllowed", "Method not allowed")
	}
}
