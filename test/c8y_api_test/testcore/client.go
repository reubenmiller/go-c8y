package testcore

import (
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func CreateTestClient() *c8y_api.Client {
	return c8y_api.NewClientV2(c8y_api.ClientOptions{
		BaseURL:  os.Getenv("C8Y_HOST"),
		Username: os.Getenv("C8Y_USERNAME"),
		Password: os.Getenv("C8Y_PASSWORD"),
	})
}

func CreateRandomTestDevice(prefix ...string) (*c8y.ManagedObject, error) {
	return TestEnvironment.NewRandomTestDevice(prefix...)
}
