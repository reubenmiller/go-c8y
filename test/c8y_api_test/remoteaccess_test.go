package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/remoteaccess/remoteaccess_configurations"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_RemoteAccess(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

	ctx := context.Background()

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	client.ManagedObjects.Update(context.Background(), mo.Data.ID(), map[string]any{
		"c8y_SupportedOperations": []string{
			"c8y_Command",
			"c8y_DeviceProfile",
			"c8y_DownloadConfigFile",
			"c8y_Firmware",
			"c8y_LogfileRequest",
			"c8y_RemoteAccessConnect",
			"c8y_Restart",
			"c8y_SoftwareUpdate",
			"c8y_UploadConfigFile",
		},
	})

	// create
	createResult := client.RemoteAccess.Configurations.Create(ctx, remoteaccess_configurations.CreateOptions{
		ManagedObjectID: mo.Data.ID(),
		Body: model.RemoteAccessConfiguration{
			Name:            "test",
			Hostname:        "127.0.0.1",
			Port:            22,
			CredentialsType: model.RemoteAccessCredentialsTypeNone,
			Protocol:        model.RemoteAccessProtocolPassthrough,
		},
	})
	assert.NoError(t, createResult.Err)
	assert.NotEmpty(t, createResult.Data.ID())

	// update
	updateResult := client.RemoteAccess.Configurations.Update(ctx, remoteaccess_configurations.UpdateOptions{
		ManagedObjectID: mo.Data.ID(),
		ConfigurationID: createResult.Data.ID(),
		Body: model.RemoteAccessConfiguration{
			Name:            "test",
			Hostname:        "127.0.1.1",
			Port:            22,
			CredentialsType: model.RemoteAccessCredentialsTypeNone,
			Protocol:        model.RemoteAccessProtocolPassthrough,
		},
	})
	assert.NoError(t, updateResult.Err)
	assert.NotEmpty(t, updateResult.Data.ID())

	listResult := client.RemoteAccess.Configurations.List(context.Background(), remoteaccess_configurations.ListOptions{
		ManagedObjectID: mo.Data.ID(),
	})

	assert.NoError(t, listResult.Err)
	assert.GreaterOrEqual(t, listResult.Data.Length(), 1)
	for item := range op.Iter(listResult) {
		assert.NotEmpty(t, item.Hostname())
	}

	// delete
	deleteResult := client.RemoteAccess.Configurations.Delete(ctx, remoteaccess_configurations.DeleteOptions{
		ManagedObjectID: mo.Data.ID(),
		ConfigurationID: createResult.Data.ID(),
	})
	assert.NoError(t, deleteResult.Err)
}
