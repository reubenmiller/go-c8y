package c8y_api_test

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/destel/rill"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_NewPagerManagedObjectsIncludeAll(t *testing.T) {
	client := testcore.CreateTestClient()

	// Create pager
	pager := pagination.NewPager[model.ManagedObject](
		client.ManagedObjects.ListPager(context.Background(), managedobjects.ListOptions{}),
	)

	// Start pager and iterate through all of the results
	go pager.IncludeAll()

	// Range over the output channel
	for item := range pager.Output {
		slog.Info("Processing message", "id", item.ID)
	}
}

func Test_NewPagerManagedObjectsPages(t *testing.T) {
	client := testcore.CreateTestClient()

	// Create pager which includes the creation of the channel
	pager := pagination.NewPager[model.ManagedObject](
		client.ManagedObjects.ListPager(context.Background(), managedobjects.ListOptions{}),
	)

	// Start pager and iterate through the first 2 pages
	go pager.Pages(pagination.PagerOptions{
		MaxPages: 2,
		PageSize: 10,
	})

	// Range over the output channel
	total := 0
	for item := range pager.Output {
		slog.Info("Processing message", "id", item.ID)
		total++
	}
	assert.Equal(t, total, 20)
}

func Test_ForEachManagedObjectsIncludeAll(t *testing.T) {
	client := testcore.CreateTestClient()

	out := make(chan model.ManagedObject)

	// Start pager and iterate through all of the results
	go pagination.ForEach(
		client.ManagedObjects.ListPager(context.Background(), managedobjects.ListOptions{}),
		pagination.IncludeAll(),
		out,
	)

	// Range over the output channel
	for item := range out {
		slog.Info("Processing message", "id", item.ID)
	}
}

func Test_ForEachManagedObjectsMaxPages(t *testing.T) {
	client := testcore.CreateTestClient()

	// Create output channel standard managed object
	out := make(chan model.ManagedObject)

	// Create list pager and iterate over the results
	go pagination.ForEach(
		client.ManagedObjects.ListPager(context.Background(), managedobjects.ListOptions{}),
		pagination.PagerOptions{
			MaxPages: 2,
			PageSize: 10,
		},
		out,
	)

	// Process the results
	total := 0
	for item := range out {
		slog.Info("Processing message", "id", item.ID)
		total++
	}
	assert.Equal(t, total, 20)
}

func Test_ForEachCustomModel(t *testing.T) {
	client := testcore.CreateTestClient()

	// Create custom model which can also re-use fields from the default model
	type CustomModel struct {
		model.ManagedObject
		Agent map[string]string `json:"c8y_Agent,omitempty"`
	}

	// Create output channel with custom data model
	out := make(chan CustomModel)

	// Create list pager and iterate over the results
	go pagination.ForEach(
		client.ManagedObjects.ListPager(context.Background(), managedobjects.ListOptions{
			Type: "thin-edge.io",
		}),
		pagination.PagerOptions{
			MaxPages: 2,
			PageSize: 10,
		},
		out,
	)

	// Process the results
	matches := 0
	for item := range out {
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
	client := testcore.CreateTestClient()

	// Create channel with desired data model
	out := make(chan model.ManagedObject)

	// List managed objects and iterate of the list
	go pagination.ForEach(
		client.ManagedObjects.ListPager(context.Background(), managedobjects.ListOptions{}),
		pagination.PagerOptions{
			MaxPages: 5,
			PageSize: 2000,
		},
		out,
	)

	// Apply client side filter
	matchingMos := rill.Filter(rill.FromChan(out, nil), 1, func(mo model.ManagedObject) (bool, error) {
		return strings.HasPrefix(mo.Name, "TestDevice"), nil
	})

	// Apply batching
	batches := rill.Batch(matchingMos, 10, 2*time.Second)

	// Process the results
	matches := atomic.Int64{}
	rill.ForEach(batches, 2, func(mos []model.ManagedObject) error {
		slog.Info("Processing batch", "size", len(mos))
		matches.Add(int64(len(mos)))
		return nil
	})

	assert.GreaterOrEqual(t, matches.Load(), int64(1))
}
