package testcore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8ytestutils"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

var TestEnvironment *c8ytestutils.SetupConfiguration

func CreateTestClient(t *testing.T) *c8y_api.Client {
	t.Setenv("C8Y_TOKEN", "")
	client := c8y_api.NewClientFromEnvironment(c8y_api.ClientOptions{})
	client.Client.SetRetryCount(1)
	return client
}

func CreateTestClientNoAuth(t *testing.T) *c8y_api.Client {
	envvars := make([]string, 0)
	envvars = append(envvars, authentication.EnvironmentToken...)
	envvars = append(envvars, authentication.EnvironmentUsername...)
	envvars = append(envvars, authentication.EnvironmentPassword...)
	for _, name := range envvars {
		t.Setenv(name, "")
	}

	return c8y_api.NewClientFromEnvironment(c8y_api.ClientOptions{})
}

func CreateTestClientWithToken(t *testing.T) *c8y_api.Client {
	return c8y_api.NewClientFromEnvironment(c8y_api.ClientOptions{})
}

func CreateRandomTestDevice(prefix ...string) (*c8y.ManagedObject, error) {
	return TestEnvironment.NewRandomTestDevice(prefix...)
}

func NewService() *core.Service {
	return &core.Service{
		Client: resty.New().SetBaseURL("https://example.cumulocity.com"),
	}
}

func CreateManagedObject(t *testing.T, client *c8y_api.Client) op.Result[jsonmodels.ManagedObject] {
	mo := client.ManagedObjects.Create(context.TODO(), map[string]any{
		"name": "ci_" + testingutils.RandomString(16),
	})
	if !mo.IsError() {
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.TODO(), mo.Data.ID(), managedobjects.DeleteOptions{})
		})
	}
	return mo
}

// CreateDevice creates a new test device in Cumulocity IoT and registers a cleanup function
// to delete the device after the test completes.
func CreateDevice(t *testing.T, client *c8y_api.Client) op.Result[jsonmodels.ManagedObject] {
	mo := client.Devices.Create(context.TODO(), jsonmodels.NewDevice("ci_"+testingutils.RandomString(16)))
	if !mo.IsError() {
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.TODO(), mo.Data.ID(), managedobjects.DeleteOptions{
				Cascade: true,
			})
		})
	}
	require.NoError(t, mo.Err)
	return mo
}

func CreateEvent(t *testing.T, client *c8y_api.Client, mo *jsonmodels.ManagedObject) op.Result[jsonmodels.Event] {
	return client.Events.Create(context.TODO(), model.Event{
		Source: model.NewSource(mo.ID()),
		Type:   "ci_" + testingutils.RandomString(10),
		Text:   "Test event",
		Time:   time.Now(),
	})
}

// NewDummyFile creates a temporary test file with the given name and contents.
// The file will be created in the current directory and it's the caller's
// responsibility to clean it up.
func NewDummyFile(t *testing.T, name string, contents string) (createFilePath string) {
	if name == "" {
		name = "test-dummy-dummy"
	}
	fullPath := filepath.Join(t.TempDir(), name)
	f, err := os.Create(fullPath)
	if err != nil {
		panic(errors.Wrap(err, "Error creating dummy file"))
	}

	defer f.Close()

	f.WriteString(contents)

	if err := f.Sync(); err != nil {
		panic(errors.Wrap(err, "Failed to fill file with dummy information"))
	}

	createFilePath = f.Name()
	return
}

// NewDummyFileWithSize creates a temporary test file with the given name and size.
// The file will be created in the current directory and it's the caller's
// responsibility to clean it up.
func NewDummyFileWithSize(name string, size int64) (filepath string) {
	if name == "" {
		name = "test-dummy-dummy"
	}

	if size < 0 {
		size = 10_000_000
	}

	f, err := os.Create(name)
	if err != nil {
		panic(errors.Wrap(err, "Error creating dummy file"))
	}

	defer f.Close()

	if err := f.Truncate(size); err != nil {
		panic(errors.Wrap(err, "Failed to fill file with dummy information"))
	}

	if err := f.Sync(); err != nil {
		panic(errors.Wrap(err, "Failed to sync file with dummy information"))
	}

	filepath = f.Name()
	return
}
