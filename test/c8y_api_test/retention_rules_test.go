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
	rules := client.RetentionRules.List(context.Background(), retentionrules.ListOptions{})
	assert.NoError(t, rules.Err)
	assert.Greater(t, rules.Data.Length(), 0)

	if rules.Data.Length() > 0 {
		ruleID := ""
		// TODO: Add a better iterator for retention rules, or have a method which returns a plain array
		for item := range rules.Data.Iter() {
			ruleID = item.Get("id").String()
			break
		}
		rule := client.RetentionRules.Get(context.Background(), ruleID)
		assert.NoError(t, rule.Err)
		assert.Equal(t, rule.Data.ID(), ruleID)
	}
}
