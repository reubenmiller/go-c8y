package testcore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

func CreateTestClient(t *testing.T) *api.Client {
	t.Setenv("C8Y_TOKEN", "")
	client := api.NewClientFromEnvironment(api.ClientOptions{})
	client.Client.SetRetryCount(1)
	return client
}

func CreateTestClientNoAuth(t *testing.T) *api.Client {
	envvars := make([]string, 0)
	envvars = append(envvars, authentication.EnvironmentToken...)
	envvars = append(envvars, authentication.EnvironmentUsername...)
	envvars = append(envvars, authentication.EnvironmentPassword...)
	for _, name := range envvars {
		t.Setenv(name, "")
	}

	return api.NewClientFromEnvironment(api.ClientOptions{})
}

func CreateTestClientWithToken(t *testing.T) *api.Client {
	return api.NewClientFromEnvironment(api.ClientOptions{})
}

func NewService() *core.Service {
	return &core.Service{
		Client: resty.New().SetBaseURL("https://example.cumulocity.com"),
	}
}

func CreateManagedObject(t *testing.T, client *api.Client) op.Result[jsonmodels.ManagedObject] {
	mo := client.ManagedObjects.Create(context.TODO(), map[string]any{
		"name": "ci_" + testingutils.RandomString(16),
	})
	if !mo.IsError() {
		t.Cleanup(func() {
			client.ManagedObjects.Delete(context.TODO(), mo.Data.ID(), managedobjects.DeleteOptions{})
		})
	}
	require.NoError(t, mo.Err)
	return mo
}

// CreateDevice creates a new test device in Cumulocity IoT and registers a cleanup function
// to delete the device after the test completes.
func CreateDevice(t *testing.T, client *api.Client) op.Result[jsonmodels.ManagedObject] {
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

// CreateDeviceAgent creates a new test agent in Cumulocity IoT and registers a cleanup function
// to delete the agent after the test completes.
func CreateDeviceAgent(t *testing.T, client *api.Client) op.Result[jsonmodels.ManagedObject] {
	mo := client.ManagedObjects.Create(context.TODO(), map[string]any{
		"name":                       "ci_" + testingutils.RandomString(16),
		"com_cumulocity_model_Agent": map[string]any{},
	})
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

func CreateEvent(t *testing.T, client *api.Client, mo *jsonmodels.ManagedObject) op.Result[jsonmodels.Event] {
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
		panic(fmt.Errorf("Error creating dummy file. %w", err))
	}

	defer f.Close()

	f.WriteString(contents)

	if err := f.Sync(); err != nil {
		panic(fmt.Errorf("Failed to fill file with dummy information. %w", err))
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
		panic(fmt.Errorf("Error creating dummy file. %w", err))
	}

	defer f.Close()

	if err := f.Truncate(size); err != nil {
		panic(fmt.Errorf("Failed to fill file with dummy information. %w", err))
	}

	if err := f.Sync(); err != nil {
		panic(fmt.Errorf("Failed to sync file with dummy information. %w", err))
	}

	filepath = f.Name()
	return
}

func MustReadAll(t *testing.T, r io.Reader) []byte {
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return out
}

func DecodeAnsi(v []byte) []byte {
	ansi_escape := regexp.MustCompile("\x1B(?:[@-Z\\-_]|[[0-?]*[ -/]*[@-~])")
	return ansi_escape.ReplaceAll(v, []byte{})
}
