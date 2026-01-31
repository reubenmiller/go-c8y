package c8y_api_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ManagedObjectCreation(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	result := client.ManagedObjects.Create2(context.Background(), map[string]any{
		"name": testingutils.RandomString(16),
	})
	assert.NoError(t, result.Err)
	assert.Equal(t, result.Data.Length(), 1)

	// decode to custom model
	mo, err := jsondoc.Decode[model.ManagedObject](result.Data.JSONDoc)
	assert.NoError(t, err)
	assert.Equal(t, mo.ID, result.Data.ID())

	// Delete object
	err = client.ManagedObjects.Delete(context.Background(), result.Data.ID(), managedobjects.DeleteOptions{})
	assert.NoError(t, err)
}

func Test_ManagedObjectList(t *testing.T) {
	client := testcore.CreateTestClient(t)
	// client.Client.SetDebug(true)

	result := client.ManagedObjects.List2(context.Background(), managedobjects.ListOptions{})
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
	it2 := client.ManagedObjects.ListLimit(context.Background(), managedobjects.ListOptions{
		Type: "thin-edge.io",
	}, 10)
	for item := range it2.Items() {
		fmt.Printf("count=%d | id=%s, type=%s\n", count, item.ID(), item.Type())
	}
	if err := it2.Err(); err != nil {
		t.Fatalf("pagination error: %v", err)
	}
}
