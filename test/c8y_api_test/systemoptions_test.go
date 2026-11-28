package api_test

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/systemoptions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_SystemOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

	// list
	collection := client.Tenants.SystemOptions.List(context.Background(), systemoptions.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0)

	// compatible with include all even through the api does not support pagination
	it := client.Tenants.SystemOptions.ListAll(context.Background(), systemoptions.ListOptions{})
	for item, err := range it.Items() {
		assert.NoError(t, err)
		_ = item
	}

	var firstOption *model.SystemOption
	for item, err := range jsondoc.DecodeSeq2[model.SystemOption](it.Items()) {
		require.NoError(t, err)
		firstOption = item
		break
	}
	require.NotNil(t, firstOption)

	// get
	option := client.Tenants.SystemOptions.Get(context.Background(), systemoptions.GetOptions{
		Category: firstOption.Category,
		Key:      firstOption.Key,
	})
	assert.NoError(t, option.Err)
	assert.Equal(t, option.Data.Category(), firstOption.Category)
	assert.Equal(t, option.Data.Key(), firstOption.Key)
}
