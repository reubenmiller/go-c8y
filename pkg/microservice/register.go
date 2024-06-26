package microservice

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"go.uber.org/zap"
)

// GetAgent returns the agent representation of the microservice
func (m *Microservice) GetAgent() *c8y.ManagedObject {
	var agent *c8y.ManagedObject
	extID, _, err := m.Client.Identity.GetExternalID(m.WithServiceUser(), m.Config.GetIdentityType(), m.Config.GetApplicationName())

	if err != nil {
		zap.S().Warnf("No external identity exists for type=%s, id=%s. err %s", m.Config.GetIdentityType(), m.Config.GetApplicationName(), err)
	} else {
		zap.L().Info("Retrieving managed object by id found in external id")
		mo, _, err := m.Client.Inventory.GetManagedObject(m.WithServiceUser(), extID.ManagedObject.ID, nil)

		if err != nil {
			zap.S().Errorf("Failed to return managed object by the ID [%s] given in the External Identity definition. %s", extID.ManagedObject.ID, err)
		}
		agent = mo
	}

	return agent
}

// CreateMicroserviceRepresentation Create a microservice representation in the Cumulocity platform, so that the microservice can store its configuration in the managed object
func (m *Microservice) CreateMicroserviceRepresentation() (*c8y.ManagedObject, error) {
	mo := m.GetAgent()

	if mo != nil {
		zap.S().Infof("Found agent by its identity. [%s]", mo.ID)
		zap.S().Infof("Updating agent meta information (info and supported operations). [%s]", mo.ID)

		agentMo := &AgentManagedObject{
			AgentSupportedOperations: m.SupportedOperations,
		}
		// Only set information if revision is set
		if m.AgentInformation.Revision != "" {
			agentMo.AgentInformation = &m.AgentInformation
		}
		updatedMo, _, err := m.Client.Inventory.Update(m.WithServiceUser(), mo.ID, agentMo)

		if err != nil {
			zap.S().Errorf("Failed to update agent managed object with meta information. %s", err)
			return mo, nil
		}
		zap.S().Infof("Updated agent meta information successfully [%s]", mo.ID)
		return updatedMo, nil
	}

	zap.S().Infof("Could not find agent so it will be created")

	// Create Managed Object (with agent fragment)

	identityType := m.Config.GetIdentityType()
	externalID := m.Config.GetApplicationName()
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
		ManagedObject: c8y.ManagedObject{
			Name: m.Config.GetApplicationName(),
			Type: m.Config.GetIdentityType(),
		},
		AgentInformation:         &agentInfo,
		AgentSupportedOperations: m.SupportedOperations,
		DeviceFragment:           c8y.DeviceFragment{},
	}

	mo, _, err := m.Client.Inventory.Create(m.WithServiceUser(), agentMo)

	if err != nil {
		zap.S().Errorf("Could not create device managed object. %s", err)
		return nil, fmt.Errorf("Error creating the device managed object")
	}
	zap.S().Infof("Created managed object: %s", mo.ID)

	// Create External ID reference to the new managed object
	if _, _, err := m.Client.Identity.Create(m.WithServiceUser(), mo.ID, identityType, externalID); err != nil {
		return mo, fmt.Errorf("Error creating external id for managed object, however the managed object was created. %s", err)
	}

	return mo, nil
}

// GetConfiguration returns the Agent configuration as text. This needs to be parsed separately by the calling function.
func (m *Microservice) GetConfiguration() (string, error) {
	mo := m.GetAgent()

	if mo == nil {
		return "", fmt.Errorf("Could not retrieve managed object")
	}

	if mo.C8yConfiguration == nil {
		return "", fmt.Errorf("No configuration found on managed object id=%s", mo.ID)
	}

	return mo.C8yConfiguration.Configuration, nil
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

	if _, _, err := m.Client.Inventory.Update(m.WithServiceUser(), m.AgentID, body); err != nil {
		return fmt.Errorf("Error updating the configuration in the managed object. %s", err)
	}

	return nil
}

// DeleteMicroserviceAgent removes the microservice's agent managed object if it exists.
func (m *Microservice) DeleteMicroserviceAgent() error {
	if m.AgentID == "" {
		return nil
	}
	zap.S().Infof("Deleting microservice's agent managed object [id=%s]", m.AgentID)

	_, err := m.Client.Inventory.Delete(
		m.WithServiceUser(),
		m.AgentID,
	)
	if err != nil {
		zap.S().Errorf("Could not delete microservice's agent managed object. %s", err)
	}
	return err
}

