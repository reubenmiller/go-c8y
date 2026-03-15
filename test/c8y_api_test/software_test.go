package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/repository/software/softwareitems"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_Software(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

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
	software3 := client.Repository.Software.Get(
		context.Background(),
		software.Data.ID(),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, software3.Err)
	assert.Equal(t, software3.Data.ID(), software.Data.ID())
	assert.Equal(t, software3.Data.Name(), software.Data.Name())

	// delete
	result2 := client.Repository.Software.Delete(context.Background(), software.Data.ID(), softwareitems.DeleteOptions{})
	assert.NoError(t, result2.Err)
}

func Test_SoftwareResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

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

	// Get By name with deferred execution
	deferredGet := client.Repository.Software.Get(
		api.WithDeferredExecution(context.Background(), true),
		softwareitems.NewRef().ByName(softwareName),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, deferredGet.Err)
	assert.True(t, deferredGet.IsDeferred())
	assert.NotEmpty(t, deferredGet.Meta["id"])

	// execute
	result := deferredGet.Execute(context.Background())
	assert.NoError(t, result.Err)
	assert.Equal(t, software.Data.ID(), result.Data.ID())
}

func Test_SoftwareResolver_ByID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"

	// Create software item
	software := client.Repository.Software.Create(context.Background(), model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Test software",
	})
	assert.NoError(t, software.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.Data.ID(), managedobjects.DeleteOptions{})
	})

	// Test direct ID resolution
	result := client.Repository.Software.Get(
		context.Background(),
		software.Data.ID(),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, result.Err)
	assert.Equal(t, software.Data.ID(), result.Data.ID())
	assert.Equal(t, "id", result.Meta["resolverType"])

	// Test Ref.ByID
	result2 := client.Repository.Software.Get(
		context.Background(),
		softwareitems.NewRef().ByID(software.Data.ID()),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, result2.Err)
	assert.Equal(t, software.Data.ID(), result2.Data.ID())
}

func Test_SoftwareResolver_ByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"

	// Create software item
	software := client.Repository.Software.Create(context.Background(), model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Test software",
	})
	assert.NoError(t, software.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.Data.ID(), managedobjects.DeleteOptions{})
	})

	// Test name-only resolution
	result := client.Repository.Software.Get(
		context.Background(),
		softwareitems.NewRef().ByName(softwareName),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, result.Err)
	assert.Equal(t, software.Data.ID(), result.Data.ID())
	assert.Equal(t, "name", result.Meta["resolverType"])
	assert.Equal(t, softwareName, result.Meta["name"])

	// Test name + type resolution
	result2 := client.Repository.Software.Get(
		context.Background(),
		softwareitems.NewRef().ByName(softwareName, softwareType),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, result2.Err)
	assert.Equal(t, software.Data.ID(), result2.Data.ID())
	assert.Equal(t, "nameAndType", result2.Meta["resolverType"])
	assert.Equal(t, softwareName, result2.Meta["name"])
	assert.Equal(t, softwareType, result2.Meta["softwareType"])
}

func Test_SoftwareResolver_ByQuery(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"

	// Create software item
	software := client.Repository.Software.Create(context.Background(), model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Test software",
	})
	assert.NoError(t, software.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.Data.ID(), managedobjects.DeleteOptions{})
	})

	// Test query resolution
	result := client.Repository.Software.Get(
		context.Background(),
		softwareitems.NewRef().ByQuery("name eq '"+softwareName+"'"),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, result.Err)
	assert.Equal(t, software.Data.ID(), result.Data.ID())
	assert.Equal(t, "query", result.Meta["resolverType"])
}

func Test_SoftwareResolver_Errors(t *testing.T) {
	client := testcore.CreateTestClient(t)

	t.Run("empty identifier", func(t *testing.T) {
		result := client.Repository.Software.Get(
			context.Background(),
			"",
			softwareitems.GetOptions{},
		)
		assert.Error(t, result.Err)
		assert.Contains(t, result.Err.Error(), "cannot be empty")
	})

	t.Run("not found by name", func(t *testing.T) {
		result := client.Repository.Software.Get(
			context.Background(),
			softwareitems.NewRef().ByName("nonexistent-software-"+testingutils.RandomString(16)),
			softwareitems.GetOptions{},
		)
		assert.Error(t, result.Err)
		assert.Contains(t, result.Err.Error(), "not found")
	})

	t.Run("not found by query", func(t *testing.T) {
		result := client.Repository.Software.Get(
			context.Background(),
			softwareitems.NewRef().ByQuery("name eq 'definitely-does-not-exist-"+testingutils.RandomString(16)+"'"),
			softwareitems.GetOptions{},
		)
		assert.Error(t, result.Err)
		assert.Contains(t, result.Err.Error(), "not found")
	})
}

