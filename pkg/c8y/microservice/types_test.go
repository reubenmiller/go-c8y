package microservice

import (
	"encoding/json"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
)

func TestManifestParsing(t *testing.T) {
	contents := `
{
    "apiVersion": "v2",
    "name": "my-microservice",
    "version": "1.0.0",
    "provider": {
        "name": "New Company Ltd.",
        "domain": "https://new-company.com",
        "support": "support@new-company.com"
    },
    "isolation": "MULTI_TENANT",
    "scale": "AUTO",
    "replicas": 2,
    "resources": {
        "cpu": "1",
        "memory": "1G"
    },
    "requestedResources":{
            "cpu": "100m",
            "memory": "128Mi"
    },
    "requiredRoles": [
        "ROLE_ALARM_READ"
    ],
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
    },
    "settingsCategory": "myms",
    "settings": [
        {
            "key": "tracker-id",
            "defaultValue": "1234"
        }
    ]
}
`
	manifest := new(Manifest)
	err := json.Unmarshal([]byte(contents), &manifest)
	testingutils.Ok(t, err)
	testingutils.Equals(t, "my-microservice", manifest.Name)
	testingutils.Equals(t, "1.0.0", manifest.Version)
	testingutils.Equals(t, APIVersion2, manifest.APIVersion)
	testingutils.Equals(t, IsolationMultiTenant, manifest.Isolation)
	testingutils.Equals(t, 2, manifest.Replicas)
	testingutils.Equals(t, "1", manifest.Resources.CPU)
	testingutils.Equals(t, "1G", manifest.Resources.Memory)
}
