package httpcore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
)

// objectField extracts the JSON "object" multipart field built by
// NewMultiPartFileFields.
func objectField(t *testing.T, opt core.UploadFileOptions) map[string]any {
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
	obj := objectField(t, core.UploadFileOptions{
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
	obj = objectField(t, core.UploadFileOptions{
		FilePath:   "report.pdf",
		Name:       "explicit",
		Properties: map[string]any{"name": "from-properties"},
	})
	if obj["name"] != "explicit" {
		t.Errorf("name = %v, want explicit", obj["name"])
	}

	// No properties: behaves as before (name falls back to the file base name).
	obj = objectField(t, core.UploadFileOptions{FilePath: "dir/report.pdf"})
	if obj["name"] != "report.pdf" {
		t.Errorf("name = %v, want report.pdf", obj["name"])
	}
}

// TestNewMultiPartFileFieldsObjectType verifies the object "type" follows go-c8y-cli
// v2.59.0: the type of the source file's extension, application/octet-stream when
// the file has none (its content is not sniffed), and an explicit type wins.
func TestNewMultiPartFileFieldsObjectType(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	text := write("nofile", "hello text\n")
	unknown := write("c.foo", "hello text\n")
	txt := write("a.txt", "hello text\n")
	jsonFile := write("b.json", `{"a":1}`)

	tests := []struct {
		name string
		opt  core.UploadFileOptions
		want string
	}{
		{"no extension is not sniffed", core.UploadFileOptions{FilePath: text}, "application/octet-stream"},
		{"unknown extension is not sniffed", core.UploadFileOptions{FilePath: unknown}, "application/octet-stream"},
		{"known extension", core.UploadFileOptions{FilePath: txt}, "text/plain; charset=utf-8"},
		{"the upload name does not change the type", core.UploadFileOptions{FilePath: txt, Name: "x.json"}, "text/plain; charset=utf-8"},
		{"the extension of the file wins over a name without one", core.UploadFileOptions{FilePath: jsonFile, Name: "noext"}, "application/json"},
		{"a name does not type a file without extension", core.UploadFileOptions{FilePath: text, Name: "x.json"}, "application/octet-stream"},
		{"a reader is typed by the extension of its name", core.UploadFileOptions{Reader: strings.NewReader("x"), Name: "x.json"}, "application/json"},
		{"a reader without an extension", core.UploadFileOptions{Reader: strings.NewReader("x"), Name: "x"}, "application/octet-stream"},
		{"explicit type", core.UploadFileOptions{FilePath: txt, ContentType: "my/type"}, "my/type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := objectField(t, tt.opt)
			if obj["type"] != tt.want {
				t.Errorf("type = %v, want %v", obj["type"], tt.want)
			}
		})
	}
}
