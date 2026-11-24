package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Microservices_ListServiceUsers(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	app, found := client.Microservices.FindFirst(ctx, microservices.ListOptions{})
	if !found || app.Err != nil {
		t.Skip("no microservice to read the service users of")
	}

	users := client.Microservices.ListServiceUsers(ctx, app.Data.ID())
	require.NoError(t, users.Err)
	n := 0
	for user, err := range users.Items() {
		require.NoError(t, err)
		assert.NotEmpty(t, user.Get("tenant").String())
		n++
	}
	assert.Positive(t, n, "a microservice has at least the service user of its own tenant")

	byName := client.Microservices.ListServiceUsers(ctx, "name:"+app.Data.Name())
	require.NoError(t, byName.Err, "the microservice can also be given by name")
}
