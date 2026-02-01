package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/systemoptions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_SystemOptions(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// list
	collection := client.Tenants.SystemOptions.List(context.Background(), systemoptions.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0)

	// compatible with include all even through the api does not support pagination
	it := client.Tenants.SystemOptions.ListAll(context.Background(), systemoptions.ListOptions{})
	for item := range it.Items() {
		_ = item
	}

	firstOption, found := op.First(jsondoc.DecodeIter[model.SystemOption](it.Items()))
	assert.True(t, found)
	assert.NoError(t, firstOption.Err)

	// get
	option := client.Tenants.SystemOptions.Get(context.Background(), systemoptions.GetOption{
		Category: firstOption.Data.Category,
		Key:      firstOption.Data.Key,
	})
	assert.NoError(t, option.Err)
	assert.Equal(t, option.Data.Category(), firstOption.Data.Category)
	assert.Equal(t, option.Data.Key(), firstOption.Data.Key)
}
