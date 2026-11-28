package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest_SimpleGET(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
	})

	assert.NoError(t, result.Err)
	assert.False(t, result.IsError())
	assert.Equal(t, 200, result.HTTPStatus)
	assert.NotEmpty(t, result.Response)
}

func TestRequest_QueryString(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  "pageSize=5&type=c8y_Device",
	})

	assert.NoError(t, result.Err)
	assert.False(t, result.IsError())

	// Verify we can parse JSON response
	managedObjects := result.Data.Get("managedObjects")
	assert.True(t, managedObjects.Exists())
	queryParams := result.Request.URL.Query().Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=5")
}

func TestRequest_QueryStruct(t *testing.T) {
	client := testcore.CreateTestClient(t)

	type ListOptions struct {
		PageSize int    `url:"pageSize"`
		Type     string `url:"type"`
	}

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query: ListOptions{
			PageSize: 10,
			Type:     "c8y_Device",
		},
	})

	assert.NoError(t, result.Err)
	assert.False(t, result.IsError())
	queryParams := result.Request.URL.Query().Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=10")
}

func TestRequest_QueryURLValues(t *testing.T) {
	client := testcore.CreateTestClient(t)

	query := url.Values{}
	query.Add("pageSize", "10")
	query.Add("type", "c8y_Device")

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  query,
	})

	assert.NoError(t, result.Err)
	assert.False(t, result.IsError())
	queryParams := result.Request.URL.Query().Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=10")
}

func TestRequest_InlineQueryParams(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Query params in path should be parsed and added
	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects?pageSize=10",
		Query:  "type=c8y_Device", // Additional query
	})

	assert.NoError(t, result.Err)
	assert.False(t, result.IsError())
	queryParams := result.Request.URL.Query().Encode()
	assert.Contains(t, queryParams, "type=c8y_Device")
	assert.Contains(t, queryParams, "pageSize=10")
}

func TestRequest_POST(t *testing.T) {
	client := testcore.CreateTestClient(t)
	mo := testcore.CreateManagedObject(t, client)
	require.NoError(t, mo.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), mo.Data.ID(), managedobjects.DeleteOptions{})
	})

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "POST",
		Path:   "/event/events",
		Body: map[string]any{
			"source": map[string]string{"id": mo.Data.ID()},
			"time":   time.Now(),
			"type":   "ci_TestEvent",
			"text":   "Test event from SendRequest",
		},
	})

	require.NoError(t, result.Err)
	require.False(t, result.IsError())
	assert.Equal(t, 201, result.HTTPStatus)

	// Verify JSON parsing
	eventID := result.Data.Get("id").String()
	assert.NotEmpty(t, eventID)
	assert.Equal(t, "ci_TestEvent", result.Data.Get("type").String())
}

func TestRequest_CustomHeaders(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
	})

	assert.NoError(t, result.Err)
	assert.False(t, result.IsError())
	assert.Equal(t, "test-value", result.Request.Header.Get("X-Custom-Header"))
}

func TestRequest_ProcessingMode(t *testing.T) {
	client := testcore.CreateTestClient(t)
	mo := testcore.CreateManagedObject(t, client)
	require.NoError(t, mo.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), mo.Data.ID(), managedobjects.DeleteOptions{})
	})

	result := client.Request(context.TODO(), api.RequestOptions{
		Method:         "POST",
		Path:           "/event/events",
		ProcessingMode: types.ProcessingModeQuiescent,
		Body: map[string]any{
			"source": map[string]string{"id": mo.Data.ID()},
			"time":   time.Now(),
			"type":   "ci_TestEvent",
			"text":   "Test event with processing mode",
		},
	})

	assert.NoError(t, result.Err)
	assert.False(t, result.IsError())
	assert.Equal(t, string(types.ProcessingModeQuiescent), result.Request.Header.Get(types.HeaderProcessingMode))
}

func TestRequest_IgnoreAccept(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method:       "GET",
		Path:         "/inventory/managedObjects",
		IgnoreAccept: true,
	})

	assert.NoError(t, result.Err)
	assert.NotContains(t, result.Request.Header, "Accept")
	assert.False(t, result.IsError())
}

func TestRequest_CustomAccept(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Accept: "application/vnd.com.nsn.cumulocity.managedObjectCollection+json",
	})

	assert.NoError(t, result.Err)
	assert.Contains(t, result.Request.Header, "Accept")
	assert.Equal(t, "application/vnd.com.nsn.cumulocity.managedObjectCollection+json", result.Request.Header.Get("Accept"))
	assert.False(t, result.IsError())
}

func TestRequest_ResultJSON(t *testing.T) {
	client := testcore.CreateTestClient(t)
	// client.SetDebug(true)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  "pageSize=1",
	})

	require.NoError(t, result.Err)
	require.False(t, result.IsError(), "Expected successful response, got status %d: %s", result.HTTPStatus, string(result.Response))

	// Test JSON with path
	mos := result.Data.Get("managedObjects")
	assert.True(t, mos.Exists(), "managedObjects field should exist in response")
	assert.True(t, mos.IsArray(), "managedObjects should be an array")

	// Test JSON without path (full document)
	fullDoc := result.Data.Get()
	assert.True(t, fullDoc.Exists(), "Full document should parse")
	assert.True(t, fullDoc.Get("managedObjects").Exists(), "managedObjects should exist in full document")
}

