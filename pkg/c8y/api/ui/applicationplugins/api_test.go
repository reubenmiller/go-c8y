package applicationplugins

import (
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/stretchr/testify/assert"
	"resty.dev/v3"
)

func newTestService() *Service {
	return NewService(&core.Service{
		Client: resty.New().SetBaseURL("https://example.cumulocity.com"),
	})
}

func TestGetApplicationB(t *testing.T) {
	s := newTestService()
	applicationID := "12345"

	req := s.getApplicationB(applicationID)

	assert.Equal(t, resty.MethodGet, req.Request.Method)
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Accept"))
	// Template path before resolution
	assert.Equal(t, ApiApplication, req.URL().Path)
}

func TestUpdateApplicationB(t *testing.T) {
	s := newTestService()
	applicationID := "12345"

	body := map[string]interface{}{
		"applicationBuilder": map[string]interface{}{
			"plugins": []interface{}{
				map[string]interface{}{"id": "plugin1"},
			},
		},
	}

	req := s.updateApplicationB(applicationID, body)

	assert.Equal(t, resty.MethodPut, req.Request.Method)
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Accept"))
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Content-Type"))
	assert.Equal(t, ApiApplication, req.URL().Path)
	assert.NotNil(t, req.Request.Body)
}

func TestInstallPluginFlow(t *testing.T) {
	// This tests the request structure for Install operation
	s := newTestService()
	applicationID := "app-123"

	// Test that getApplication creates correct request
	getReq := s.getApplicationB(applicationID)
	assert.Equal(t, resty.MethodGet, getReq.Request.Method)
	assert.Equal(t, ApiApplication, getReq.URL().Path)

	// Test that updateApplication would be called with correct structure
	body := map[string]interface{}{
		"applicationBuilder": map[string]interface{}{
			"plugins": []interface{}{
				map[string]interface{}{"id": "new-plugin-id"},
			},
		},
	}
	updateReq := s.updateApplicationB(applicationID, body)
	assert.Equal(t, resty.MethodPut, updateReq.Request.Method)
	assert.NotNil(t, updateReq.Request.Body)
}

func TestUpdatePluginReferencesFlow(t *testing.T) {
	s := newTestService()
	applicationID := "app-123"

	pluginRefs := []PluginReference{
		{ID: "plugin1", Version: "1.0.0"},
		{ID: "plugin2", Name: "My Plugin", Version: "2.0.0"},
	}

	// Create the body structure that Update would use
	plugins := make([]map[string]interface{}, len(pluginRefs))
	for i, ref := range pluginRefs {
		plugin := map[string]interface{}{}
		if ref.ID != "" {
			plugin["id"] = ref.ID
		}
		if ref.Name != "" {
			plugin["name"] = ref.Name
		}
		if ref.Version != "" {
			plugin["version"] = ref.Version
		}
		plugins[i] = plugin
	}

	body := map[string]interface{}{
		"applicationBuilder": map[string]interface{}{
			"plugins": plugins,
		},
	}

	req := s.updateApplicationB(applicationID, body)

	assert.Equal(t, resty.MethodPut, req.Request.Method)
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Content-Type"))
	assert.Equal(t, ApiApplication, req.URL().Path)
}

func TestDeletePluginFlow(t *testing.T) {
	s := newTestService()
	applicationID := "app-123"

	// Test that getApplication would be called first
	getReq := s.getApplicationB(applicationID)
	assert.Equal(t, resty.MethodGet, getReq.Request.Method)

	// Test that updateApplication would be called with filtered plugins
	body := map[string]interface{}{
		"applicationBuilder": map[string]interface{}{
			"plugins": []interface{}{
				// Plugin to delete would be filtered out
				map[string]interface{}{"id": "remaining-plugin"},
			},
		},
	}
	updateReq := s.updateApplicationB(applicationID, body)
	assert.Equal(t, resty.MethodPut, updateReq.Request.Method)
}

func TestPluginReferenceStructure(t *testing.T) {
	ref := PluginReference{
		ID:      "plugin-123",
		Name:    "My Plugin",
		Version: "1.0.0",
	}

	assert.Equal(t, "plugin-123", ref.ID)
	assert.Equal(t, "My Plugin", ref.Name)
	assert.Equal(t, "1.0.0", ref.Version)
}

func TestEmptyPluginReference(t *testing.T) {
	ref := PluginReference{}

	assert.Empty(t, ref.ID)
	assert.Empty(t, ref.Name)
	assert.Empty(t, ref.Version)
}

func TestMultiplePluginReferences(t *testing.T) {
	refs := []PluginReference{
		{ID: "plugin1"},
		{ID: "plugin2", Version: "1.0.0"},
		{ID: "plugin3", Name: "Third Plugin", Version: "2.0.0"},
	}

	assert.Len(t, refs, 3)
	assert.Equal(t, "plugin1", refs[0].ID)
	assert.Empty(t, refs[0].Version)
	assert.Equal(t, "1.0.0", refs[1].Version)
	assert.Equal(t, "Third Plugin", refs[2].Name)
}

func TestApiApplicationPath(t *testing.T) {
	// Verify the API path constant is correct
	assert.Equal(t, "/application/applications/{id}", ApiApplication)
}

func TestParamIdConstant(t *testing.T) {
	// Verify the parameter name constant
	assert.Equal(t, "id", ParamId)
}

func TestUpdateApplicationBodyStructure(t *testing.T) {
	s := newTestService()
	applicationID := "app-123"

	// Test various body structures
	testCases := []struct {
		name string
		body map[string]interface{}
	}{
		{
			name: "Single plugin",
			body: map[string]interface{}{
				"applicationBuilder": map[string]interface{}{
					"plugins": []interface{}{
						map[string]interface{}{"id": "plugin1"},
					},
				},
			},
		},
		{
			name: "Multiple plugins",
			body: map[string]interface{}{
				"applicationBuilder": map[string]interface{}{
					"plugins": []interface{}{
						map[string]interface{}{"id": "plugin1"},
						map[string]interface{}{"id": "plugin2", "version": "1.0.0"},
					},
				},
			},
		},
		{
			name: "No plugins",
			body: map[string]interface{}{
				"applicationBuilder": map[string]interface{}{
					"plugins": []interface{}{},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := s.updateApplicationB(applicationID, tc.body)
			assert.Equal(t, resty.MethodPut, req.Request.Method)
			assert.NotNil(t, req.Request.Body)
		})
	}
}

func TestApplicationPluginsUsesCorrectEndpoint(t *testing.T) {
	// Verify that application plugins operations use the application endpoint,
	// not a dedicated /plugins endpoint
	s := newTestService()
	applicationID := "app-123"

	getReq := s.getApplicationB(applicationID)
	updateReq := s.updateApplicationB(applicationID, map[string]interface{}{})

	// Both operations should use the same application endpoint template
	assert.Equal(t, ApiApplication, getReq.URL().Path, "Get should use application endpoint")
	assert.Equal(t, ApiApplication, updateReq.URL().Path, "Update should use application endpoint")

	// Verify it's NOT using a /plugins subpath
	assert.NotContains(t, getReq.URL().Path, "/plugins")
	assert.NotContains(t, updateReq.URL().Path, "/plugins")
}
