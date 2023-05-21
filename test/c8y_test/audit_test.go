package c8y_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func auditRecordFactory(client *c8y.Client, recordType string, application string) func() (*c8y.AuditRecord, *c8y.Response, error) {
	counter := 1
	return func() (*c8y.AuditRecord, *c8y.Response, error) {
		counter++
		recordInput := c8y.AuditRecord{
			Activity:    "Test audit entry",
			Type:        recordType,
			Text:        fmt.Sprintf("Test alarm %d", counter),
			Severity:    "MAJOR",
			Time:        c8y.NewTimestamp(),
			Application: application,
		}

		return client.Audit.Create(
			context.Background(),
			recordInput,
		)
	}
}

func TestAuditService_CreateAuditRecord(t *testing.T) {
	client := createTestClient()

	recordInput := c8y.AuditRecord{
		Activity:    "Test audit entry",
		Type:        "testalarm",
		Text:        "Test audit record 1",
		Severity:    "MAJOR",
		Time:        c8y.NewTimestamp(),
		Application: "myCITestApp",
	}

	auditRecord, resp, err := client.Audit.Create(context.Background(), recordInput)
	testingutils.Ok(t, err)

	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, auditRecord != nil, "Audit record should not be empty")

	if auditRecord != nil {
		testingutils.Assert(t, auditRecord.ID != "", "Audit record id should not be empty")
		testingutils.Equals(t, "Test audit record 1", auditRecord.Text)
	}

	// Retrieve audit record by its ID
	auditRecord2, resp, err := client.Audit.GetAuditRecord(
		context.Background(),
		auditRecord.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, auditRecord.ID, auditRecord2.ID)

}

func TestAuditService_GetAuditRecords(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice("auditLogs")

	testingutils.Ok(t, err)

	recordType1 := "auditType 1"
	recordType2 := "auditType 2"
	createAuditRecordType1 := auditRecordFactory(client, recordType1, testDevice.Name)
	createAuditRecordType2 := auditRecordFactory(client, recordType2, testDevice.Name)

	createAuditRecordType1()
	createAuditRecordType2()
	createAuditRecordType1()
	createAuditRecordType1()

	data, resp, err := client.Audit.GetAuditRecords(
		context.Background(),
		&c8y.AuditRecordCollectionOptions{
			Type:        recordType1,
			Application: testDevice.Name,
			PaginationOptions: c8y.PaginationOptions{
				PageSize: 1000,
			},
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, 3, len(data.AuditRecords))

	// Delete the audit records
	searchOptions := &c8y.AuditRecordCollectionOptions{
		Type:        recordType1,
		Application: testDevice.Name,
	}
	resp, err = client.Audit.DeleteAuditRecords(
		context.Background(),
		searchOptions,
	)

	testingutils.Assert(t, err != nil, "deleting audit records is not allowed")
	testingutils.Equals(t, http.StatusMethodNotAllowed, resp.StatusCode())

	data2, resp, err := client.Audit.GetAuditRecords(
		context.Background(),
		&c8y.AuditRecordCollectionOptions{
			Type:        recordType2,
			Application: testDevice.Name,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, 1, len(data2.AuditRecords))
}
