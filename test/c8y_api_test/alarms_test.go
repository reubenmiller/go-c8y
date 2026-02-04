package c8y_api_test

import (
	"context"
	"testing"

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

func Test_AlarmCreateByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	count := client.Alarms.Create(
		context.Background(),
		model.Alarm{
			// Source: ,
		},
	)
	assert.NoError(t, count.Err)
	assert.Greater(t, count.Data, int64(0))
}
