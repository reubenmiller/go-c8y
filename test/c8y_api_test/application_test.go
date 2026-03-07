package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ApplicationGetWithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)
	// Test that the Get method accepts a direct ID string
	// In a real environment, this could also use: client.Applications.ByName("cockpit", "")
	item := client.Applications.Get(context.Background(), client.Applications.ByName("cockpit", "HOSTED"))
	assert.NoError(t, item.Err)
	assert.NotEmpty(t, item.Data.ID())
}
