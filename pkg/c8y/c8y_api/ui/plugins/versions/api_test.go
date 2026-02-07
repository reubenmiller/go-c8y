package versions

import (
	"strings"
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
	req := s.listB("12345", ListOptions{})

	assert.Equal(t, resty.MethodGet, req.Request.Method)
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Accept"))
	assert.Equal(t, ApiVersions, req.URL().Path)
}

func TestGetB(t *testing.T) {
	s := newTestService()
	req := s.getB("12345", "1.0.0")

	assert.Equal(t, resty.MethodGet, req.Request.Method)
	assert.Equal(t, ApiVersionByName, req.URL().Path)
}

func TestCreateB(t *testing.T) {
	s := newTestService()
	opts := CreateOptions{
		Version:  "1.0.0",
		Reader:   strings.NewReader("content"),
		Filename: "plugin.zip",
	}
	req := s.createB("12345", opts)

	assert.Equal(t, resty.MethodPost, req.Request.Method)
	assert.Equal(t, ApiVersions, req.URL().Path)
}

func TestUpdateB(t *testing.T) {
	s := newTestService()
	req := s.updateB("12345", "1.0.0", []string{"latest"})

	assert.Equal(t, resty.MethodPut, req.Request.Method)
	assert.Equal(t, types.MimeTypeApplicationJSON, req.Request.Header.Get("Content-Type"))
	assert.Equal(t, ApiVersionByID, req.URL().Path)
}

func TestDeleteB(t *testing.T) {
	s := newTestService()
	req := s.deleteB("12345", "1.0.0")

	assert.Equal(t, resty.MethodDelete, req.Request.Method)
	assert.Equal(t, ApiVersionByName, req.URL().Path)
}
