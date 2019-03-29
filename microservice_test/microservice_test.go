package microservice_test

import (
	"log"
	"sync/atomic"
	"testing"
	"time"

	// c8y "github.com/reubenmiller/go-c8y"
	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/microservice"
	"github.com/reubenmiller/go-c8y/microservice_test/testingutils"
)

func TestMicroservice_TestClientConnection(t *testing.T) {
	app := bootstrapApplication()
	err := app.RegisterMicroserviceAgent()
	testingutils.Ok(t, err)
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

	app.SubscribeToOperations(func(msg *c8y.Message) error {
		log.Printf("Received message: %s", msg)
		return nil
	})

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
