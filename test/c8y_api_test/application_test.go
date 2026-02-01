package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/applications"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ApplicationGetWithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	name := "cockpit"
	item := client.Applications.Get(context.Background(), applications.GetOptions{
		ApplicationRef: client.Applications.ByName(name),
	})
	assert.NoError(t, item.Err)
	assert.NotEmpty(t, item.Data.ID())
	assert.Equal(t, name, item.Data.Name())
}
