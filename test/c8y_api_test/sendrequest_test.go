package c8y_api_test

import (
	"context"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendRequest_SimpleGET(t *testing.T) {
	client := testcore.CreateTestClient(t)

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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
	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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
	// client.Client.SetDebug(true)

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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

	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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
	result := client.SendRequest(context.TODO(), c8y_api.RequestOptions{
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
