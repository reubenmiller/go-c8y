package c8y_test

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	TestEnvironment = c8ytestutils.NewTestSetup()
	defer TestEnvironment.Cleanup()

	res := m.Run()
	os.Exit(res)
}

func createTestClient() *c8y.Client {
	return TestEnvironment.NewClient()
}

func createRandomTestDevice(prefix ...string) (*c8y.ManagedObject, error) {
	return TestEnvironment.NewRandomTestDevice()
}
