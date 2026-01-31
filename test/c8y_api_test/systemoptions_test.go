package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
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
	out := make(chan model.SystemOption)
	go pagination.ForEach(
		context.Background(),
		client.Tenants.SystemOptions.ListB(systemoptions.ListOptions{}),
		pagination.IncludeAll(),
		out,
	)
	for item := range out {
		_ = item
	}

	// TODO: find a nicer way to access the first item in the results
	var firstOption model.SystemOption
	for item := range jsondoc.DecodeIter[model.SystemOption](collection.Data.Iter()) {
		firstOption = *item
		break
	}

	// get
	option := client.Tenants.SystemOptions.Get(context.Background(), systemoptions.GetOption{
		Category: firstOption.Category,
		Key:      firstOption.Key,
	})
	assert.NoError(t, option.Err)
	assert.Equal(t, option.Data.Category(), firstOption.Category)
	assert.Equal(t, option.Data.Key(), firstOption.Key)
}
