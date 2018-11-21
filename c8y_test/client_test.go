package c8y_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/spf13/viper"
)

func createTestClient() *c8y.Client {
	config := readConfig()

	host := config.GetString("c8y.host")
	tenant := config.GetString("c8y.tenant")
	username := config.GetString("c8y.username")
	password := config.GetString("c8y.password")

	fmt.Printf("Host=%s, Tenant=%s, Username=%s, Password=%s\n", host, tenant, username, password)
	client := c8y.NewClient(nil, host, tenant, username, password)
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

	fmt.Printf("Values: %s", mo)

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

	fmt.Printf("Values: %s", managedObjects)

	if err != nil {
		t.Errorf("Could not decode json. want: nil, got: %s", err)
	}
}
