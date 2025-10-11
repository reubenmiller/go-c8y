package testcore

import (
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func CreateTestClient() *c8y_api.Client {
	return c8y_api.NewClient(c8y_api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})
}

func CreateRandomTestDevice(prefix ...string) (*c8y.ManagedObject, error) {
	return TestEnvironment.NewRandomTestDevice(prefix...)
}
