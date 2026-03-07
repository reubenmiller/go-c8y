package api_test

import (
	"context"
	"io"
	"mime"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"resty.dev/v3"
)

// completingReader wraps a reader and completes the progress bar when EOF is reached
type completingReader struct {
	io.Reader
	bar    *mpb.Bar
	closer io.Closer
}

func (r *completingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	if err == io.EOF {
		// Explicitly complete the bar when EOF is reached
		// SetTotal with current count and complete=true
		current := r.bar.Current()
		r.bar.SetTotal(current, true)
		// Also abort to ensure it completes immediately
		r.bar.Abort(true)
	}
	return n, err
}

func (r *completingReader) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

func TestSendRequest_SimpleGET(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
	})

	assert.NoError(t, result.Error)
	assert.False(t, result.IsError())
	assert.Equal(t, 200, result.StatusCode())
	assert.NotEmpty(t, result.Body())
}

func TestSendRequest_QueryString(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  "pageSize=5&type=c8y_Device",
	})

	assert.NoError(t, result.Error)
	assert.False(t, result.IsError())

	// Verify we can parse JSON response
	managedObjects := result.JSON("managedObjects")
	assert.True(t, managedObjects.Exists())
	queryParams := result.Response.Request.QueryParams.Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=5")
}

func TestSendRequest_QueryStruct(t *testing.T) {
	client := testcore.CreateTestClient(t)

	type ListOptions struct {
		PageSize int    `url:"pageSize"`
		Type     string `url:"type"`
	}

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query: ListOptions{
			PageSize: 10,
			Type:     "c8y_Device",
		},
	})

	assert.NoError(t, result.Error)
	assert.False(t, result.IsError())
	queryParams := result.Response.Request.QueryParams.Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=10")
}

func TestSendRequest_QueryURLValues(t *testing.T) {
	client := testcore.CreateTestClient(t)

	query := url.Values{}
	query.Add("pageSize", "10")
	query.Add("type", "c8y_Device")

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  query,
	})

	assert.NoError(t, result.Error)
	assert.False(t, result.IsError())
	queryParams := result.Response.Request.QueryParams.Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=10")
}

func TestSendRequest_InlineQueryParams(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Query params in path should be parsed and added
	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects?pageSize=10",
		Query:  "type=c8y_Device", // Additional query
	})

	assert.NoError(t, result.Error)
	assert.False(t, result.IsError())
	queryParams := result.Response.Request.QueryParams.Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=10")
}

func TestSendRequest_POST(t *testing.T) {
	client := testcore.CreateTestClient(t)
	mo := testcore.CreateManagedObject(t, client)
	require.NoError(t, mo.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), mo.Data.ID(), managedobjects.DeleteOptions{})
	})

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "POST",
		Path:   "/event/events",
		Body: map[string]any{
			"source": map[string]string{"id": mo.Data.ID()},
			"time":   time.Now(),
			"type":   "ci_TestEvent",
			"text":   "Test event from SendRequest",
		},
	})

	require.NoError(t, result.Error)
	require.False(t, result.IsError())
	assert.Equal(t, 201, result.StatusCode())

	// Verify JSON parsing
	eventID := result.JSON("id").String()
	assert.NotEmpty(t, eventID)
	assert.Equal(t, "ci_TestEvent", result.JSON("type").String())
}

func TestSendRequest_CustomHeaders(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
	})

	assert.NoError(t, result.Error)
	assert.False(t, result.IsError())
	assert.Equal(t, "test-value", result.Response.Request.Header.Get("X-Custom-Header"))
}

func TestSendRequest_ProcessingMode(t *testing.T) {
	client := testcore.CreateTestClient(t)
	mo := testcore.CreateManagedObject(t, client)
	require.NoError(t, mo.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), mo.Data.ID(), managedobjects.DeleteOptions{})
	})

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method:         "POST",
		Path:           "/event/events",
		ProcessingMode: types.ProcessingModeQuiescent,
		Body: map[string]interface{}{
			"source": map[string]string{"id": mo.Data.ID()},
			"time":   time.Now(),
			"type":   "ci_TestEvent",
			"text":   "Test event with processing mode",
		},
	})

	assert.NoError(t, result.Error)
	assert.False(t, result.IsError())
	assert.Equal(t, string(types.ProcessingModeQuiescent), result.Response.Request.Header.Get(types.HeaderProcessingMode))
}

