package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Token401s(t *testing.T) {
	client := testcore.CreateTestClientWithToken(t)
	client.Client.SetDebug(true)
	collection := client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	assert.NoError(t, collection.Err)
	assert.Greater(t, collection.Data.Length(), 0)
}

func Test_DeviceCertificates(t *testing.T) {
	t.Skip("TODO: generate test device certificate to be used for testing")
	client := testcore.CreateTestClientNoAuth(t)
	client.Client.SetDebug(true)
	client.SetAuth(authentication.AuthOptions{
		CertificateKey: testcore.ProjectFile("testdevice01.key"),
		Certificate:    testcore.ProjectFile("testdevice01.crt"),
	})

	_, err := client.Devices.List(context.Background(), devices.ListOptions{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, c8y_api.ErrUnauthorized)

	_, err = client.Login(context.Background())
	assert.NoError(t, err)
	assert.NotEmpty(t, client.Auth.Token)

	collection, err := client.Devices.List(context.Background(), devices.ListOptions{})
	assert.NoError(t, err)
	assert.Greater(t, len(collection.ManagedObjects), 0)
}
