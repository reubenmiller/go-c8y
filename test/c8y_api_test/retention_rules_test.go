package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/retentionrules"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RetentionRules(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)
	rules := client.RetentionRules.List(context.Background(), retentionrules.ListOptions{})
	assert.NoError(t, rules.Err)
	assert.Greater(t, rules.Data.Length(), 0)

	if rules.Data.Length() > 0 {

		firstRule, err := rules.First()
		require.NoError(t, err)

		rule := client.RetentionRules.Get(context.Background(), firstRule.ID())
		assert.NoError(t, rule.Err)
		assert.Equal(t, rule.Data.ID(), firstRule.ID())
	}
}
