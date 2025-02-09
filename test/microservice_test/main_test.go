package microservice_test

import (
	"os"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
	"github.com/reubenmiller/go-c8y/pkg/microservice"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func TestMain(m *testing.M) {
	TestEnvironment = c8ytestutils.NewTestSetup()
	defer TestEnvironment.Cleanup()

	res := m.Run()
	os.Exit(res)
}

func bootstrapApplication(appName ...string) *microservice.Microservice {
	return TestEnvironment.BootstrapApplication(appName...)
}
