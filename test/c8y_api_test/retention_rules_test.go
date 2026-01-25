package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/retentionrules"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_RetentionRules(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	rules, err := client.RetentionRules.List(context.Background(), retentionrules.ListOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, rules)

	if len(rules.RetentionRules) > 0 {
		rule, err := client.RetentionRules.Get(context.Background(), rules.RetentionRules[0].ID)
		assert.NoError(t, err)
		assert.Equal(t, rule.ID, rules.RetentionRules[0].ID)
	}
}
