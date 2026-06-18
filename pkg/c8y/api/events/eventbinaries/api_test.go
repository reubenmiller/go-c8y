package eventbinaries

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func testService(handler http.HandlerFunc) (*Service, func()) {
	ts := httptest.NewServer(handler)
	return NewService(&core.Service{Client: resty.New().SetBaseURL(ts.URL)}), ts.Close
}

// TestGet verifies the event id is substituted into the binaries path and the
// raw body + Content-Disposition filename are exposed via BinaryResponse.
func TestGet(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.Header().Set("Content-Disposition", `attachment; filename="report.bin"`)
		_, _ = w.Write([]byte("binary-content"))
	})
	defer closeFn()

	res := svc.Get(context.Background(), "12345")
	if res.Err != nil {
		t.Fatalf("Get: %v", res.Err)
	}
	if method != http.MethodGet {
		t.Errorf("method = %q, want GET", method)
	}
	if path != "/event/events/12345/binaries" {
		t.Errorf("path = %q, want /event/events/12345/binaries", path)
	}
	bin := res.Data
	defer bin.Close()
	if got := bin.FileName(); got != "report.bin" {
		t.Errorf("FileName = %q, want report.bin", got)
	}
	body, _ := io.ReadAll(bin.Reader())
	if string(body) != "binary-content" {
		t.Errorf("body = %q, want binary-content", string(body))
	}
}

// TestCreate verifies the multipart POST: an "object" part carries the
// name/type metadata and a "file" part carries the content.
func TestCreate(t *testing.T) {
	var path, method, gotObject, gotFile, gotFileName string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		mr, err := r.MultipartReader()
		if err != nil {
			t.Errorf("MultipartReader: %v", err)
		} else {
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Errorf("NextPart: %v", err)
					break
				}
				data, _ := io.ReadAll(part)
				switch part.FormName() {
				case "object":
					gotObject = string(data)
				case "file":
					gotFile = string(data)
					gotFileName = part.FileName()
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"b1","name":"myfile.txt"}`))
	})
	defer closeFn()

	res := svc.Create(context.Background(), "12345", UploadFileOptions{
		Reader:      strings.NewReader("hello"),
		Name:        "myfile.txt",
		ContentType: "text/plain",
	})
	if res.Err != nil {
		t.Fatalf("Create: %v", res.Err)
	}
	if method != http.MethodPost {
		t.Errorf("method = %q, want POST", method)
	}
	if path != "/event/events/12345/binaries" {
		t.Errorf("path = %q, want /event/events/12345/binaries", path)
	}
	if !strings.Contains(gotObject, `"name":"myfile.txt"`) || !strings.Contains(gotObject, `"type":"text/plain"`) {
		t.Errorf("object part = %q, want name/type metadata", gotObject)
	}
	if gotFile != "hello" {
		t.Errorf("file part = %q, want hello", gotFile)
	}
	if gotFileName != "myfile.txt" {
		t.Errorf("file part filename = %q, want myfile.txt", gotFileName)
	}
	if res.Data.Name() != "myfile.txt" {
		t.Errorf("name = %q, want myfile.txt", res.Data.Name())
	}
}

// TestUpdate verifies the binary content is replaced via an octet-stream PUT.
func TestUpdate(t *testing.T) {
	var path, method, contentType, body string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		contentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"b1"}`))
	})
	defer closeFn()

	res := svc.Update(context.Background(), "12345", UploadFileOptions{Reader: strings.NewReader("replaced")})
	if res.Err != nil {
		t.Fatalf("Update: %v", res.Err)
	}
	if method != http.MethodPut {
		t.Errorf("method = %q, want PUT", method)
	}
	if path != "/event/events/12345/binaries" {
		t.Errorf("path = %q, want /event/events/12345/binaries", path)
	}
	if contentType != "application/octet-stream" {
		t.Errorf("content-type = %q, want application/octet-stream", contentType)
	}
	if body != "replaced" {
		t.Errorf("body = %q, want replaced", body)
	}
}

// TestUpsertFallsBackToPut verifies that when the initial POST returns 409 the
// upsert replaces the existing binary via PUT.
func TestUpsertFallsBackToPut(t *testing.T) {
	var methods []string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"already exists"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"b1"}`))
	})
	defer closeFn()

	res := svc.Upsert(context.Background(), "12345", UploadFileOptions{Reader: strings.NewReader("data")})
	if res.Err != nil {
		t.Fatalf("Upsert: %v", res.Err)
	}
	if len(methods) != 2 || methods[0] != http.MethodPost || methods[1] != http.MethodPut {
		t.Errorf("methods = %v, want [POST PUT]", methods)
	}
}

// TestDelete verifies the binary is removed via DELETE on the binaries path.
func TestDelete(t *testing.T) {
	var path, method string
	svc, closeFn := testService(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	})
	defer closeFn()

	res := svc.Delete(context.Background(), "12345")
	if res.Err != nil {
		t.Fatalf("Delete: %v", res.Err)
	}
	if method != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", method)
	}
	if path != "/event/events/12345/binaries" {
		t.Errorf("path = %q, want /event/events/12345/binaries", path)
	}
}