func TestRequest_ResultUnmarshal(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/tenant/currentTenant",
	})

	require.NoError(t, result.Err)

	var tenant map[string]any
	err := json.Unmarshal(result.Response, &tenant)
	assert.NoError(t, err)
	assert.NotEmpty(t, tenant)
}

func TestRequest_ResultString(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects",
		Query:  "pageSize=1",
	})

	require.NoError(t, result.Err)

	str := string(result.Response)
	assert.NotEmpty(t, str)
	assert.True(t, strings.Contains(str, "managedObjects"))
}

func TestRequest_InvalidPath(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "://invalid-url",
	})

	assert.Error(t, result.Err)
	assert.True(t, result.IsError())
	assert.Contains(t, result.Err.Error(), "invalid path")
}

func TestRequest_FormData(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create a test string as a reader
	fileContent := "test file content"
	reader := strings.NewReader(fileContent)

	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "POST",
		Path:   "/inventory/binaries",
		FormData: map[string]io.Reader{
			"file":   reader,
			"object": strings.NewReader(`{"name":"testfile.txt","type":"text/plain"}`),
		},
	})

	if result.Err == nil {
		t.Cleanup(func() {
			binaryID := result.Data.Get("id").String()
			if binaryID != "" {
				client.Binaries.Delete(context.Background(), binaryID)
			}
		})
	}

	// Note: This test might fail if the endpoint expects specific multipart structure
	// The main goal is to verify the code path works
	assert.NotNil(t, result)
	assert.Equal(t, 201, result.HTTPStatus)
}

func TestRequest_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Enable dry run using the RequestOptions.DryRun field (v1 compatibility)
	dryRun := true
	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects/12345",
		DryRun: &dryRun,
	})

	// Verify no error occurred
	assert.NoError(t, result.Err)

	// Dry run should return mock data
	assert.NotNil(t, result)

	// Mock responses typically return 200 OK status
	assert.Equal(t, 200, result.HTTPStatus)

	// Verify dry run headers are set
	assert.Equal(t, "true", result.Header.Get("X-Dry-Run"), "X-Dry-Run header should be set to true")
	assert.Equal(t, "true", result.Header.Get("X-Mock-Response"), "X-Mock-Response header should be set to true")
	assert.Equal(t, "application/json", result.Header.Get("Content-Type"), "Content-Type should be application/json")

	// Response should have a body (mock data)
	body := result.Response
	assert.NotEmpty(t, body)

	// Verify we can access JSON from the mock response
	jsonResult := result.Data.Get("id")
	assert.True(t, jsonResult.Exists(), "Mock response should contain data")
}

func TestRequest_PathAndQueryEncoding(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Use dry run to test path and query encoding without making actual requests
	dryRun := true

	// Test that query parameters from path and Query field are properly merged
	// and that special characters are handled correctly
	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects?pageSize=5",
		Query:  "type=c8y_Device&withChildren=false",
		DryRun: &dryRun,
	})

	require.NoError(t, result.Err, "Request should succeed")
	require.NotNil(t, result.Response, "Response should not be nil")
	assert.Equal(t, 200, result.HTTPStatus, "Should return 200 OK in dry run")

	// Verify query parameters were properly merged and encoded
	queryParams := result.Request.URL.Query()
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
	result := client.Request(context.TODO(), api.RequestOptions{
		Host:   "https://c8y.example/base/",
		Method: "GET",
		Path:   "/path/with space?query=test eq%20'me'",
		Query:  "pageSize=100&another=%20again ",
		DryRun: &dryRun,
	})

	// path
	escapedPath := result.Request.URL.EscapedPath()
	assert.Equal(t, "/base/path/with%20space", escapedPath)

	// query parameters
	queryParams := result.Request.URL.Query()
	encodedQueryParams := queryParams.Encode()
	encodedQueryParamsParts := strings.Split(encodedQueryParams, "&")
	assert.Contains(t, encodedQueryParamsParts, "query=test+eq+%27me%27")
	assert.Contains(t, encodedQueryParamsParts, "another=+again+")
}

func TestRequest_MultipleBodyReads(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Use dry run to avoid authentication issues
	dryRun := true
	result := client.Request(context.TODO(), api.RequestOptions{
		Method: "GET",
		Path:   "/inventory/managedObjects/12345",
		DryRun: &dryRun,
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	// Read body multiple times to verify caching works
	body1 := result.Response
	body2 := result.Response
	str1 := string(result.Response)
	str2 := string(result.Response)

	// All reads should return the same data
	assert.NotEmpty(t, body1)
	assert.Equal(t, body1, body2, "Multiple Body() calls should return same data")
	assert.Equal(t, string(body1), str1, "String() should match Body()")
	assert.Equal(t, str1, str2, "Multiple String() calls should return same data")

	// JSON parsing should also work multiple times
	json1 := result.Data.Get("id")
	json2 := result.Data.Get("id")
	assert.True(t, json1.Exists())
	assert.True(t, json2.Exists())
	assert.Equal(t, json1.String(), json2.String(), "Multiple JSON() calls should return same data")

	// Unmarshal should work after all the above reads
	var response map[string]any
	err := json.Unmarshal(result.Response, &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response)
}
