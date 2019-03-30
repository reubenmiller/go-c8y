package microservice_test

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/microservice"

	"github.com/reubenmiller/go-c8y/microservice_test/testingutils"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/spf13/viper"
)

var CumulocityConfiguration SetupConfiguration

type SetupConfiguration struct {
	mu              sync.Mutex
	BootstrapClient *c8y.Client
	Microservices   []*microservice.Microservice
}

func TestMain(m *testing.M) {
	rand.Seed(time.Now().UnixNano())

	setupTestSystem()

	res := m.Run()

	cleanupTestSystem()

	os.Exit(res)
}

func setupTestSystem() {

}

func cleanupTestSystem() {
	CumulocityConfiguration.mu.Lock()

	client := CumulocityConfiguration.BootstrapClient

	if client == nil {
		log.Printf("Can't clean up anything because the bootstrap client is nil")
		return
	}

	for _, ms := range CumulocityConfiguration.Microservices {
		ms.DeleteMicroserviceAgent()

		log.Printf("Deleting application id=%s", ms.Application.ID)
		if _, err := client.Application.Delete(
			context.Background(),
			ms.Application.ID,
		); err != nil {
			log.Printf("Failed to delete microservice application. %s", err)
		}
	}
	CumulocityConfiguration.mu.Unlock()

}

// bootstrapApplication creates an application
func bootstrapApplication(appName ...string) *microservice.Microservice {
	config := readConfig()

	var applicationName string

	if len(appName) == 0 {
		applicationName = "citest" + strings.ToLower(testingutils.RandomString(5))
	} else {
		applicationName = appName[0]
	}

	host := config.GetString("c8y.host")
	tenant := config.GetString("c8y.tenant")
	username := config.GetString("c8y.username")
	password := config.GetString("c8y.password")

	log.Printf("Host=%s, Tenant=%s, Username=%s, Password=%s\n", host, tenant, username, password)
	client := c8y.NewClient(nil, host, tenant, username, password, false)

	app, _, err := client.Application.Create(
		context.Background(),
		c8y.NewApplicationMicroservice(applicationName),
	)

	if err != nil {
		log.Fatalf("Could not create application. %s", err)
	}

	// Set required roles
	_, _, err = client.Application.Update(
		context.Background(),
		app.ID,
		&c8y.Application{
			RequiredRoles: []string{
				"ROLE_INVENTORY_READ",
				"ROLE_INVENTORY_CREATE",
				"ROLE_INVENTORY_ADMIN",
				"ROLE_IDENTITY_READ",
				"ROLE_IDENTITY_ADMIN",
				"ROLE_AUDIT_READ",
				"ROLE_AUDIT_ADMIN",
				"ROLE_MEASUREMENT_READ",
				"ROLE_MEASUREMENT_ADMIN",
				"ROLE_EVENT_READ",
				"ROLE_EVENT_ADMIN",
				"ROLE_ALARM_ADMIN",
				"ROLE_ALARM_READ",
				"ROLE_DEVICE_CONTROL_READ",
				"ROLE_DEVICE_CONTROL_ADMIN",
			},
		},
	)

	if err != nil {
		log.Fatalf("Could not update microservice's requiredRoles. %s", err)
	}

	// Subscribe to application
	_, _, err = client.Tenant.AddApplicationReference(
		context.Background(),
		client.TenantName,
		app.Self,
	)

	if err != nil {
		log.Fatalf("Could not subscribe to application. %s", err)
	}

	// Get Microservice Credentials
	appCredentials, _, err := client.Application.GetApplicationUser(
		context.Background(),
		app.ID,
	)

	if err != nil {
		log.Fatalf("Could not get application credentials. %s", err)
	}

	// Set microservice env variables
	os.Setenv(c8y.EnvironmentApplicationName, applicationName)
	os.Setenv(c8y.EnvironmentBootstrapTenant, appCredentials.Tenant)
	os.Setenv(c8y.EnvironmentBootstrapUsername, appCredentials.Username)
	os.Setenv(c8y.EnvironmentBootstrapPassword, appCredentials.Password)

	ms := microservice.NewDefaultMicroservice(microservice.Options{})

	if err := ms.TestClientConnection(); err != nil {
		log.Fatalf("Microservice test connection failed. %s", err)
	}

	CumulocityConfiguration.mu.Lock()
	if CumulocityConfiguration.BootstrapClient == nil {
		CumulocityConfiguration.BootstrapClient = client
	}
	CumulocityConfiguration.Microservices = append(
		CumulocityConfiguration.Microservices,
		ms,
	)
	CumulocityConfiguration.mu.Unlock()

	return ms
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
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	// Add extra aliases for Cumulocity Microservice SDK Specific environment variables
	config.BindEnv("c8y.host", c8y.EnvironmentBaseURL)
	config.BindEnv("c8y.tenant", c8y.EnvironmentBootstrapTenant)
	config.BindEnv("c8y.username", c8y.EnvironmentBootstrapUsername)
	config.BindEnv("c8y.password", c8y.EnvironmentBootstrapPassword)

	return config
}
