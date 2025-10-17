package testcore

import (
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func CreateTestClient(t *testing.T) *c8y_api.Client {
	t.Setenv("C8Y_TOKEN", "")
	return c8y_api.NewClientFromEnvironment(c8y_api.ClientOptions{})
}

func CreateRandomTestDevice(prefix ...string) (*c8y.ManagedObject, error) {
	return TestEnvironment.NewRandomTestDevice(prefix...)
}
