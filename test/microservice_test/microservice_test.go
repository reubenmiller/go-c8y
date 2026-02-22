package microservice_test

import (
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/pkg/microservice"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bootstrapApplication(t *testing.T) *microservice.Microservice {
	return testcore.BootstrapApplication(t)
}

// TestMicroservice_RegisterMicroserviceAgent test if the microservice registers an agent which is used to represent the microservice (i.e. to enable operations, alarm and events)
func TestMicroservice_RegisterMicroserviceAgent(t *testing.T) {
	app := bootstrapApplication(t)
	err := app.RegisterMicroserviceAgent()
	assert.NoError(t, err)

	mo := app.GetAgent()
	assert.NoError(t, mo.Err)
	assert.NotEmpty(t, mo.Data.ID(), "Agent managed object should not be empty")
}

func TestMicroservice_GetConfiguration(t *testing.T) {
	/*
		Default configuration should be written to the Agent managed object and it should be be retrievable
	*/
	app := bootstrapApplication(t)
	app.Config.SetDefault("test.prop", "hello-x")

	err := app.RegisterMicroserviceAgent()
	assert.NoError(t, err)

	configStr, err := app.GetConfiguration()
	assert.NoError(t, err)
	assert.Contains(t, configStr, "test.prop=hello-x")
}

func TestMicroservice_SaveConfiguration(t *testing.T) {
	/*
		Microservice should be able to update the configuration to its Agent managed object
	*/
	app := bootstrapApplication(t)

	err := app.RegisterMicroserviceAgent()
	assert.NoError(t, err)
	configStr := `
	custom.prop1.list=1,2,3
	custom.prop2.delay=true
	`
	err = app.SaveConfiguration(configStr)
	assert.NoError(t, err)

	// Get the configuration
	mo := app.GetAgent()
	node := mo.Data.Get("c8y_Configuration")

	require.True(t, node.Exists())
	config := node.Get("config").String()
	assert.Contains(t, config, "custom.prop1.list=1,2,3")
	assert.Contains(t, config, "custom.prop2.delay=true")
}

// TestMicroservice_OnUpdateConfigurationHook tests if the OnUpdateConfiguration hook
// is called when
func TestMicroservice_OnUpdateConfigurationHook(t *testing.T) {
	app := bootstrapApplication(t)
	var configUpdateCounter int64
	var msConfig *microservice.Configuration
	ch := make(chan bool, 1)
	var err error

	app.Hooks.OnConfigurationUpdateFunc = func(config microservice.Configuration) {
		atomic.AddInt64(&configUpdateCounter, 1)
		msConfig = &config
		ch <- true
	}

	app.Config.SetDefault("agent.operations.pollRate", "@every 5s")

	err = app.RegisterMicroserviceAgent()
	app.Scheduler.Start()
	assert.NoError(t, err)

	//
	// Create update config operation
	result := app.Client.Operations.Create(
		app.ServiceUserContext(),
		map[string]any{
			"deviceId": app.AgentID,
			"c8y_Configuration": map[string]any{
				"name": "Update configuration",
				"config": `
prop1=1
prop2=2
				`,
			},
		},
	)
	assert.NoError(t, result.Err)

	timeout := time.NewTimer(20 * time.Second)

	select {
	case <-ch:
		slog.Info("Received hook")
		break
	case <-timeout.C:
		// timeout
		slog.Info("Timeout whilst waiting for update configuration hook")
		break
	}

	assert.EqualValues(t, 1, configUpdateCounter)
	assert.Equal(t, "1", msConfig.GetString("prop1"))
	assert.Equal(t, "2", msConfig.GetString("prop2"))
}

func TestMicroservice_SubscribeToNotifications(t *testing.T) {
	/*
		Microservice should be able to subscribe to notifications
	*/
	var eventCounter int64
	var operationCounter int64
	var err error

	app := bootstrapApplication(t)
	err = app.RegisterMicroserviceAgent()
	assert.NoError(t, err)

	err = app.SubscribeToNotifications(
		app.WithServiceUserCredentials(),
		realtime.Events(app.AgentID),
		func(msg *realtime.Message) {
			// New message received
			atomic.AddInt64(&eventCounter, 1)
		},
	)
	assert.NoError(t, err)

	err = app.SubscribeToNotifications(
		app.WithServiceUserCredentials(),
		realtime.Operations(app.AgentID),
		func(msg *realtime.Message) {
			// New message received
			atomic.AddInt64(&operationCounter, 1)
		},
	)
	assert.NoError(t, err)

	// Wait for subscriptions to be processed
	time.Sleep(5 * time.Second)

	// Create event
	result1 := app.Client.Events.Create(
		app.ServiceUserContext(),
		&model.Event{
			Time:   time.Now(),
			Text:   "Something happened",
			Source: model.NewSource(app.AgentID),
			Type:   "testType1",
		},
	)
	assert.NoError(t, result1.Err)

	// Create operation
	op := map[string]any{
		"deviceId": app.AgentID,
		"com_custom_Operation": map[string]any{
			"name": "Custom Operation 1",
		},
	}
	result2 := app.Client.Operations.Create(
		app.ServiceUserContext(),
		op,
	)
	assert.NoError(t, result2.Err)

	// Give the cep engine a chance to send the notification
	time.Sleep(2000 * time.Millisecond)

	assert.EqualValues(t, 1, atomic.LoadInt64(&eventCounter))
	assert.EqualValues(t, 1, atomic.LoadInt64(&operationCounter))
}
