package microservice

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/operations"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
)

// GetAgent returns the agent representation of the microservice
func (m *Microservice) GetAgent() op.Result[jsonmodels.ManagedObject] {
	return m.Client.ManagedObjects.Get(
		m.WithServiceUser(),
		m.Client.ManagedObjects.ByExternalID(m.Config.GetIdentityType(), m.Config.GetApplicationName()),
		managedobjects.GetOptions{},
	)
}

// CreateMicroserviceRepresentation Create a microservice representation in the Cumulocity platform, so that the microservice can store its configuration in the managed object
func (m *Microservice) CreateMicroserviceRepresentation() op.Result[jsonmodels.ManagedObject] {
	mo := m.Client.ManagedObjects.GetOrCreateByExternalID(m.WithServiceUser(), managedobjects.GetOrCreateByExternalIDOptions{
		ExternalIDType: m.Config.GetIdentityType(),
		ExternalID:     m.Config.GetApplicationName(),
		Body: map[string]any{
			"c8y_IsDevice":            map[string]any{},
			"c8y_SupportedOperations": m.SupportedOperations,
		},
	})
	if mo.Err != nil {
		return mo
	}

	// Create Managed Object (with agent fragment)
	configuration := m.Config.GetConfigurationString()

	// Set default agent information
	agentInfo := m.AgentInformation
	if agentInfo.Model == "" {
		agentInfo.Model = m.Config.GetApplicationName()
	}

	agentMo := &AgentManagedObject{
		AgentConfiguration: &AgentConfiguration{
			Config: configuration,
		},
		ManagedObject: model.ManagedObject{
			Name: m.Config.GetApplicationName(),
			Type: m.Config.GetIdentityType(),
		},
		AgentInformation:         &agentInfo,
		AgentSupportedOperations: m.SupportedOperations,
		DeviceFragment:           model.DeviceFragment{},
	}

	return m.Client.ManagedObjects.Update(context.Background(), mo.Data.ID(), agentMo)
}

// GetConfiguration returns the Agent configuration as text. This needs to be parsed separately by the calling function.
func (m *Microservice) GetConfiguration() (string, error) {
	mo := m.GetAgent()

	if mo.Err != nil {
		return "", mo.Err
	}

	node := mo.Data.Get("c8y_Configuration.configuration")
	if !node.Exists() {
		return "", fmt.Errorf("No configuration found on managed object id=%s", mo.Data.ID())
	}

	return node.String(), nil
}

// SaveConfiguration save the agent configuration to it's managed object
func (m *Microservice) SaveConfiguration(rawConfiguration string) error {
	body := make(map[string]interface{})
	configuration := make(map[string]interface{})
	timestamp := time.Now().Format(time.UnixDate)
	lines := strings.Split(rawConfiguration, "\n")

	if len(lines) > 0 && strings.HasPrefix(lines[0], "#") {
		if _, err := dateparse.ParseAny(lines[0][1:]); err == nil {
			// Remove previous date from the first line
			lines = lines[1:]
		}
	}
	lines = append([]string{"#" + timestamp}, lines...)
	configuration["config"] = strings.Join(lines, "\n")
	body["c8y_Configuration"] = configuration

	if result := m.Client.ManagedObjects.Update(m.WithServiceUser(), m.AgentID, body); result.Err != nil {
		return fmt.Errorf("Error updating the configuration in the managed object. %s", result.Err)
	}

	return nil
}

// DeleteMicroserviceAgent removes the microservice's agent managed object if it exists.
func (m *Microservice) DeleteMicroserviceAgent() error {
	if m.AgentID == "" {
		return nil
	}
	slog.Info("Deleting microservice's agent managed object", "id", m.AgentID)

	result := m.Client.ManagedObjects.Delete(
		m.WithServiceUser(),
		m.AgentID,
		managedobjects.DeleteOptions{},
	)
	if result.Err != nil {
		slog.Error("Could not delete microservice's agent managed object", "err", result.Err)
	}
	return result.Err
}