func TestSendRequest_IgnoreAccept(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method:       "GET",
		Path:         "/inventory/managedObjects",
		IgnoreAccept: true,
	})

	assert.NoError(t, result.Error)
	assert.NotContains(t, result.Response.Request.Header, "Accept")
	assert.False(t, result.IsError())
}

func TestSendRequest_CustomAccept(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Accept: "application/vnd.com.nsn.cumulocity.managedObjectCollection+json",
	})

	assert.NoError(t, result.Error)
	assert.Contains(t, result.Response.Request.Header, "Accept")
	assert.Equal(t, "application/vnd.com.nsn.cumulocity.managedObjectCollection+json", result.Response.Request.Header.Get("Accept"))
	assert.False(t, result.IsError())
}

func TestRequestResult_JSON(t *testing.T) {
	client := testcore.CreateTestClient(t)
	// client.SetDebug(true)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  "pageSize=1",
	})

	require.NoError(t, result.Error)
	require.False(t, result.IsError(), "Expected successful response, got status %d: %s", result.StatusCode(), result.String())

	// Test JSON with path
	mos := result.JSON("managedObjects")
	assert.True(t, mos.Exists(), "managedObjects field should exist in response")
	assert.True(t, mos.IsArray(), "managedObjects should be an array")

	// Test JSON without path (full document)
	fullDoc := result.JSON()
	assert.True(t, fullDoc.Exists(), "Full document should parse")
	assert.True(t, fullDoc.Get("managedObjects").Exists(), "managedObjects should exist in full document")
}

func TestRequestResult_Unmarshal(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/tenant/currentTenant",
	})

	require.NoError(t, result.Error)

	var tenant map[string]interface{}
	err := result.Unmarshal(&tenant)
	assert.NoError(t, err)
	assert.NotEmpty(t, tenant)
}

func TestRequestResult_String(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  "pageSize=1",
	})

	require.NoError(t, result.Error)

	str := result.String()
	assert.NotEmpty(t, str)
	assert.True(t, strings.Contains(str, "managedObjects"))
}

func TestSendRequest_InvalidPath(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "://invalid-url",
	})

	assert.Error(t, result.Error)
	assert.True(t, result.IsError())
	assert.Contains(t, result.Error.Error(), "invalid path")
}

func TestSendRequest_FormData(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create a test string as a reader
	fileContent := "test file content"
	reader := strings.NewReader(fileContent)

	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "POST",
		Path:   "/inventory/binaries",
		FormData: map[string]io.Reader{
			"file":   reader,
			"object": strings.NewReader(`{"name":"testfile.txt","type":"text/plain"}`),
		},
	})

	if result.Error == nil {
		t.Cleanup(func() {
			binaryID := result.JSON("id").String()
			if binaryID != "" {
				client.Binaries.Delete(context.Background(), binaryID)
			}
		})
	}

	// Note: This test might fail if the endpoint expects specific multipart structure
	// The main goal is to verify the code path works
	assert.NotNil(t, result)
	assert.Equal(t, 201, result.StatusCode())
}

func TestSendRequest_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run using the RequestOptions.DryRun field (v1 compatibility)
	dryRun := true
	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects/12345",
		DryRun: &dryRun,
	})

	// Verify no error occurred
	assert.NoError(t, result.Error)

	// Dry run should return mock data
	assert.NotNil(t, result)

	// Mock responses typically return 200 OK status
	assert.Equal(t, 200, result.StatusCode())

	// Verify dry run headers are set
	assert.Equal(t, "true", result.Header("X-Dry-Run"), "X-Dry-Run header should be set to true")
	assert.Equal(t, "true", result.Header("X-Mock-Response"), "X-Mock-Response header should be set to true")
	assert.Equal(t, "application/json", result.Header("Content-Type"), "Content-Type should be application/json")

	// Response should have a body (mock data)
	body := result.Body()
	assert.NotEmpty(t, body)

	// Verify we can access JSON from the mock response
	jsonResult := result.JSON("id")
	assert.True(t, jsonResult.Exists(), "Mock response should contain data")
}

