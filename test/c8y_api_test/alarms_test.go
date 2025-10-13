package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alarms"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_AlarmCount(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	count, err := client.Alarms.Count(context.Background(), alarms.ListOptions{})
	assert.NoError(t, err)
	assert.Greater(t, count, int64(0))
}
