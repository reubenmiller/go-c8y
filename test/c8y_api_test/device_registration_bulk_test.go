package api_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices/registration"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/identity"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Registration_CreateBulk(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	ids := []string{"bulk" + testingutils.RandomString(10), "bulk" + testingutils.RandomString(10)}
	t.Cleanup(func() {
		for _, id := range ids {
			ident := client.Identity.Get(ctx, identity.IdentityOptions{ExternalID: id, Type: "c8y_Serial"})
			if ident.Err == nil {
				client.ManagedObjects.Delete(ctx, ident.Data.ManagedObjectID(), managedobjects.DeleteOptions{})
			}
		}
	})

	var csv strings.Builder
	csv.WriteString("ID\tCREDENTIALS\tNAME\tTYPE\tIDTYPE\n")
	for _, id := range ids {
		fmt.Fprintf(&csv, "%s\t%s\t%s\tc8y_Test\tc8y_Serial\n", id, "Pass-"+testingutils.RandomString(16), id)
	}

	result := client.Devices.Registration.CreateBulk(ctx, registration.UploadFileOptions{
		Reader: strings.NewReader(csv.String()),
		Name:   "devices.csv",
	})
	require.NoError(t, result.Err)
	assert.EqualValues(t, 2, result.Data.Total())
	assert.EqualValues(t, 0, result.Data.TotalFailed())

	for _, id := range ids {
		ident := client.Identity.Get(ctx, identity.IdentityOptions{ExternalID: id, Type: "c8y_Serial"})
		require.NoError(t, ident.Err, "each device gets its external id")
		mo := client.ManagedObjects.Get(ctx, ident.Data.ManagedObjectID(), managedobjects.GetOptions{})
		require.NoError(t, mo.Err)
		assert.Equal(t, id, mo.Data.Name())
	}

	// Registering the same devices again fails per row, not for the request.
	again := client.Devices.Registration.CreateBulk(ctx, registration.UploadFileOptions{
		Reader: strings.NewReader(csv.String()),
		Name:   "devices.csv",
	})
	require.NoError(t, again.Err)
	assert.EqualValues(t, 2, again.Data.TotalFailed())
	for _, failure := range again.Data.FailedCreationList() {
		assert.NotEmpty(t, failure.FailureReason())
	}
}

func Test_Registration_Register(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	id := "reg" + testingutils.RandomString(10)
	opt := registration.RegisterOptions{
		ExternalID:  id,
		AuthType:    registration.AuthTypeBasic,
		Credentials: "Pass-" + testingutils.RandomString(16),
		Type:        "c8y_Test",
	}
	result := client.Devices.Registration.Register(ctx, opt)
	require.NoError(t, result.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(ctx, result.Data.ManagedObjectID(), managedobjects.DeleteOptions{})
	})
	assert.Equal(t, id, result.Data.ExternalID())
	assert.NotEmpty(t, result.Data.ManagedObjectID())

	again := client.Devices.Registration.Register(ctx, opt)
	require.Error(t, again.Err, "an external ID that exists is an error")
	assert.Contains(t, again.Err.Error(), id)
}

func Test_Registration_Register_DryRun(t *testing.T) {
	testcore.SkipLive(t)
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	ctx := api.WithDryRun(context.Background(), true)

	before := srv.ManagedObjects.Count()
	var requests []string
	ctx = api.WithDryRunHandler(ctx, func(req *http.Request) {
		requests = append(requests, req.Method+" "+req.URL.Path)
	})
	result := client.Devices.Registration.Register(ctx, registration.RegisterOptions{ExternalID: "dryrun01", AuthType: registration.AuthTypeBasic, Credentials: "x"})
	require.NoError(t, result.Err)
	assert.Equal(t, []string{"POST /devicecontrol/bulkNewDeviceRequests"}, requests, "only the registration is shown")
	assert.Equal(t, before, srv.ManagedObjects.Count(), "nothing is created")
	n := 0
	for range result.Documents() {
		n++
	}
	assert.Zero(t, n)
}
