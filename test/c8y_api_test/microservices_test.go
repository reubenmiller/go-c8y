package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/microservices"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_MicroserviceUpload(t *testing.T) {
	client := testcore.CreateTestClient(t)

	app, err := client.Microservices.CreateOrUpdate(context.Background(), microservices.CreateOptions{
		File: "helloworld.zip",
	})
	assert.NoError(t, err)
	assert.Equal(t, "helloworld", app.Name)
}
