package c8y_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createCachedClient(keys []string) *api.Client {
	isCacheableRequest := func(req *http.Request) bool {
		if strings.EqualFold(req.Method, "GET") || strings.EqualFold(req.Method, "HEAD") {
			return true
		}

		if strings.EqualFold(req.Method, "POST") || strings.EqualFold(req.Method, "PUT") {
			return true
		}

		return false
	}

	cacheDir := filepath.Join(os.TempDir(), "go-c8y-cache")
	transport := c8y.NewCachedTransport(nil, 100*time.Second, cacheDir, isCacheableRequest, c8y.CacheOptions{
		BodyKeys: keys,
	})
	return api.NewClientFromEnvironment(api.ClientOptions{
		Transport: transport,
	})
}

func Test_CachedClientWithMissingBodyKeys(t *testing.T) {

	parameters := []struct {
		Keys  []string
		Body1 map[string]any
		Body2 map[string]any
	}{
		{
			[]string{"name", "complex.arrays.#"},

			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item3", "item4", "item5"},
				},
			},
		},
		{
			[]string{"name", "complex.arrays"},

			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item1", "item2", "item3"},
				},
			},
		},
		{
			[]string{"name", "index"},
			map[string]any{
				"name":  "test_device_100",
				"other": false,
				"index": 101,
			},
			map[string]any{
				"name":  "test_device_100",
				"other": true,
				"index": 102,
			},
		},
		{
			[]string{"name", "index"},
			map[string]any{
				"name":  "test_device_100",
				"other": false,
				"index": 101,
			},
			map[string]any{
				"name":  "test_device_100",
				"other": true,
			},
		},
	}

	for _, params := range parameters {
		client := createCachedClient(params.Keys)

		result1 := client.ManagedObjects.Create(context.Background(), params.Body1)
		require.NoError(t, result1.Err)
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.Background(), result1.Data.ID(), managedobjects.DeleteOptions{})
		})

		result2 := client.ManagedObjects.Create(context.Background(), params.Body2)
		require.NoError(t, result2.Err)
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.Background(), result2.Data.ID(), managedobjects.DeleteOptions{})
		})

		assert.NotEqual(t, result1.Data.ID(), result2.Data.ID(), "Expected IDs to differ. body1: %s, body2: %s", result1.Data.JSONDoc, result2.Data.JSONDoc)
	}
}

func Test_CachedClientWithSelectKeys(t *testing.T) {

	parameters := []struct {
		Keys  []string
		Body1 map[string]any
		Body2 map[string]any
	}{
		{
			[]string{"name", "complex.arrays.#"},

			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item3", "item4"},
				},
			},
		},
		{
			[]string{"name", "complex.arrays"},

			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]any{
				"name": "test_device_100",
				"complex": map[string]any{
					"arrays": []string{"item1", "item2"},
				},
			},
		},
		{
			[]string{"name", "index"},
			map[string]any{
				"name":  "test_device_100",
				"other": false,
				"index": 101,
			},
			map[string]any{
				"name":  "test_device_100",
				"other": true,
				"index": 101,
			},
		},
	}

	for _, params := range parameters {
		client := createCachedClient(params.Keys)

		result1 := client.ManagedObjects.Create(context.Background(), params.Body1)
		require.NoError(t, result1.Err)
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.Background(), result1.Data.ID(), managedobjects.DeleteOptions{})
		})

		result2 := client.ManagedObjects.Create(context.Background(), params.Body2)
		require.NoError(t, result2.Err)
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.Background(), result2.Data.ID(), managedobjects.DeleteOptions{})
		})

		assert.Equal(t, result1.Data.ID(), result2.Data.ID(), "Expected IDs to match. body1: %s, body2: %s", result1.Data.JSONDoc, result2.Data.JSONDoc)
	}
}
