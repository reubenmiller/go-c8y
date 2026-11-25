package api_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	api "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/binaries"
)

func exampleClient() (*api.Client, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"1","name":"firmware.bin"}`))
	}))
	client := api.NewClient(api.ClientOptions{
		BaseURL:          srv.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant: "t1", Username: "user", Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
	return client, srv.Close
}

// A request hook runs on the final request of every call made with the
// context, typed or raw. Here it adds a header.
func ExampleWithRequestHook() {
	client, closeServer := exampleClient()
	defer closeServer()

	ctx := api.WithRequestHook(context.Background(), func(req *http.Request) error {
		req.Header.Set("X-Trace", "abc")
		fmt.Println("sending", req.Method, req.URL.Path)
		return nil
	})
	result := client.Request(ctx, api.RequestOptions{Method: http.MethodPost, Path: "/inventory/managedObjects", Body: map[string]any{"name": "x"}})
	fmt.Println(result.HTTPStatus, result.Data.Get("name").String())
	// Output:
	// sending POST /inventory/managedObjects
	// 201 firmware.bin
}

// UploadFileOptions.Progress reports the file bytes uploaded by any typed
// upload (binaries, event binaries, applications, firmware, software, ...).
func ExampleClient_Request_uploadProgress() {
	client, closeServer := exampleClient()
	defer closeServer()

	data := bytes.Repeat([]byte("x"), 1000)
	var last, total int64
	result := client.Binaries.Create(context.Background(), binaries.UploadFileOptions{
		Reader: bytes.NewReader(data),
		Name:   "firmware.bin",
		Progress: func(read, size int64) {
			last, total = read, size
		},
	})
	fmt.Println(result.Err, last, total)
	// Output: <nil> 1000 1000
}
