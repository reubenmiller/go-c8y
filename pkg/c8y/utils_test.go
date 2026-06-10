package c8y

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type partInfo struct {
	ContentType string
	Filename    string
	Body        string
}

// readParts parses the multipart request body and returns the content type and contents of each part by name
func readParts(t *testing.T, req *http.Request) map[string]*partInfo {
	t.Helper()
	_, params, err := mime.ParseMediaType(req.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("ParseMediaType: %v", err)
	}
	parts := map[string]*partInfo{}
	mr := multipart.NewReader(req.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("NextPart: %v", err)
		}
		body, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("ReadAll part: %v", err)
		}
		parts[part.FormName()] = &partInfo{
			ContentType: part.Header.Get("Content-Type"),
			Filename:    part.FileName(),
			Body:        string(body),
		}
	}
	return parts
}

func createTestFile(t *testing.T, name string, contents string) *os.File {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}

func TestPrepareMultipartRequest(t *testing.T) {
	textContents := "Jan 01 00:00:00 host service: started\n"

	testCases := []struct {
		name     string
		filename string
		contents string
		// additional form values, e.g. filename, contentType and object
		fields map[string]string
		// expected file part metadata (the body must always equal contents)
		wantType     string
		wantFilename string
		// expected plain form fields by name and body
		wantFields map[string]string
	}{
		{
			name:         "content type is detected from the file extension",
			filename:     "syslog.txt",
			contents:     textContents,
			wantType:     "text/plain",
			wantFilename: "syslog.txt",
		},
		{
			name:         "content type is sniffed from the contents when the extension is unknown",
			filename:     "syslog.log",
			contents:     textContents,
			wantType:     "text/plain",
			wantFilename: "syslog.log",
		},
		{
			name:         "binary contents with an unknown extension fall back to application/octet-stream",
			filename:     "firmware.bin",
			contents:     string([]byte{0x00, 0x01, 0x02, 0x03, 0xff, 0xfe}),
			wantType:     "application/octet-stream",
			wantFilename: "firmware.bin",
		},
		{
			name:         "manual filename overrides the file's name and drives the content type detection",
			filename:     "syslog.log",
			contents:     "some plain text\n",
			fields:       map[string]string{"filename": "device-syslog-2026-06-10.txt"},
			wantType:     "text/plain",
			wantFilename: "device-syslog-2026-06-10.txt",
		},
		{
			name:         "manual content type takes precedence over detection",
			filename:     "settings.txt",
			contents:     "key=value\n",
			fields:       map[string]string{"contentType": "application/x-config"},
			wantType:     "application/x-config",
			wantFilename: "settings.txt",
		},
		{
			name:         "object is sent as a plain form field without a filename",
			filename:     "notes.txt",
			contents:     "hello\n",
			fields:       map[string]string{"object": `{"name":"notes.txt","type":"text/plain"}`},
			wantType:     "text/plain",
			wantFilename: "notes.txt",
			wantFields:   map[string]string{"object": `{"name":"notes.txt","type":"text/plain"}`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			values := map[string]io.Reader{
				"file": createTestFile(t, tc.filename, tc.contents),
			}
			for key, value := range tc.fields {
				values[key] = strings.NewReader(value)
			}

			req, err := prepareMultipartRequest("POST", "https://example.com/event/events/1/binaries", values)
			if err != nil {
				t.Fatalf("prepareMultipartRequest: %v", err)
			}

			parts := readParts(t, req)
			filePart := parts["file"]
			if filePart == nil {
				t.Fatal("missing file part")
			}
			if filePart.ContentType != tc.wantType {
				t.Errorf("file part content type: got %q, want %q", filePart.ContentType, tc.wantType)
			}
			if filePart.Filename != tc.wantFilename {
				t.Errorf("file part filename: got %q, want %q", filePart.Filename, tc.wantFilename)
			}
			if filePart.Body != tc.contents {
				t.Errorf("file part body: got %q, want %q", filePart.Body, tc.contents)
			}

			for key, want := range tc.wantFields {
				part := parts[key]
				if part == nil {
					t.Fatalf("missing %q part", key)
				}
				if part.Filename != "" {
					t.Errorf("%q part should not have a filename, got %q", key, part.Filename)
				}
				if part.Body != want {
					t.Errorf("%q part body: got %q, want %q", key, part.Body, want)
				}
			}

			// metadata fields must never be sent as standalone form fields
			for _, key := range []string{"filename", "contentType"} {
				if _, ok := parts[key]; ok {
					t.Errorf("%q must not be sent as a separate form field", key)
				}
			}
		})
	}
}
