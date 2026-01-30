package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
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
	software, found, err := client.Repository.Software.GetOrCreate(context.Background(), softwareitems.GetOrCreateOptions{
		Software: model.Software{
			Name:         softwareName,
			Type:         "c8y_Software",
			SoftwareType: softwareType,
			Description:  "Some custom artifact name",
		},
	})
	assert.NoError(t, err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.ID, managedobjects.DeleteOptions{})
	})
	assert.False(t, found)
	assert.Equal(t, softwareName, software.Name)
	assert.NotEmpty(t, software.ID)

	// list
	collection, err := client.Repository.Software.List(context.Background(), softwareitems.ListOptions{
		Name:         softwareName,
		SoftwareType: softwareType,
	})
	assert.NoError(t, err)
	assert.Len(t, collection.ManagedObjects, 1)

	// Get Or Create software item (existing item)
	software2, found, err := client.Repository.Software.GetOrCreate(context.Background(), softwareitems.GetOrCreateOptions{
		Software: model.Software{
			Name:         softwareName,
			Type:         "c8y_Software",
			SoftwareType: softwareType,
			Description:  "Some custom artifact name",
		},
	})
	assert.NoError(t, err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software2.ID, managedobjects.DeleteOptions{})
	})
	assert.True(t, found)
	assert.Equal(t, softwareName, software.Name)
	assert.NotEmpty(t, software2.ID)
	assert.Equal(t, software2.ID, software.ID)

	// Custom GetOrCreate command
	pagination.FindOrCreate[model.Software](
		context.Background(),
		client.Repository.Software.ListB(softwareitems.ListOptions{
			Name:         softwareName,
			SoftwareType: softwareType,
		}),
		client.Repository.Software.CreateB(model.Software{}),
		pagination.DefaultSearch(),
	)

	// get
	software3, err := client.Repository.Software.Get(context.Background(), softwareitems.GetOptions{
		ID: software.ID,
	})
	assert.NoError(t, err)
	assert.Equal(t, software3.ID, software.ID)
	assert.Equal(t, software3.Name, software.Name)

	// delete
	err = client.Repository.Software.Delete(context.Background(), softwareitems.DeleteOptions{})
	assert.NoError(t, err)
}
