package devices

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/stretchr/testify/assert"
	"resty.dev/v3"
)

func newTestService() *Service {
	return NewService(&core.Service{
		Client: resty.New().SetBaseURL("https://example.cumulocity.com"),
	})
}

func TestCreateTokenB(t *testing.T) {
	t.Skip("Create a temp cert key pair")
	s := newTestService()
	s.Client.SetCertificateFromFile("todo.crt", "todo.key")
	req := s.createAccessTokenB()

	certChain := req.Request.Header.Get(types.HeaderSSLCertificateChain)
	assert.NotEmpty(t, certChain)
	assert.Equal(t, "example.cumulocity.com:8443", req.URL().Host)
	assert.Equal(t, ApiDeviceControlAccessToken, req.URL().Path)
}