func TestSendRequest_PathAndQueryEncoding(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Use dry run to test path and query encoding without making actual requests
	dryRun := true

	// Test that query parameters from path and Query field are properly merged
	// and that special characters are handled correctly
	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects?pageSize=5",
		Query:  "type=c8y_Device&withChildren=false",
		DryRun: &dryRun,
	})

	require.NoError(t, result.Error, "Request should succeed")
	require.NotNil(t, result.Response, "Response should not be nil")
	assert.Equal(t, 200, result.StatusCode(), "Should return 200 OK in dry run")

	// Verify query parameters were properly merged and encoded
	queryParams := result.Response.Request.QueryParams
	assert.NotEmpty(t, queryParams)

	// Both inline path query params and additional Query params should be present
	encodedQueryParamsParts := strings.Split(queryParams.Encode(), "&")
	assert.Contains(t, encodedQueryParamsParts, "pageSize=5", "Query param from path should be present")
	assert.Contains(t, encodedQueryParamsParts, "type=c8y_Device", "Query param from Query field should be present")
	assert.Contains(t, encodedQueryParamsParts, "withChildren=false", "Additional query param should be present")
}

func Test_ParseRequestWithSpaces(t *testing.T) {
	client := testcore.CreateTestClient(t)
	dryRun := true
	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Host:   "https://c8y.example/base/",
		Method: "GET",
		Path:   "/path/with space?query=test eq%20'me'",
		Query:  "pageSize=100&another=%20again ",
		DryRun: &dryRun,
	})

	// path
	escapedPath := result.Response.Request.RawRequest.URL.EscapedPath()
	assert.Equal(t, "/base/path/with%20space", escapedPath)

	// query parameters
	queryParams := result.Response.Request.QueryParams
	encodedQueryParams := queryParams.Encode()
	encodedQueryParamsParts := strings.Split(encodedQueryParams, "&")
	assert.Contains(t, encodedQueryParamsParts, "query=test+eq+%27me%27")
	assert.Contains(t, encodedQueryParamsParts, "another=+again+")
}

func TestSendRequest_MultipleBodyReads(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Use dry run to avoid authentication issues
	dryRun := true
	result := client.SendRequest(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects/12345",
		DryRun: &dryRun,
	})

	require.NoError(t, result.Error)
	assert.Equal(t, 200, result.StatusCode())

	// Read body multiple times to verify caching works
	body1 := result.Body()
	body2 := result.Body()
	str1 := result.String()
	str2 := result.String()

	// All reads should return the same data
	assert.NotEmpty(t, body1)
	assert.Equal(t, body1, body2, "Multiple Body() calls should return same data")
	assert.Equal(t, string(body1), str1, "String() should match Body()")
	assert.Equal(t, str1, str2, "Multiple String() calls should return same data")

	// JSON parsing should also work multiple times
	json1 := result.JSON("id")
	json2 := result.JSON("id")
	assert.True(t, json1.Exists())
	assert.True(t, json2.Exists())
	assert.Equal(t, json1.String(), json2.String(), "Multiple JSON() calls should return same data")

	// Unmarshal should work after all the above reads
	var response map[string]interface{}
	err := result.Unmarshal(&response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
}

// countingReader is a simple io.Reader that counts bytes read
type countingReader struct {
	io.Reader
	count int64
}

func (r *countingReader) Read(p []byte) (n int, err error) {
	n, err = r.Reader.Read(p)
	r.count += int64(n)
	return n, err
}