func Test_SoftwareUpdate_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"

	// Create software item
	software := client.Repository.Software.Create(context.Background(), model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Test software",
	})
	assert.NoError(t, software.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.Data.ID(), managedobjects.DeleteOptions{})
	})

	// Update by name
	newDescription := "Updated description"
	updateResult := client.Repository.Software.Update(
		context.Background(),
		softwareitems.NewRef().ByName(softwareName, softwareType),
		model.Software{
			Description: newDescription,
		},
	)
	assert.NoError(t, updateResult.Err)
	assert.Equal(t, newDescription, updateResult.Data.Description())
	assert.Equal(t, "nameAndType", updateResult.Meta["resolverType"])
}

func Test_SoftwareDelete_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"

	// Create software item
	software := client.Repository.Software.Create(context.Background(), model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Test software",
	})
	assert.NoError(t, software.Err)

	// Delete by name
	deleteResult := client.Repository.Software.Delete(
		context.Background(),
		softwareitems.NewRef().ByName(softwareName, softwareType),
		softwareitems.DeleteOptions{},
	)
	assert.NoError(t, deleteResult.Err)
	assert.Equal(t, "nameAndType", deleteResult.Meta["resolverType"])

	// Verify deletion
	getResult := client.Repository.Software.Get(
		context.Background(),
		software.Data.ID(),
		softwareitems.GetOptions{},
	)
	assert.Error(t, getResult.Err)
}

func Test_SoftwareDeferredExecution(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"

	// Create software item
	software := client.Repository.Software.Create(context.Background(), model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Test software",
	})
	assert.NoError(t, software.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.Data.ID(), managedobjects.DeleteOptions{})
	})

	t.Run("Get deferred", func(t *testing.T) {
		deferredResult := client.Repository.Software.Get(
			api.WithDeferredExecution(context.Background(), true),
			softwareitems.NewRef().ByName(softwareName, softwareType),
			softwareitems.GetOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
		assert.Equal(t, software.Data.ID(), deferredResult.Meta["id"])

		// Execute the deferred operation
		execResult := deferredResult.Execute(context.Background())
		assert.NoError(t, execResult.Err)
		assert.Equal(t, software.Data.ID(), execResult.Data.ID())
	})

	t.Run("Update deferred", func(t *testing.T) {
		deferredResult := client.Repository.Software.Update(
			api.WithDeferredExecution(context.Background(), true),
			softwareitems.NewRef().ByName(softwareName),
			model.Software{
				Description: "Deferred update",
			},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
	})

	t.Run("Delete deferred", func(t *testing.T) {
		deferredResult := client.Repository.Software.Delete(
			api.WithDeferredExecution(context.Background(), true),
			softwareitems.NewRef().ByName(softwareName, softwareType),
			softwareitems.DeleteOptions{},
		)
		assert.NoError(t, deferredResult.Err)
		assert.True(t, deferredResult.IsDeferred())
		assert.NotEmpty(t, deferredResult.Meta["id"])
	})
}

func Test_SoftwareMetadata(t *testing.T) {
	client := testcore.CreateTestClient(t)
	softwareName := "software" + testingutils.RandomString(16)
	softwareType := "ci-artifact"

	// Create software item
	software := client.Repository.Software.Create(context.Background(), model.Software{
		Name:         softwareName,
		Type:         "c8y_Software",
		SoftwareType: softwareType,
		Description:  "Test software",
	})
	assert.NoError(t, software.Err)
	t.Cleanup(func() {
		client.ManagedObjects.Delete(context.Background(), software.Data.ID(), managedobjects.DeleteOptions{})
	})

	// Get with name resolver - check metadata
	result := client.Repository.Software.Get(
		context.Background(),
		softwareitems.NewRef().ByName(softwareName, softwareType),
		softwareitems.GetOptions{},
	)
	assert.NoError(t, result.Err)
	assert.Equal(t, "nameAndType", result.Meta["resolverType"])
	assert.Equal(t, softwareName, result.Meta["name"])
	assert.Equal(t, softwareType, result.Meta["softwareType"])
	assert.Equal(t, software.Data.ID(), result.Meta["id"])
	assert.Equal(t, "name:"+softwareName+":"+softwareType, result.Meta["identifier"])
}
