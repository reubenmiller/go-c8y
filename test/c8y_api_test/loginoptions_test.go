package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/loginoptions"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_LoginOptionsWithoutAuth(t *testing.T) {
	client := testcore.CreateTestClient(t)

	collection, err := client.LoginOptions.ListNoAuth(context.Background(), loginoptions.ListOptions{})
	assert.NoError(t, err)
	assert.Greater(t, len(collection.LoginOptions), 0)
}
