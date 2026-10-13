package api_test

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/destel/rill"
	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// createTestAgents creates n agents with a unique type and a thin-edge.io
// c8y_Agent fragment, so the custom-model tests don't depend on what data the
// tenant already holds. Returns the type to list by.
func createTestAgents(t *testing.T, client *api.Client, n int) string {
	t.Helper()
	agentType := "ci_agent_" + testingutils.RandomString(8)
	for i := range n {
		mo := client.ManagedObjects.Create(context.Background(), map[string]any{
			"name":      agentType + "_" + strconv.Itoa(i),
			"type":      agentType,
			"c8y_Agent": map[string]any{"name": "thin-edge.io", "version": "1.0.0"},
		})
		require.NoError(t, mo.Err)
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.Background(), mo.Data.ID(), managedobjects.DeleteOptions{})
		})
	}
	testcore.SleepIfLive(2 * time.Second)
	return agentType
}

func Test_ForEachCustomModel_Infallible(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// Create custom model which can also re-use fields from the default model
	type CustomModel struct {
		model.ManagedObject
		Agent map[string]string `json:"c8y_Agent,omitempty"`
	}

	agentType := createTestAgents(t, client, 3)

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type: agentType,
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 20,
			PageSize: 10,
		},
	})
	assert.NoError(t, it.Err())

	// Process the results (errors are skipped)
	matches := 0
	for item := range jsondoc.DecodeIter[CustomModel](it.Seq()) {
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

	agentType := createTestAgents(t, client, 3)

	it := client.ManagedObjects.ListAll(context.Background(), managedobjects.ListOptions{
		Type: agentType,
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 20,
			PageSize: 10,
		},
	})
	assert.NoError(t, it.Err())

	// Process the results
	matches := 0
	for item, err := range jsondoc.DecodeIterErr[CustomModel](it.Seq()) {
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
			MaxItems: 1000,
		},
	})
	assert.NoError(t, it.Err())

	assert.NoError(t, it.Preview())

	assert.GreaterOrEqual(t, it.TotalCount(), int64(1))

	// Apply client side filter
	matchingMos := rill.Filter(rill.FromSeq(it.Seq(), it.Err()), 1, func(mo jsonmodels.ManagedObject) (bool, error) {
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

	assert.GreaterOrEqual(t, matches.Load(), int64(0))
}