func (r *countingReader) Close() error {
	if closer, ok := r.Reader.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

func TestSendRequest_CustomBodyWrapper_Simple(t *testing.T) {
	client := testcore.CreateTestClient(t)

	var counter *countingReader
	var wrapperCalled bool

	dryRun := false // Test with real response
	options := api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Accept: types.MimeTypeApplicationJSON,
		DryRun: &dryRun,
		OnResponse: func(response *resty.Response) error {
			wrapperCalled = true
			t.Logf("OnResponse called, ContentLength: %d", response.RawResponse.ContentLength)

			// Wrap the response body with a counting reader
			// Use response.Body (Resty's body) for consistency
			counter = &countingReader{
				Reader: response.Body,
			}
			response.Body = counter
			response.RawResponse.Body = counter

			return nil
		},
		DoNotParseResponse: true,
	}

	resp := client.SendRequest(context.Background(), options)

	// Even if authentication fails, OnResponse should still be called
	// and we should be able to read the error response body
	if resp.StatusCode() == 401 {
		t.Logf("Got 401, but OnResponse should still work")

		// Read the error body using response.Body (Resty's body)
		contents, err := io.ReadAll(resp.Response.Body)
		assert.NoError(t, err)
		assert.NotEmpty(t, contents)
		t.Logf("Read %d bytes from error response, counter tracked %d bytes", len(contents), counter.count)

		resp.Response.Body.Close()

		// Verify the wrapper worked even for error responses
		assert.True(t, wrapperCalled, "OnResponse callback should have been called")
		assert.NotNil(t, counter, "Counter should have been created")
		assert.Equal(t, int64(len(contents)), counter.count, "Counter should track all bytes read")

		return
	}

	assert.NoError(t, resp.Error)

	// Read the body
	contents, err := io.ReadAll(resp.Response.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, contents)

	t.Logf("Read %d bytes, counter tracked %d bytes", len(contents), counter.count)

	// Close the body
	resp.Response.Body.Close()

	// Verify the mechanism worked
	assert.True(t, wrapperCalled, "OnResponse callback should have been called")
	assert.NotNil(t, counter, "Counter should have been created")
	assert.Equal(t, int64(len(contents)), counter.count, "Counter should track all bytes read")
}

func TestSendRequest_CustomBodyWriter(t *testing.T) {
	client := testcore.CreateTestClient(t)

	progressOut := new(strings.Builder)

	progress := mpb.New(
		mpb.WithOutput(progressOut),
		mpb.WithWidth(180),
		mpb.WithRefreshRate(100*time.Millisecond), // Explicit refresh rate for testing
	)

	var bodyContents []byte
	var bar *mpb.Bar
	var wrapperCalled bool

	dryRun := false // Test with real response to ensure it works in production
	options := api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Accept: types.MimeTypeApplicationJSON,
		DryRun: &dryRun,
		// OnResponse is called BEFORE the body is read, allowing us to wrap it
		OnResponse: func(response *resty.Response) error {
			wrapperCalled = true

			basename := "download"
			_, params, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
			if err == nil {
				if filename, ok := params["filename"]; ok {
					basename = filename
				}
			}

			// Note: ContentLength is set to -1 if the response is chunked/compressed
			// For unknown sizes, set total to 0 and let it auto-complete on EOF
			barTotal := int64(0)
			if response.RawResponse.ContentLength > 0 {
				barTotal = response.RawResponse.ContentLength
			}

			bar = progress.AddBar(barTotal,
				mpb.PrependDecorators(
					decor.Name("elapsed", decor.WC{W: len("elapsed") + 1, C: decor.DindentRight}),
					decor.Elapsed(decor.ET_STYLE_MMSS, decor.WC{W: 8, C: decor.DindentRight}),
					decor.Name(basename, decor.WC{W: len(basename) + 1, C: decor.DindentRight}),
				),
				mpb.AppendDecorators(
					decor.Percentage(decor.WC{W: 6, C: decor.DindentRight}),
					decor.CountersKibiByte("% .2f / % .2f"),
				),
			)

			// Wrap the response body with the progress bar proxy reader
			// Use response.Body (Resty's body) which is the correct body to wrap
			wrappedBody := &completingReader{
				Reader: bar.ProxyReader(response.Body),
				bar:    bar,
				closer: response.Body,
			}

			response.Body = wrappedBody

			return nil
		},
		DoNotParseResponse: true,
	}

	resp := client.SendRequest(context.Background(), options)

	// Read the body (it's now wrapped with the progress bar)
	// Use response.Body (Resty's body field) which we wrapped in OnResponse
	contents, err := io.ReadAll(resp.Response.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, contents)
	bodyContents = contents

	// Close the body
	resp.Response.Body.Close()

	// Wait for progress bars to complete
	progress.Wait()

	// Verify the mechanism worked
	assert.True(t, wrapperCalled, "OnResponse callback should have been called")
	assert.NotNil(t, bar, "Progress bar should have been created")
	assert.NotEmpty(t, bodyContents, "Body should have been read")
	assert.True(t, bar.Completed(), "Progress bar should be completed")

	// Verify the bar tracked the bytes
	bytesRead := bar.Current()
	assert.Equal(t, int64(len(bodyContents)), bytesRead, "Progress bar should track all bytes read")
}
