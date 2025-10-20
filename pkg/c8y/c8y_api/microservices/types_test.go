package microservices

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_NewManifestFromJSON(t *testing.T) {
	dummyManifest := `
{
  "apiVersion": "v2",
  "name": "example",
  "version": "1.0.0-SNAPSHOT",
  "provider": {
    "name": "New Company Ltd.",
    "domain": "http://new-company.com",
    "support": "support@new-company.com"
  },
  "isolation": "PER_TENANT",
  "requiredRoles": [
    "ROLE_NOTIFICATION_2_ADMIN"
  ],
  "replicas": 1,
  "resources": {
    "cpu": "0.5",
    "memory": "256Mi"
  },
  "livenessProbe": {
    "httpGet": {
      "path": "/health"
    },
    "initialDelaySeconds": 60,
    "periodSeconds": 10
  },
  "readinessProbe": {
    "httpGet": {
      "path": "/health",
      "port": 80
    },
    "initialDelaySeconds": 20,
    "periodSeconds": 10
  }
}
	`
	manifest, err := NewManifest(nil, FromJSON(bytes.NewBufferString(dummyManifest)))
	assert.NoError(t, err)
	assert.Equal(t, APIVersion2, manifest.APIVersion)
}
