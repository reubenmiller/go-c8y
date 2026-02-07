package plugins

import (
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/stretchr/testify/assert"
	"resty.dev/v3"
)

func newTestService() *Service {
	return NewService(&core.Service{
		Client: resty.New().SetBaseURL("https://example.cumulocity.com"),
	})
}

func TestListB(t *testing.T) {
	s := newTestService()
	req := s.listB(ListOptions{Name: "my-plugin"})

	assert.Equal(t, resty.MethodGet, req.Request.Method)
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Accept"))
	assert.Equal(t, ApiPlugins, req.URL().Path)
}

func TestGetB(t *testing.T) {
	s := newTestService()
	req := s.getB("12345")

	assert.Equal(t, resty.MethodGet, req.Request.Method)
	assert.Equal(t, ApiPlugin, req.URL().Path)
}

func TestCreateB(t *testing.T) {
	s := newTestService()
	plugin := NewPlugin("test-plugin")
	req := s.createB(plugin)

	assert.Equal(t, resty.MethodPost, req.Request.Method)
	assert.Equal(t, ApiPlugins, req.URL().Path)
}

func TestUpdateB(t *testing.T) {
	s := newTestService()
	plugin := NewPlugin("test")
	req := s.updateB("12345", plugin)

	assert.Equal(t, resty.MethodPut, req.Request.Method)
	assert.Equal(t, ApiPlugin, req.URL().Path)
}

func TestDeleteB(t *testing.T) {
	s := newTestService()
	req := s.deleteB("12345")

	assert.Equal(t, resty.MethodDelete, req.Request.Method)
	assert.Equal(t, ApiPlugin, req.URL().Path)
}

func TestNewPlugin(t *testing.T) {
	plugin := NewPlugin("my-plugin")

	assert.Equal(t, "my-plugin", plugin.Name)
	assert.Equal(t, "my-plugin-key", plugin.Key)
	assert.Equal(t, ApplicationTypeHosted, plugin.Type)
	assert.NotNil(t, plugin.Manifest)
}