// RegisterMicroserviceAgent registers an agent representation of the microservice
func (m *Microservice) RegisterMicroserviceAgent() error {
	zap.L().Info("Registering microservice agent")

	mo, err := m.CreateMicroserviceRepresentation()

	if err == nil {
		zap.S().Infof("Start Polling for Operations on device %s", mo.ID)
		m.AgentID = mo.ID

		// Get existing configuration
		m.CheckForNewConfiguration()

		if existingConfig, configErr := m.GetConfiguration(); configErr == nil {
			zap.L().Info("Loading existing configuration from the platform")
			m.UpdateApplicationConfiguration(existingConfig)
		}

		for _, key := range m.Config.viper.AllKeys() {
			value := m.Config.viper.GetString(key)
			log.Printf("property: %s=%s", key, value)
		}

		m.StartOperationPolling()
		// m.SubscribeToOperations(nil)
	}
	return err
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
func (m *Microservice) GetOperations(status string) (*c8y.OperationCollection, *c8y.Response, error) {
	opt := &c8y.OperationCollectionOptions{
		Status:  status,
		AgentID: m.AgentID,
		PaginationOptions: c8y.PaginationOptions{
			PageSize:       5,
			WithTotalPages: false,
		},
	}

	data, resp, err := m.Client.Operation.GetOperations(m.WithServiceUser(), opt)
	return data, resp, err
}

// UpdateApplicationConfiguration updates the application configuration based on a new config which is parsed from a string. Values should be in the form of "<key>=<value>" separated by a \n char
func (m *Microservice) UpdateApplicationConfiguration(configAsString string) {
	zap.L().Info("Updating application configuration")
	items := strings.Split(configAsString, "\n")

	for _, item := range items {
		zap.S().Infof("Parsing configuration item: %s", item)
		parts := strings.Split(item, "=")

		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			if m.Config.isPrivateSetting(key) {
				zap.S().Infof("Ignoring private property [%s]", key)
			} else if strings.HasPrefix(key, "#") {
				zap.S().Infof("Ignore comment [%s]", key)
			} else {
				zap.S().Infof("Setting property [%s] to [%s]", key, value)
				m.Config.viper.Set(key, value)
			}
		} else {
			zap.L().Info("Checking item")
		}
	}
}

func (m *Microservice) onUpdateConfigurationOperation(operationID string, newConfiguration string) {

	m.Client.Operation.Update(
		m.WithServiceUser(),
		operationID,
		&c8y.OperationUpdateOptions{
			Status: c8y.OperationStatusPending,
		},
	)

	// Save configuration
	if updateErr := m.SaveConfiguration(newConfiguration); updateErr != nil {
		// Failed Operation
		m.Client.Operation.Update(
			m.WithServiceUser(),
			operationID,
			&c8y.OperationUpdateOptions{
				Status: c8y.OperationStatusFailed,
			},
		)
	} else {
		// Successful Operation
		m.UpdateApplicationConfiguration(newConfiguration)
		m.Client.Operation.Update(
			m.WithServiceUser(),
			operationID,
			&c8y.OperationUpdateOptions{
				Status: c8y.OperationStatusSuccessful,
			},
		)

		if m.Hooks.OnConfigurationUpdateFunc != nil {
			zap.S().Info("Calling OnConfigurationUpdate lifecycle hook")
			go m.Hooks.OnConfigurationUpdateFunc(*m.Config)
		}
	}
}

// CheckForNewConfiguration checks for any pending operations with new configuration
func (m *Microservice) CheckForNewConfiguration() {
	zap.L().Info("checking pending operations")
	data, _, err := m.GetOperations(c8y.OperationStatusPending)

	if err != nil {
		log.Printf("Error getting operations. %s", err)
		return
	}

	for _, op := range data.Items {

		//
		// Update Configuration Operation
		//
		if c8yConfig := op.Get("c8y_Configuration.config"); c8yConfig.Exists() {
			m.onUpdateConfigurationOperation(op.Get("id").String(), c8yConfig.String())
			configurationChangeCount.Inc()
		}
	}
}

// StartOperationPolling start the polling of the operations
func (m *Microservice) StartOperationPolling() {
	interval := strings.TrimSpace(m.Config.viper.GetString("agent.operations.pollRate"))

	if interval == "" || interval == "0" {
		zap.S().Infof("Skipping operation polling task")
		return
	}
	zap.S().Infof("Adding operation polling task with interval: %s", interval)
	_, err := m.Scheduler.cronjob.AddFunc(interval, func() {
		m.CheckForNewConfiguration()
	})

	if err != nil {
		zap.S().Errorf("Could not create polling task with interval [%s]. %s", interval, err)
	}
}
