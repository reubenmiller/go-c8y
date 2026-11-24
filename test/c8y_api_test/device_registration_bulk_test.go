package api_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
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
