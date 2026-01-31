package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Software(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// Get Or Create software item (new item)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"
	software := client.Repository.Software.GetOrCreateByName(context.Background(), softwareName, softwareType, model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Some custom artifact name",
	})
	assert.NoError(t, software.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.Data.ID(), managedobjects.DeleteOptions{})
	})
	assert.Equal(t, false, software.Meta["found"])
	assert.Equal(t, softwareName, software.Data.Name())
	assert.NotEmpty(t, software.Data.ID())

	// list
	collection := client.Repository.Software.List(context.Background(), softwareitems.ListOptions{
		Name:         softwareName,
		SoftwareType: softwareType,
	})
	assert.NoError(t, collection.Err)
	assert.Equal(t, 1, collection.Data.Length())

	// Get Or Create software item (existing item)
	software2 := client.Repository.Software.GetOrCreateByName(context.Background(), softwareName, softwareType, model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Some custom artifact name",
	})
	assert.NoError(t, software2.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software2.Data.ID(), managedobjects.DeleteOptions{})
	})
	assert.Equal(t, true, software2.Meta["found"])
	assert.Equal(t, softwareName, software.Data.Name())
	assert.NotEmpty(t, software2.Data.ID())
	assert.Equal(t, software2.Data.ID(), software.Data.ID())

	// Custom GetOrCreate command
	result := client.Repository.Software.GetOrCreateByName(context.Background(), softwareName, softwareType, model.Software{
		Name:         softwareName,
		SoftwareType: softwareType,
	})
	assert.NoError(t, result.Err)

	// get
	software3 := client.Repository.Software.Get(context.Background(), softwareitems.GetOptions{
		ID: software.Data.ID(),
	})
	assert.NoError(t, software3.Err)
	assert.Equal(t, software3.Data.ID(), software.Data.ID())
	assert.Equal(t, software3.Data.Name(), software.Data.Name())

	// delete
	result = client.Repository.Software.Delete(context.Background(), softwareitems.DeleteOptions{
		ID: software.Data.ID(),
	})
	assert.NoError(t, result.Err)
}
