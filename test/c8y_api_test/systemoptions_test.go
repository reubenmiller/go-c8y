package c8y_api_test

import (
	"context"
	"testing"

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
	collection, err := client.Tenants.SystemOptions.List(context.Background(), systemoptions.ListOptions{})
	assert.NoError(t, err)
	assert.Greater(t, len(collection.Options), 0)

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

	// get
	option, err := client.Tenants.SystemOptions.Get(context.Background(), systemoptions.GetOption{
		Category: collection.Options[0].Category,
		Key:      collection.Options[0].Key,
	})
	assert.NoError(t, err)
	assert.Equal(t, option.Category, collection.Options[0].Category)
	assert.Equal(t, option.Key, collection.Options[0].Key)
}
