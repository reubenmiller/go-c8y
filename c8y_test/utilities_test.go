package c8y_test

import (
	"context"
	"fmt"
	"log"
	"math"
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
	config.SetDefault("log.file", "application.log")

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

// createTestDevice create a test device by looking for the special test external identity
func createTestDevice() (*c8y.ManagedObject, error) {
	client := createTestClient()

	var err error
	var mo *c8y.ManagedObject

	externalType := "c8y_Testing"
	externalID := "c8yDeviceTest001"

	moRef, _, _ := client.Identity.GetExternalID(context.Background(), externalType, externalID)

	if moRef == nil {
		mo, _, err = client.Inventory.CreateManagedObject(context.Background(), c8y.NewAgent(externalID))
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

const float64EqualityThreshold = 1e-7

func almostEqual(a, b float64) bool {
	diff := math.Abs(a - b)
	return diff <= float64EqualityThreshold
}