// RegisterMicroserviceAgent registers an agent representation of the microservice
func (m *Microservice) RegisterMicroserviceAgent() error {
	slog.Info("Registering microservice agent")

	mo := m.CreateMicroserviceRepresentation()

	if mo.Err == nil {
		slog.Info("Start Polling for Operations on device", "id", mo.Data.ID())
		m.AgentID = mo.Data.ID()

		// Get existing configuration
		m.CheckForNewConfiguration()

		if existingConfig, configErr := m.GetConfiguration(); configErr == nil {
			slog.Info("Loading existing configuration from the platform")
			m.UpdateApplicationConfiguration(existingConfig)
		}

		for _, key := range m.Config.viper.AllKeys() {
			value := m.Config.viper.GetString(key)
			slog.Info("property", "key", key, "value", value)
		}

		m.StartOperationPolling()
		// m.SubscribeToOperations(nil)
	}
	return mo.Err
}

var (
	configurationChangeCount = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "configuration_change_count",
			Help: "Number of times the configuration has been updated",
		},
	)
)

// GetOperations returns a list of operations in the given status i.e. PENDING, EXECUTING, SUCCESS, FAILED
func (m *Microservice) GetOperations(status types.OperationStatus) op.Result[jsonmodels.Operation] {
	return m.Client.Operations.List(m.WithServiceUser(), operations.ListOptions{
		Status:  status,
		AgentID: m.AgentID,
		PaginationOptions: pagination.PaginationOptions{
			PageSize: 5,
		},
	})
}

// UpdateApplicationConfiguration updates the application configuration based on a new config which is parsed from a string. Values should be in the form of "<key>=<value>" separated by a \n char
func (m *Microservice) UpdateApplicationConfiguration(configAsString string) {
	slog.Info("Updating application configuration")
	items := strings.Split(configAsString, "\n")

	for _, item := range items {
		slog.Info("Parsing configuration item", "value", item)
		parts := strings.Split(item, "=")

		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if m.Config.isPrivateSetting(key) {
				slog.Info("Ignoring private property", "key", key)
			} else if strings.HasPrefix(key, "#") {
				slog.Info("Ignore comment", "key", key)
			} else {
				slog.Info("Setting property", "key", key, "value", value)
				m.Config.viper.Set(key, value)
			}
		}
	}
}

func (m *Microservice) onUpdateConfigurationOperation(operationID string, newConfiguration string) {

	m.Client.Operations.Update(
		m.WithServiceUser(),
		operationID,
		model.Operation{
			Status: types.OperationStatusPending,
		},
	)

	// Save configuration
	if updateErr := m.SaveConfiguration(newConfiguration); updateErr != nil {
		// Failed Operation
		m.Client.Operations.Update(
			m.WithServiceUser(),
			operationID,
			model.Operation{
				Status: types.OperationStatusFailed,
			},
		)
	} else {
		// Successful Operation
		m.UpdateApplicationConfiguration(newConfiguration)
		m.Client.Operations.Update(
			m.WithServiceUser(),
			operationID,
			model.Operation{
				Status: types.OperationStatusSuccessful,
			},
		)

		if m.Hooks.OnConfigurationUpdateFunc != nil {
			slog.Info("Calling OnConfigurationUpdate lifecycle hook")
			go m.Hooks.OnConfigurationUpdateFunc(*m.Config)
		}
	}
}

// CheckForNewConfiguration checks for any pending operations with new configuration
func (m *Microservice) CheckForNewConfiguration() {
	slog.Info("Checking pending operations")
	result := m.GetOperations(types.OperationStatusPending)

	if result.Err != nil {
		slog.Error("Failed to get operations", "err", result.Err)
		return
	}

	for item, _ := range op.Iter2(result) {

		//
		// Update Configuration Operation
		//
		if c8yConfig := item.Get("c8y_Configuration.config"); c8yConfig.Exists() {
			m.onUpdateConfigurationOperation(item.ID(), c8yConfig.String())
			configurationChangeCount.Inc()
		}
	}
}

// StartOperationPolling start the polling of the operations
func (m *Microservice) StartOperationPolling() {
	interval := strings.TrimSpace(m.Config.viper.GetString("agent.operations.pollRate"))

	if interval == "" || interval == "0" {
		slog.Info("Skipping operation polling task")
		return
	}
	slog.Info("Adding operation polling task with interval", "value", interval)
	_, err := m.Scheduler.cronjob.AddFunc(interval, func() {
		m.CheckForNewConfiguration()
	})

	if err != nil {
		slog.Error("Could not create polling task with interval", "value", interval, "err", err)
	}
}
