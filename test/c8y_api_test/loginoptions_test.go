package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/loginoptions"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_LoginOptionsWithoutAuth(t *testing.T) {
	client := testcore.CreateTestClient(t)

	collection := client.LoginOptions.ListNoAuth(context.Background(), loginoptions.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0)
}
