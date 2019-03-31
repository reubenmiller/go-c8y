package microservice_test

import (
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	// c8y "github.com/reubenmiller/go-c8y"
	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/microservice"
	"github.com/reubenmiller/go-c8y/microservice_test/testingutils"
)

// TestMicroservice_RegisterMicroserviceAgent test if the microservice registers an agent which is used to represent the microservice (i.e. to enable operations, alarm and events)
func TestMicroservice_RegisterMicroserviceAgent(t *testing.T) {
	app := bootstrapApplication()
	err := app.RegisterMicroserviceAgent()
	testingutils.Ok(t, err)

	mo := app.GetAgent()
	testingutils.Assert(t, mo != nil, "Agent managed object should not be nil")
}

func TestMicroservice_GetConfiguration(t *testing.T) {
	/*
		Default configuration should be written to the Agent managed object and it should be be retrievable
	*/
	app := bootstrapApplication()
	app.Config.SetDefault("test.prop", "hello-x")

	err := app.RegisterMicroserviceAgent()
	testingutils.Ok(t, err)

	configStr, err := app.GetConfiguration()
	testingutils.Ok(t, err)
	testingutils.Assert(t, strings.Contains(configStr, "test.prop=hello-x"), "Configuration text should contain the given property")
}

func TestMicroservice_SaveConfiguration(t *testing.T) {
	/*
		Microservice should be able to update the configuration to its Agent managed object
	*/
	app := bootstrapApplication()

	err := app.RegisterMicroserviceAgent()
	testingutils.Ok(t, err)
	configStr := `
	custom.prop1.list=1,2,3
	custom.prop2.delay=true
	`
	err = app.SaveConfiguration(configStr)
	testingutils.Ok(t, err)

	// Get the configuration
	mo := app.GetAgent()
	testingutils.Assert(t, mo.C8yConfiguration != nil, "Configuration fragment should not be empty")
	testingutils.Assert(t, strings.Contains(mo.C8yConfiguration.Configuration, "custom.prop1.list=1,2,3"), "Should contain property in config")
	testingutils.Assert(t, strings.Contains(mo.C8yConfiguration.Configuration, "custom.prop2.delay=true"), "Should contain property in config")
}

// TestMicroservice_OnUpdateConfigurationHook tests if the OnUpdateConfiguration hook
// is called when
func TestMicroservice_OnUpdateConfigurationHook(t *testing.T) {
	app := bootstrapApplication()
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
	testingutils.Ok(t, err)

	//
	// Create update config operation
	_, _, err = app.Client.Operation.Create(
		app.WithServiceUser(),
		map[string]interface{}{
			"deviceId": app.AgentID,
			"c8y_Configuration": map[string]interface{}{
				"name": "Update configuration",
				"config": `
prop1=1
prop2=2
				`,
			},
		},
	)
	testingutils.Ok(t, err)

	timeout := time.NewTimer(20 * time.Second)

	select {
	case <-ch:
		log.Printf("Received hook")
		break
	case <-timeout.C:
		// timeout
		log.Printf("Timeout whilst waiting for update configuration hook")
		break
	}

	testingutils.Equals(t, int64(1), configUpdateCounter)
	testingutils.Equals(t, "1", msConfig.GetString("prop1"))
	testingutils.Equals(t, "2", msConfig.GetString("prop2"))
}

func TestMicroservice_SubscribeToNotifications(t *testing.T) {
	/*
		Microservice should be able to subscribe to notifications
	*/
	var eventCounter int64
	var operationCounter int64
	var err error

	app := bootstrapApplication()
	err = app.RegisterMicroserviceAgent()
	testingutils.Ok(t, err)

	err = app.SubscribeToNotifications(
		app.WithServiceUserCredentials(),
		c8y.RealtimeEvents(app.AgentID),
		func(msg *c8y.Message) {
			// New message received
			atomic.AddInt64(&eventCounter, 1)
		},
	)
	testingutils.Ok(t, err)

	err = app.SubscribeToNotifications(
		app.WithServiceUserCredentials(),
		c8y.RealtimeOperations(app.AgentID),
		func(msg *c8y.Message) {
			// New message received
			atomic.AddInt64(&operationCounter, 1)
		},
	)
	testingutils.Ok(t, err)

	// Create event
	_, _, err = app.Client.Event.Create(
		app.WithServiceUser(),
		&c8y.Event{
			Time:   c8y.NewTimestamp(),
			Text:   "Something happened",
			Source: c8y.NewSource(app.AgentID),
			Type:   "testType1",
		},
	)
	testingutils.Ok(t, err)

	// Create operation
	op := c8y.NewCustomOperation(app.AgentID)
	op.Set("com_custom_Operation", map[string]string{
		"name": "Custom Operation 1",
	})
	_, _, err = app.Client.Operation.Create(
		app.WithServiceUser(),
		op,
	)
	testingutils.Ok(t, err)

	// Give the cep engine a chance to send the notification
	time.Sleep(2000 * time.Millisecond)

	testingutils.Equals(t, int64(1), atomic.LoadInt64(&eventCounter))
	testingutils.Equals(t, int64(1), atomic.LoadInt64(&operationCounter))
}
