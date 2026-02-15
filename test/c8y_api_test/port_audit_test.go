package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/auditrecords"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateAuditRecord(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	recordInput := model.AuditRecord{
		Activity:    "Test audit entry",
		Type:        "testAlarm",
		Text:        "Test audit record 1",
		Severity:    "MAJOR",
		Time:        time.Now(),
		Application: "myCITestApp",
	}

	// Create audit record
	createResult := client.AuditRecords.Create(ctx, recordInput)

	require.NoError(t, createResult.Err)
	assert.Equal(t, 201, createResult.HTTPStatus)
	assert.NotEmpty(t, createResult.Data.ID(), "Audit record id should not be empty")
	assert.Equal(t, "Test audit record 1", createResult.Data.Get("text").String())

	auditID := createResult.Data.ID()

	// Retrieve audit record by its ID
	getResult := client.AuditRecords.Get(ctx, auditID)

	require.NoError(t, getResult.Err)
	assert.Equal(t, 200, getResult.HTTPStatus)
	assert.Equal(t, auditID, getResult.Data.ID())
}

func Test_GetAuditRecords(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	testDevice := testcore.CreateDevice(t, client)

	recordType1 := "auditType1_" + testingutils.RandomString(8)
	recordType2 := "auditType2_" + testingutils.RandomString(8)

	// Create audit records
	counter := 0
	createAuditRecord := func(recordType string) {
		counter++
		recordInput := model.AuditRecord{
			Activity:    "Test audit entry",
			Type:        recordType,
			Text:        fmt.Sprintf("Test audit record %d", counter),
			Severity:    "MAJOR",
			Time:        time.Now(),
			Application: testDevice.Data.Name(),
		}
		createResult := client.AuditRecords.Create(ctx, recordInput)
		require.NoError(t, createResult.Err)
	}

	createAuditRecord(recordType1)
	createAuditRecord(recordType2)
	createAuditRecord(recordType1)
	createAuditRecord(recordType1)

	// Get audit records filtered by type
	result := client.AuditRecords.List(ctx, auditrecords.ListOptions{
		Type:        recordType1,
		Application: testDevice.Data.Name(),
	})

	require.NoError(t, result.Err)
	assert.Equal(t, 200, result.HTTPStatus)

	records, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.Equal(t, 3, len(records), "Should have 3 audit records of type1")

	// Get audit records of type2
	result2 := client.AuditRecords.List(ctx, auditrecords.ListOptions{
		Type:        recordType2,
		Application: testDevice.Data.Name(),
	})

	require.NoError(t, result2.Err)
	assert.Equal(t, 200, result2.HTTPStatus)

	records2, err := op.ToSliceR(result2)
	require.NoError(t, err)
	assert.Equal(t, 1, len(records2), "Should have 1 audit record of type2")
}
