package c8y_api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ManagedObjectCreation(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	result := client.ManagedObjects.Create(context.Background(), map[string]any{
		"name": testingutils.RandomString(16),
	})
	assert.NoError(t, result.Err)
	assert.Equal(t, result.Data.Length(), 1)

	// decode to custom model
	mo, err := jsondoc.Decode[model.ManagedObject](result.Data.JSONDoc)
	assert.NoError(t, err)
	assert.Equal(t, mo.ID, result.Data.ID())

	// Delete object
	result = client.ManagedObjects.Delete(context.Background(), result.Data.ID(), managedobjects.DeleteOptions{})
	assert.NoError(t, result.Err)
}

func Test_ManagedObjectList(t *testing.T) {
	client := testcore.CreateTestClient(t)
	// client.Client.SetDebug(true)

	result := client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	assert.NoError(t, result.Err)
	assert.GreaterOrEqual(t, result.Data.Length(), 1)

	// iterate over the jsondocs
	for item := range result.Data.Iter() {
		fmt.Printf("id=%s, %s\n", item.Get("id").String(), item.Bytes())
	}

	// iterate over items but decode them into a custom object
	for item := range jsondoc.DecodeIter[model.ManagedObject](result.Data.Iter()) {
		fmt.Printf("id=%s, %s\n", item.ID, item.CreationTime)
	}

	// paginate over all items
	count := 0
	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type: "thin-edge.io",
	})
	for item := range it.Items() {
		fmt.Printf("count=%d | id=%s, type=%s\n", count, item.ID(), item.Type())
		count += 1
		if count > 3000 {
			break
		}
	}
	if err := it.Err(); err != nil {
		t.Fatalf("pagination error: %v", err)
	}

	// paginate over first N items
	count = 0
	it2 := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type: "thin-edge.io",
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 10,
		},
	})
	for item := range it2.Items() {
		fmt.Printf("count=%d | id=%s, type=%s\n", count, item.ID(), item.Type())
	}
	if err := it2.Err(); err != nil {
		t.Fatalf("pagination error: %v", err)
	}
}

func Test_ManagedObjectCRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create
	name := testingutils.RandomString(16)
	createResult := client.ManagedObjects.Create(ctx, map[string]any{
		"name":         name,
		"type":         "test_device",
		"c8y_IsDevice": map[string]any{},
	})
	assert.NoError(t, createResult.Err)
	assert.Equal(t, "Created", string(createResult.Status))
	assert.NotEmpty(t, createResult.Data.ID())
	assert.Equal(t, name, createResult.Data.Name())

	id := createResult.Data.ID()

	// Get
	getResult := client.ManagedObjects.Get(ctx, id, managedobjects.GetOptions{})
	assert.NoError(t, getResult.Err)
	assert.Equal(t, "OK", string(getResult.Status))
	assert.Equal(t, id, getResult.Data.ID())
	assert.Equal(t, name, getResult.Data.Name())

	// Update
	newName := testingutils.RandomString(16)
	updateResult := client.ManagedObjects.Update(ctx, id, map[string]any{
		"name": newName,
	})
	assert.NoError(t, updateResult.Err)
	assert.Equal(t, "OK", string(updateResult.Status))
	assert.Equal(t, id, updateResult.Data.ID())
	assert.Equal(t, newName, updateResult.Data.Name())

	// Delete
	deleteResult := client.ManagedObjects.Delete(ctx, id, managedobjects.DeleteOptions{})
	assert.NoError(t, deleteResult.Err)

	// Verify deleted
	getAfterDelete := client.ManagedObjects.Get(ctx, id, managedobjects.GetOptions{})
	assert.Error(t, getAfterDelete.Err)
	assert.True(t, c8y_api.ErrHasStatus(getAfterDelete.Err, 404))
}

func Test_ManagedObjectDeleteWithPrompt(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	ctx := context.Background()
	deferredCtx := c8y_api.WithDeferredExecution(ctx, true)
	req := client.ManagedObjects.Delete(deferredCtx, mo.Data.ID(), managedobjects.DeleteOptions{})

	if req.Request != nil {
		if c8y_api.IsDeferredExecution(req.Request.Context()) {
			fmt.Printf("Confirm deletion of %#v", req.Meta)
		}
	}

	result := req.Execute(ctx)
	assert.NoError(t, result.Err)
}

func Test_ManagedObjectGetByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	mo := testcore.CreateManagedObject(t, client)
	assert.NoError(t, mo.Err)

	ctx := context.Background()
	deferredCtx := c8y_api.WithDeferredExecution(ctx, true)
	name := mo.Data.Name()
	namePattern := name[0:len(name)-4] + "*"
	source := client.ManagedObjects.ByName(namePattern).String()
	req := client.ManagedObjects.Delete(deferredCtx, source, managedobjects.DeleteOptions{})

	if req.Request != nil {
		if c8y_api.IsDeferredExecution(req.Request.Context()) {
			fmt.Printf("Confirm deletion of %#v", req.Meta["name"])
		}
	}

	result := req.Execute(ctx)
	assert.NoError(t, result.Err)
}
