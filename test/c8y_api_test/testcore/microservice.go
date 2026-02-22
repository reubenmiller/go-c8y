package testcore

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/microservice"
	"github.com/spf13/viper"
)

func readConfig() *viper.Viper {
	// Read configuration
	config := viper.New()
	config.SetConfigName("application")
	config.AddConfigPath(".")
	err := config.ReadInConfig()

	if err != nil {
		slog.Warn("Could not read configuration file")
	}

	// Set default settings
	config.SetDefault("report.concurrency", 20)
	config.SetDefault("log.file", "application.log")

	// Enable all variables to be defined as (case-sensitive) environment variables in the form of
	config.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.AutomaticEnv()

	// Add extra aliases for Cumulocity Microservice SDK Specific environment variables
	config.BindEnv("c8y.host", "C8Y_BASEURL")
	config.BindEnv("c8y.tenant", "C8Y_BOOTSTRAP_TENANT")
	config.BindEnv("c8y.username", "C8Y_BOOTSTRAP_USER")
	config.BindEnv("c8y.username", "C8Y_USER")
	config.BindEnv("c8y.password", "C8Y_BOOTSTRAP_PASSWORD")
	config.BindEnv("c8y.token", "C8Y_TOKEN")

	requiredProperties := []string{
		"c8y.host",
		"c8y.tenant",
		"c8y.username",
		// "c8y.password",
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
}

// BootstrapApplication creates an application
func BootstrapApplication(t *testing.T, appName ...string) *microservice.Microservice {
	config := readConfig()

	var applicationName string

	if len(appName) == 0 {
		applicationName = "ci-test" + strings.ToLower(testingutils.RandomString(5))
	} else {
		applicationName = appName[0]
	}

	host := config.GetString("c8y.host")
	tenant := config.GetString("c8y.tenant")
	username := config.GetString("c8y.username")
	password := config.GetString("c8y.password")

	slog.Info("Bootstrapping application", "host", host, "tenant", tenant, "username", username, "password", password)

	client := CreateTestClient(t)

	app := client.Microservices.Create(
		context.Background(),
		model.NewMicroservice(applicationName),
	)

	if app.Err != nil {
		slog.Error("Could not create application. %s", "err", app.Err)
		t.FailNow()
	}

	t.Cleanup(func() {
		client.Microservices.Delete(context.Background(), app.Data.ID(), applications.DeleteOptions{
			Force: true,
		})
	})

	// Set required roles
	appUpdate := client.Applications.Update(
		context.Background(),
		app.Data.ID(),
		&model.Application{
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

	if appUpdate.Err != nil {
		slog.Error("Could not update microservice's requiredRoles", "err", appUpdate.Err)
		t.FailNow()
	}

	// Subscribe to application
	result := client.Microservices.Subscribe(
		context.Background(),
		client.Auth.Tenant,
		app.Data.Self(),
	)

	if result.Err != nil {
		slog.Error("Could not subscribe to application", "err", result.Err)
		t.FailNow()
	}

	// Get Microservice Credentials
	appCredentials := client.Microservices.BootstrapUser.Get(
		context.Background(),
		app.Data.ID(),
	)

	if appCredentials.Err != nil {
		slog.Error("Could not get application credentials", "err", appCredentials.Err)
		t.FailNow()
	}

	// Set microservice env variables
	os.Setenv(microservices.EnvironmentApplicationName, applicationName)
	os.Setenv(microservices.EnvironmentBootstrapTenant, appCredentials.Data.Tenant())
	os.Setenv(microservices.EnvironmentBootstrapUsername, appCredentials.Data.Username())
	os.Setenv(microservices.EnvironmentBootstrapPassword, appCredentials.Data.Password())
	// os.Unsetenv("C8Y_TOKEN")

	ms := microservice.NewDefaultMicroservice(microservice.Options{})

	if err := ms.TestClientConnection(); err != nil {
		slog.Error("Microservice test connection failed", "err", err)
		t.FailNow()
	}

	t.Cleanup(func() {
		client.Devices.Delete(context.Background(), ms.AgentID, managedobjects.DeleteOptions{})
	})

	return ms
}
