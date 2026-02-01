package c8y_api_test

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/destel/rill"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_ForEachManagedObjectsIncludeAll(t *testing.T) {
	client := testcore.CreateTestClient(t)

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 5000,
		},
	})
	assert.NoError(t, it.Err())
	count := 0
	for item := range it.Items() {
		if count > 2001 {
			break
		}
		slog.Info("Processing message", "id", item.ID())
		count += 1
	}
}

func Test_ForEachManagedObjectsMaxPages(t *testing.T) {
	client := testcore.CreateTestClient(t)

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type: "thin-edge.io",
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 20,
			PageSize: 10,
		},
	})
	assert.NoError(t, it.Err())

	// Process the results
	total := 0
	for item := range it.Items() {
		slog.Info("Processing message", "id", item.ID)
		total++
	}
	assert.Greater(t, total, 0)
}

func Test_ForEachCustomModel_Infallable(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create custom model which can also re-use fields from the default model
	type CustomModel struct {
		model.ManagedObject
		Agent map[string]string `json:"c8y_Agent,omitempty"`
	}

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type: "thin-edge.io",
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 20,
			PageSize: 10,
		},
	})
	assert.NoError(t, it.Err())

	// Process the results (errors are skipped)
	matches := 0
	for item := range jsondoc.DecodeIter[CustomModel](it.Items()) {
		if v, ok := item.Agent["name"]; ok {
			if v == "thin-edge.io" {
				matches += 1
			}
		}
	}
	assert.GreaterOrEqual(t, matches, 1)
}

func Test_ForEachCustomModel_Fallable(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create custom model which can also re-use fields from the default model
	type CustomModel struct {
		model.ManagedObject
		Agent map[string]string `json:"c8y_Agent,omitempty"`
	}

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type: "thin-edge.io",
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 20,
			PageSize: 10,
		},
	})
	assert.NoError(t, it.Err())

	// Process the results
	matches := 0
	for item, err := range jsondoc.DecodeIterErr[CustomModel](it.Items()) {
		if err != nil {
			continue
		}
		if v, ok := item.Agent["name"]; ok {
			if v == "thin-edge.io" {
				matches += 1
			}
		}
	}
	assert.GreaterOrEqual(t, matches, 1)
}

// Filter managed object list on the client side
func Test_ManagedObjectsAdvanced(t *testing.T) {
	client := testcore.CreateTestClient(t)

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 100,
		},
	})
	assert.NoError(t, it.Err())

	// Apply client side filter
	matchingMos := rill.Filter(rill.FromSeq(it.Items(), it.Err()), 1, func(mo jsonmodels.ManagedObject) (bool, error) {
		return strings.HasPrefix(mo.Name(), "TestDevice"), nil
	})

	// Apply batching
	batches := rill.Batch(matchingMos, 10, 2*time.Second)

	// Process the results
	matches := atomic.Int64{}
	rill.ForEach(batches, 2, func(mos []jsonmodels.ManagedObject) error {
		slog.Info("Processing batch", "size", len(mos))
		matches.Add(int64(len(mos)))
		return nil
	})

	assert.GreaterOrEqual(t, matches.Load(), int64(1))
}
