package c8y_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestRetentionRuleService_CRUDRetentionRule(t *testing.T) {
	client := createTestClient()

	//
	// Create Retention rule
	retentionRule, resp, err := client.Retention.Create(
		context.Background(),
		c8y.RetentionRule{
			DataType:   "ALARM",
			MaximumAge: 10,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Assert(t, retentionRule.ID != "", "Rention Rule should have an non-empty id")

	//
	// Get retention rule by id
	retrievedRR1, resp, err := client.Retention.GetRetentionRule(
		context.Background(),
		retentionRule.ID,
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, retentionRule.ID, retrievedRR1.ID)

	//
	// Get collection of data retention rules
	rules, resp, err := client.Retention.GetRetentionRules(
		context.Background(),
		&c8y.PaginationOptions{
			PageSize: 100,
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(rules.RetentionRules) > 0, "Should have at least 1 data retention rule")

	//
	// Remove retention rule
	resp, err = client.Retention.Delete(
		context.Background(),
		retentionRule.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}
