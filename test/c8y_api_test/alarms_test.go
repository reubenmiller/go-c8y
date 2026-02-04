package c8y_api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_AlarmCount(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	count := client.Alarms.Count(context.Background(), alarms.ListOptions{})
	assert.NoError(t, count.Err)
	assert.Greater(t, count.Data, int64(0))
}

func Test_AlarmCreateWithOptions_Simple(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	ctx := c8y_api.WithMockResponses(context.Background(), true)

	// Simple alarm creation with just required fields
	result := client.Alarms.Create(
		ctx,
		alarms.CreateOptions{
			Source:   "12345", // Direct ID (no resolution needed)
			Type:     "c8y_TestAlarm",
			Text:     "Test alarm",
			Severity: "MAJOR",
		},
	)
	assert.NoError(t, result.Err)
	assert.NotEmpty(t, result.Data.ID())
}

func Test_AlarmCreateWithOptions_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	deferredCtx := c8y_api.WithDeferredExecution(context.Background(), true)
	// Using string-based resolver
	req := client.Alarms.Create(
		deferredCtx,
		alarms.CreateOptions{
			Source:   client.Alarms.DeviceResolver.ByName(mo.Data.Name()), // Resolver string
			Type:     "c8y_TestAlarm",
			Text:     "Test alarm with resolver",
			Time:     time.Now(),
			Severity: "CRITICAL",
			Status:   "ACTIVE",
			AdditionalProperties: map[string]any{
				"foo": "bar",
			},
		},
	)
	assert.NoError(t, req.Err)
	assert.True(t, req.IsDeferred())

	// confirm
	assert.NotEmpty(t, req.Meta["id"])
	assert.NotEmpty(t, req.Meta["name"])

	// execute
	result := req.Execute(context.Background())
	assert.NoError(t, result.Err)

	assert.False(t, result.IsDeferred())
	assert.NoError(t, result.Err)
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "bar", result.Data.Get("foo").String())
}

func Test_AlarmCreateWithOptions_WithCustomStruct(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	// Custom alarm type with additional fields
	type CustomAlarmData struct {
		CustomField1 string                 `json:"customField1"`
		CustomField2 int                    `json:"customField2"`
		C8yCustom    map[string]interface{} `json:"c8y_CustomFragment"`
	}

	ctx := c8y_api.WithMockResponses(context.Background(), false)

	now := time.Now()
	result := client.Alarms.Create(
		ctx,
		alarms.CreateOptions{
			Source:   mo.Data.ID(),
			Type:     "c8y_CustomAlarm",
			Text:     "Test with custom properties",
			Severity: "MINOR",
			Time:     now,
			AdditionalProperties: CustomAlarmData{
				CustomField1: "value1",
				CustomField2: 42,
				C8yCustom: map[string]interface{}{
					"temperature": 23.5,
					"humidity":    65,
				},
			},
		},
	)
	assert.NoError(t, result.Err)
	assert.NotEmpty(t, result.Data.ID())
	assert.Equal(t, "value1", result.Data.Get("customField1").String())
	assert.Equal(t, int64(42), result.Data.Get("customField2").Int())
}

func Test_AlarmCreateWithOptions_WithInlineMap(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	ctx := c8y_api.WithMockResponses(context.Background(), false)

	// Using inline map for additional properties
	result := client.Alarms.Create(
		ctx,
		alarms.CreateOptions{
			Source:   mo.Data.ID(),
			Type:     "c8y_MapAlarm",
			Text:     "Test with inline map",
			Severity: "WARNING",
			AdditionalProperties: map[string]interface{}{
				"c8y_Measurements": map[string]interface{}{
					"temperature": map[string]interface{}{
						"value": 25.3,
						"unit":  "°C",
					},
				},
				"metadata": map[string]interface{}{
					"createdBy": "test",
					"version":   "1.0",
				},
			},
		},
	)
	assert.NoError(t, result.Err)
	assert.NotEmpty(t, result.Data.ID())
}

func Test_AlarmCreateByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	result := client.Alarms.Create(
		context.Background(),
		model.Alarm{
			// For programmatic usage, use the model directly
			Source:   model.NewSource(mo.Data.ID()),
			Type:     "c8y_MapAlarm",
			Text:     "Test with inline map",
			Severity: "WARNING",
			Time:     time.Now(),
		},
	)
	assert.NoError(t, result.Err)
	assert.Greater(t, result.Data.Length(), 0)
}
