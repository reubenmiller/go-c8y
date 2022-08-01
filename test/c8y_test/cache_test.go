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
)

func createCachedClient(keys []string) *c8y.Client {
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
	httpClient := c8y.NewCachedClient(c8y.NewHTTPClient(
		c8y.WithInsecureSkipVerify(false),
	), cacheDir, 100*time.Second, isCacheableRequest, c8y.CacheOptions{
		BodyKeys: keys,
	})
	client := c8y.NewClientFromEnvironment(httpClient, false)
	return client
}

func Test_CachedClientWithMissingBodyKeys(t *testing.T) {

	parameters := []struct {
		Keys  []string
		Body1 map[string]interface{}
		Body2 map[string]interface{}
	}{
		{
			[]string{"name", "complex.arrays.#"},

			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item3", "item4", "item5"},
				},
			},
		},
		{
			[]string{"name", "complex.arrays"},

			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item1", "item2", "item3"},
				},
			},
		},
		{
			[]string{"name", "index"},
			map[string]interface{}{
				"name":  "test_device_100",
				"other": false,
				"index": 101,
			},
			map[string]interface{}{
				"name":  "test_device_100",
				"other": true,
				"index": 102,
			},
		},
		{
			[]string{"name", "index"},
			map[string]interface{}{
				"name":  "test_device_100",
				"other": false,
				"index": 101,
			},
			map[string]interface{}{
				"name":  "test_device_100",
				"other": true,
			},
		},
	}

	for _, params := range parameters {
		client := createCachedClient(params.Keys)

		_, resp1, err := client.Inventory.Create(context.Background(), params.Body1)
		if err != nil {
			t.Error(err)
		}
		defer client.Inventory.Delete(context.Background(), resp1.JSON("id").String())

		_, resp2, err := client.Inventory.Create(context.Background(), params.Body2)
		if err != nil {
			t.Error(err)
		}
		defer client.Inventory.Delete(context.Background(), resp2.JSON("id").String())

		if resp2.JSON("id").String() == resp1.JSON("id").String() {
			t.Errorf("Expected customDate to match. wanted: %s, got: %s", resp1.JSON(), resp2.JSON())
		}
	}
}

func Test_CachedClientWithSelectKeys(t *testing.T) {

	parameters := []struct {
		Keys  []string
		Body1 map[string]interface{}
		Body2 map[string]interface{}
	}{
		{
			[]string{"name", "complex.arrays.#"},

			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item3", "item4"},
				},
			},
		},
		{
			[]string{"name", "complex.arrays"},

			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item1", "item2"},
				},
			},
			map[string]interface{}{
				"name": "test_device_100",
				"complex": map[string]interface{}{
					"arrays": []string{"item1", "item2"},
				},
			},
		},
		{
			[]string{"name", "index"},
			map[string]interface{}{
				"name":  "test_device_100",
				"other": false,
				"index": 101,
			},
			map[string]interface{}{
				"name":  "test_device_100",
				"other": true,
				"index": 101,
			},
		},
	}

	for _, params := range parameters {
		client := createCachedClient(params.Keys)

		_, resp1, err := client.Inventory.Create(context.Background(), params.Body1)
		if err != nil {
			t.Error(err)
		}
		defer client.Inventory.Delete(context.Background(), resp1.JSON("id").String())

		_, resp2, err := client.Inventory.Create(context.Background(), params.Body2)
		if err != nil {
			t.Error(err)
		}
		defer client.Inventory.Delete(context.Background(), resp2.JSON("id").String())

		if resp2.JSON("id").String() != resp1.JSON("id").String() {
			t.Errorf("Expected customDate to match. wanted: %s, got: %s", resp1.JSON(), resp2.JSON())
		}
	}
}
