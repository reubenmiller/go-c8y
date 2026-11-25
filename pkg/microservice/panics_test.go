package microservice

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/stretchr/testify/assert"
)

func Test_SetMicroserviceHost_InvalidReturnsError(t *testing.T) {
	m := &Microservice{MicroserviceHost: "http://ok"}
	assert.Error(t, m.SetMicroserviceHostE("http://exa mple.com"))
	assert.Equal(t, "http://ok", m.MicroserviceHost)

	// deprecated form no longer panics
	assert.NotPanics(t, func() { m.SetMicroserviceHost("http://exa mple.com") })
	assert.Equal(t, "http://ok", m.MicroserviceHost)

	assert.NoError(t, m.SetMicroserviceHostE("http://other:8080"))
	assert.Equal(t, "http://other:8080", m.MicroserviceHost)
}

func Test_WithServiceUserCredentials_ExtraArgsDoNotPanic(t *testing.T) {
	m := &Microservice{ServiceUsers: []model.ServiceUser{{Tenant: "a", Username: "ua"}, {Tenant: "b", Username: "ub"}}}
	assert.NotPanics(t, func() {
		assert.Equal(t, "ub", m.WithServiceUserCredentials("b", "c").Username)
	})
}
