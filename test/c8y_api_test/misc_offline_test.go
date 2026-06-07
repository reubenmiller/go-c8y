package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices/registration"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Tenants_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	createRes := client.Tenants.Create(ctx, map[string]any{
		"company":    "Foo",
		"adminName":  "fooadmin",
		"adminEmail": "admin@example.com",
		"adminPass":  "secret",
		"domain":     "foo.cumulocity.com",
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	t.Cleanup(func() {
		client.Tenants.Delete(ctx, id, tenants.DeleteOptions{})
	})

	getRes := client.Tenants.Get(ctx, id)
	require.NoError(t, getRes.Err)
	assert.Equal(t, id, getRes.Data.ID())

	updRes := client.Tenants.Update(ctx, id, map[string]any{"company": "Bar"})
	require.NoError(t, updRes.Err)

	listRes := client.Tenants.List(ctx, tenants.ListOptions{})
	require.NoError(t, listRes.Err)

	itAll := client.Tenants.ListAll(ctx, tenants.ListOptions{})
	require.NoError(t, itAll.Err())

	appRefRes := client.Tenants.ListApplicationReferences(ctx, "t12345", tenants.ListApplicationReferencesOptions{})
	require.NoError(t, appRefRes.Err)

	appRefAll := client.Tenants.ListAllApplicationReferences(ctx, "t12345", tenants.ListApplicationReferencesOptions{})
	require.NoError(t, appRefAll.Err())

	tfaRes := client.Tenants.GetTFA(ctx, "t12345")
	require.NoError(t, tfaRes.Err)

	updTFA := client.Tenants.UpdateTFA(ctx, "t12345", map[string]any{"strategy": "TOTP"})
	require.NoError(t, updTFA.Err)
}

func Test_Devices_Registration_Lifecycle(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	createRes := client.Devices.Registration.Create(ctx, registration.CreateOptions{
		ID:   "device-001-" + t.Name(),
		Type: "c8y_Linux",
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()

	getRes := client.Devices.Registration.Get(ctx, id)
	require.NoError(t, getRes.Err)

	listRes := client.Devices.Registration.List(ctx, registration.ListOptions{})
	require.NoError(t, listRes.Err)

	updRes := client.Devices.Registration.Update(ctx, id, registration.UpdateOptions{
		Status: "ACCEPTED",
	})
	require.NoError(t, updRes.Err)

	credRes := client.Devices.Registration.CreateCredentials(ctx, registration.CreateCredentialsOptions{
		ID: id,
	})
	require.NoError(t, credRes.Err)

	delRes := client.Devices.Registration.Delete(ctx, id)
	require.NoError(t, delRes.Err)
}

func Test_Devices_Registration_GeneratePassword(t *testing.T) {
	client := testcore.CreateTestClient(t)
	pw, err := client.Devices.Registration.GeneratePassword()
	require.NoError(t, err)
	assert.NotEmpty(t, pw)
}

func Test_Microservices_CurrentMicroservice(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	getRes := client.Microservices.CurrentMicroservice.Get(ctx)
	require.NoError(t, getRes.Err)

	users := client.Microservices.CurrentMicroservice.ListUsers(ctx)
	require.NoError(t, users.Err)

	settings := client.Microservices.CurrentMicroservice.ListSettings(ctx)
	require.NoError(t, settings.Err)
}
