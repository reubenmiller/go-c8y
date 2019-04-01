package c8ytestutils

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/microservice"
	"github.com/spf13/viper"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// NewTestSetup return a new setup environment
func NewTestSetup() *SetupConfiguration {
	setup := &SetupConfiguration{}
	setup.Devices = make([]c8y.ManagedObject, 0)
	setup.Microservices = make([]*microservice.Microservice, 0)
	return setup

}

// SetupConfiguration represents the Cumulocity test environment
type SetupConfiguration struct {
	mu              sync.Mutex
	Devices         []c8y.ManagedObject
	Microservices   []*microservice.Microservice
	BootstrapClient *c8y.Client
}

// NewClient returns a new test client
func (s *SetupConfiguration) NewClient() *c8y.Client {
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
		log.Printf("Warning could not read configuration file")
	}

	// Set default settings
	config.SetDefault("report.concurrency", 20)
	config.SetDefault("log.file", "application.log")

	// Enable all variables to be defined as (case-senstive) environment variables in the form of
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	// Add extra aliases for Cumulocity Microservice SDK Specific environment variables
	config.BindEnv("c8y.host", "C8Y_BASEURL")
	config.BindEnv("c8y.tenant", "C8Y_BOOTSTRAP_TENANT")
	config.BindEnv("c8y.username", "C8Y_BOOTSTRAP_USER")
	config.BindEnv("c8y.username", "C8Y_USER")
	config.BindEnv("c8y.password", "C8Y_BOOTSTRAP_PASSWORD")

	requiredProperties := []string{
		"c8y.host",
		"c8y.tenant",
		"c8y.username",
		"c8y.password",
	}

	CheckRequiredConfiguration(config, requiredProperties...)

	return config
}

// CheckRequiredConfiguration checks to see if all of the required properties are present
func CheckRequiredConfiguration(config *viper.Viper, props ...string) {
	missingProps := []string{}
	for _, prop := range props {
		if config.GetString(prop) == "" {
			missingProps = append(missingProps, prop)
		}
	}
	if len(missingProps) > 0 {
		panic(fmt.Sprintf("Missing required properties. %s", strings.Join(missingProps, ",")))
	}
	return
}

// BootstrapApplication creates an application
func (s *SetupConfiguration) BootstrapApplication(appName ...string) *microservice.Microservice {
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
	client := s.BootstrapClient
	if client == nil {
		client = c8y.NewClient(nil, host, tenant, username, password, false)
	}

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

	s.mu.Lock()
	if s.BootstrapClient == nil {
		s.BootstrapClient = client
	}
	s.Microservices = append(
		s.Microservices,
		ms,
	)
	s.mu.Unlock()

	return ms
}

// Cleanup removes all of the test devices and clients created in the Test setup
func (s *SetupConfiguration) Cleanup() {
	log.Printf("Running Cleanup\n")

	client := s.NewClient()
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove all the devices that were created during testing
	for _, mo := range s.Devices {
		_, err := client.Inventory.Delete(context.Background(), mo.ID)
		if err != nil {
			log.Printf("Could not remove the id. %s", err)
		}
	}
	s.Devices = nil

	// Cleanup the microservices that were created
	for _, ms := range s.Microservices {
		ms.DeleteMicroserviceAgent()

		log.Printf("Deleting application id=%s", ms.Application.ID)
		if _, err := client.Application.Delete(
			context.Background(),
			ms.Application.ID,
		); err != nil {
			log.Printf("Failed to delete microservice application. %s", err)
		}
	}
}

// NewRandomTestDevice create a new random test device to be used in a test
func (s *SetupConfiguration) NewRandomTestDevice(prefix ...string) (*c8y.ManagedObject, error) {
	client := s.NewClient()

	var err error
	var mo *c8y.ManagedObject
	var deviceName string

	if len(prefix) > 0 && prefix[0] != "" {
		deviceName = prefix[0]
	} else {
		deviceName = "TestDevice"
	}
	deviceName = deviceName + randSeq(10)

	mo, _, err = client.Inventory.Create(
		context.Background(),
		c8y.NewAgent(deviceName),
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create device: %s", err)
	}
	// Store a reference to the test device
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Devices = append(s.Devices, *mo)

	return mo, nil
}
