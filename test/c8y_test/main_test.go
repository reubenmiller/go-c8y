package c8y_test

import (
	"os"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func TestMain(m *testing.M) {

	TestEnvironment = c8ytestutils.NewTestSetup()
	defer TestEnvironment.Cleanup()

	res := m.Run()
	os.Exit(res)
}

func createTestClient() *c8y.Client {
	return TestEnvironment.NewClient()
}

func createRandomTestDevice(prefix ...string) (*c8y.ManagedObject, error) {
	return TestEnvironment.NewRandomTestDevice(prefix...)
}
