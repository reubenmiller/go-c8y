package core

import (
	"encoding/json"
	"testing"
)

// objectField extracts the JSON "object" multipart field built by
// NewMultiPartFileFields.
func objectField(t *testing.T, opt UploadFileOptions) map[string]any {
	t.Helper()
	fields := NewMultiPartFileFields(opt)
	var raw string
	found := false
	for _, f := range fields {
		if f.Name == "object" {
			if len(f.Values) > 0 {
				raw = f.Values[0]
			} else if f.Reader != nil {
				buf := make([]byte, 4096)
				n, _ := f.Reader.Read(buf)
				raw = string(buf[:n])
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected an object metadata field, got %+v", fields)
	}
	obj := map[string]any{}
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		t.Fatalf("unmarshal object field: %v", err)
	}
	return obj
}

func TestNewMultiPartFileFieldsProperties(t *testing.T) {
	// Custom properties are merged into the object metadata.
	obj := objectField(t, UploadFileOptions{
		FilePath: "report.pdf",
		Properties: map[string]any{
			"c8y_Global": map[string]any{},
			"name":       "from-properties",
		},
	})
	if _, ok := obj["c8y_Global"]; !ok {
		t.Errorf("custom fragment c8y_Global not merged: %v", obj)
	}
	// A "name" in properties is used when no explicit Name flag is set.
	if obj["name"] != "from-properties" {
		t.Errorf("name = %v, want from-properties", obj["name"])
	}

	// An explicit Name wins over a "name" in properties.
	obj = objectField(t, UploadFileOptions{
		FilePath:   "report.pdf",
		Name:       "explicit",
		Properties: map[string]any{"name": "from-properties"},
	})
	if obj["name"] != "explicit" {
		t.Errorf("name = %v, want explicit", obj["name"])
	}

	// No properties: behaves as before (name falls back to the file base name).
	obj = objectField(t, UploadFileOptions{FilePath: "dir/report.pdf"})
	if obj["name"] != "report.pdf" {
		t.Errorf("name = %v, want report.pdf", obj["name"])
	}
}
