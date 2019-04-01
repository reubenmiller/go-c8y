package microservice_test

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
	"github.com/reubenmiller/go-c8y/pkg/microservice"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	TestEnvironment = c8ytestutils.NewTestSetup()
	defer TestEnvironment.Cleanup()

	res := m.Run()
	os.Exit(res)
}

func bootstrapApplication(appName ...string) *microservice.Microservice {
	return TestEnvironment.BootstrapApplication()
}
