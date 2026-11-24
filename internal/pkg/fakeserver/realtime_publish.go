package fakeserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

// realtimeSources maps a REST collection to its realtime channel prefix and
// the JSON path of the device (or object) id that names the channel.
var realtimeSources = []struct {
	path    string
	channel string
	idPath  []string
}{
	{"/alarm/alarms", "/alarms/", []string{"source", "id"}},
	{"/event/events", "/events/", []string{"source", "id"}},
	{"/measurement/measurements", "/measurements/", []string{"source", "id"}},
	{"/devicecontrol/operations", "/operations/", []string{"deviceId"}},
	{"/inventory/managedObjects", "/managedobjects/", []string{"id"}},
}

// captureWriter keeps a copy of the response so it can be published.
type captureWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (c *captureWriter) WriteHeader(code int) {
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *captureWriter) Write(b []byte) (int, error) {
	if c.status == 0 {
		c.status = http.StatusOK
	}
	c.body.Write(b)
	return c.ResponseWriter.Write(b)
}

// withRealtimePublish serves r with next and, for a successful create or
// update of a realtime-enabled resource, pushes the resource to the realtime
// subscriptions that match its channel ("/events/<sourceId>", "/events/*"),
// like Cumulocity does. Deletes are not published.
func (fs *FakeServer) withRealtimePublish(w http.ResponseWriter, r *http.Request, next func(http.ResponseWriter, *http.Request)) {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		next(w, r)
		return
	}
	var src *struct {
		path    string
		channel string
		idPath  []string
	}
	for i := range realtimeSources {
		if strings.HasPrefix(r.URL.Path, realtimeSources[i].path) {
			src = &realtimeSources[i]
			break
		}
	}
	if src == nil || (src.channel == "/managedobjects/" && strings.Count(strings.TrimPrefix(r.URL.Path, src.path), "/") > 1) {
		next(w, r) // not a realtime resource (or an inventory sub-resource)
		return
	}
	cw := &captureWriter{ResponseWriter: w}
	next(cw, r)
	if cw.status < 200 || cw.status >= 300 {
		return
	}
	action := "CREATE"
	if r.Method == http.MethodPut {
		action = "UPDATE"
	}
	var docs []map[string]any
	var one map[string]any
	if err := json.Unmarshal(cw.body.Bytes(), &one); err == nil {
		if list, ok := one["measurements"].([]any); ok {
			for _, item := range list {
				if m, ok := item.(map[string]any); ok {
					docs = append(docs, m)
				}
			}
		} else {
			docs = append(docs, one)
		}
	}
	for _, doc := range docs {
		id := lookupString(doc, src.idPath)
		if id == "" {
			continue
		}
		channel := src.channel + id
		for _, conn := range fs.Realtime.Connections() {
			for _, pattern := range conn.Subscriptions() {
				if pattern == channel || pattern == src.channel+"*" {
					_ = conn.PushData(channel, action, doc)
					break
				}
			}
		}
	}
}

func lookupString(doc map[string]any, path []string) string {
	var cur any = doc
	for _, p := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = m[p]
	}
	s, _ := cur.(string)
	return s
}
