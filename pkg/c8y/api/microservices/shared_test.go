package microservices

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/stretchr/testify/assert"
)

// NewServiceWithApplications shares the given applications service, rather than
// holding a copy of it.
func TestNewServiceWithApplications_SharesInstance(t *testing.T) {
	common := &httpcore.Service{}
	apps := applications.NewService(common)
	svc := NewServiceWithApplications(common, apps)

	assert.Same(t, apps, svc.applicationAPI)
	assert.NotNil(t, svc.Current)
}
