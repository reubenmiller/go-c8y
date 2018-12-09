package c8y_test

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/spf13/viper"
)

var CumulocityConfiguration SetupConfiguration

type TestDevice struct {
	ID           string
	IdentityType string
	ExternalID   string
}

type SetupConfiguration struct {
	ExampleDevice TestDevice
}

func TestMain(m *testing.M) {
	setupTestSystem()

	res := m.Run()

	cleanupTestSystem()

	os.Exit(res)
}

// setupTestSystem configures the system ready for testing
func setupTestSystem() {
	log.Printf("Setting up tests\n")

	mo, err := createTestDevice()

	if err != nil {
		log.Printf("Could not create/find test device\n")
	} else {
		log.Printf("Using test device: %s\n", mo.ID)
	}

	if mo != nil {
		CumulocityConfiguration.ExampleDevice.ID = mo.ID
	}
}

func cleanupTestSystem() {
	log.Printf("Running Cleanup\n")

	client := createTestClient()
	config := readConfig()

	removeDevices := config.GetBool("testing.cleanup.removeDevice")
	if removeDevices && CumulocityConfiguration.ExampleDevice.ID != "" {
		log.Printf("Removing test device\n")
		_, err := client.Inventory.Delete(context.Background(), CumulocityConfiguration.ExampleDevice.ID)
		if err != nil {
			log.Printf("Could not remove the id. %s", err)
		}
	}
}

// createTestDevice create a test device by looking for the special test external identity
func createTestDevice() (*c8y.ManagedObject, error) {
	client := createTestClient()

	var err error
	var mo *c8y.ManagedObject

	externalType := "c8y_Testing"
	externalID := "c8yDeviceTest001"

	moRef, _, _ := client.Identity.GetExternalID(context.Background(), externalType, externalID)

	if moRef == nil {
		mo, _, err = client.Inventory.CreateDevice(context.Background(), externalID)
		if err != nil {
			return nil, fmt.Errorf("Failed to create device: %s", err)
		}
		// Store a reference to the test device
		CumulocityConfiguration.ExampleDevice.ID = mo.ID

		// Create Identity for new managed object
		_, _, err = client.Identity.NewExternalIdentity(context.Background(), mo.ID, &c8y.IdentityOptions{
			Type:       externalType,
			ExternalID: externalID,
		})

		if err != nil {
			return nil, fmt.Errorf("Failed to create external id for the test managed object: %s", err)
		}
	} else {
		mo, _, err = client.Inventory.GetManagedObject(context.Background(), moRef.ManagedObject.ID, nil)

		if err != nil {
			return nil, fmt.Errorf("Failed to get managed object found using the external id")
		}
	}

	return mo, nil
}

func createTestClient() *c8y.Client {
	config := readConfig()

	host := config.GetString("c8y.host")
	tenant := config.GetString("c8y.tenant")
	username := config.GetString("c8y.username")
	password := config.GetString("c8y.password")

	log.Printf("Host=%s, Tenant=%s, Username=%s, Password=%s\n", host, tenant, username, password)
	client := c8y.NewClient(nil, host, tenant, username, password, false)
	return client
}

func readConfig() *viper.Viper {
	// Read configuration
	config := viper.New()
	config.SetConfigName("application")
	config.AddConfigPath(".")
	err := config.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Error reading configuration"))
	}

	// Set default settings
	config.SetDefault("report.concurrency", 20)
	config.SetDefault("log.file", "/var/log/go-nifgate/app.log")

	// Enable all variables to be defined as (case-senstive) environment variables in the form of
	// export C8Y_USERNAME=testuser
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	// Add extra aliases for Cumulocity Microservice SDK Specific environment variables
	config.BindEnv("c8y.host", "C8Y_BASEURL")
	config.BindEnv("c8y.tenant", "C8Y_BOOTSTRAP_TENANT")
	config.BindEnv("c8y.username", "C8Y_BOOTSTRAP_USER")
	config.BindEnv("c8y.password", "C8Y_BOOTSTRAP_PASSWORD")

	return config
}

// TestInventoryService_DecodeJSONManagedObject tests whether individual managed objects can be decoded into custom objects
func TestInventoryService_DecodeJSONManagedObject(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}

	data, _, _ := client.Inventory.GetDevices(context.Background(), opt)

	var mo c8y.ManagedObject

	err := json.Unmarshal([]byte(data.Items[0].Raw), &mo)

	log.Printf("Values: %s", mo)

	if err != nil {
		t.Errorf("Could not decode json. want: nil, got: %s", err)
	}
}

// TestInventoryService_DecodeJSONManagedObject tests whether the response from the server has be decoded to a custom object
func TestInventoryService_DecodeJSONManagedObjects(t *testing.T) {
	client := createTestClient()

	pageSize := 1
	opt := &c8y.PaginationOptions{
		PageSize: pageSize,
	}

	_, resp, _ := client.Inventory.GetDevices(context.Background(), opt)

	managedObjects := make([]c8y.ManagedObject, 0)

	err := resp.DecodeJSON(&managedObjects)

	log.Printf("Values: %s", managedObjects)

	if err != nil {
		t.Errorf("Could not decode json. want: nil, got: %s", err)
	}
}
