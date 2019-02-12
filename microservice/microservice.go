package microservice

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"go.uber.org/zap"
	"gopkg.in/robfig/cron.v2"

	"github.com/reubenmiller/go-c8y"
)

// NewDefaultMicroservice returns a new microservice instance.
// The bootstrap user will be automatically read from the environment variables
// In addition it will read the configuration, and set configure the zap logger
// NOTE:
// The microservice agent will not be registered automatically, you need to call
// RegisterMicroserviceAgent(). Though before you call it be sure to set your default
// configuration values in the Config.SetDefault()
//
func NewDefaultMicroservice() *Microservice {
	ConfigureLogger(nil, "microservice_bootstrap.log")

	// Read the configuration
	config := NewConfiguration()
	config.InitConfiguration()

	ms := &Microservice{
		Config:    config,
		Scheduler: NewScheduler(),
	}

	// Init logger using default log.file value provided in settings
	ms.InitializeLogger()

	// Create a Cumulocity client
	client := c8y.NewClientUsingBootstrapUserFromEnvironment(nil, config.GetHost())
	client.UseTenantInUsername = true
	ms.Client = client

	// Test the Cumulocity Client
	if err := ms.TestClientConnection(); err != nil {
		zap.S().Errorf("Cumulocity client failed to connect to client. If you are running this microservice locally, are you sure you set the bootstrap user credentials correctly?. Error: %s", err)
	}

	// Register the agent. Don't register the application
	return ms
}

// NewMicroservice create a new microservice where the user can customise the http client and the host
// This functin will not initiliase the logger (.InitializeLogger()) nor call .Config.InitConfiguration(), it
// is up to the user to call this functions
func NewMicroservice(httpClient *http.Client, host string) *Microservice {
	return &Microservice{
		Client:    c8y.NewClientUsingBootstrapUserFromEnvironment(httpClient, host),
		Config:    NewConfiguration(),
		Scheduler: NewScheduler(),
	}
}

// InitializeLogger starts the logger with custom configuration
func (m *Microservice) InitializeLogger(logfile ...string) {
	logfilepath := ""

	if len(logfile) > 1 {
		panic("Only a max of one log file is supported.")
	} else if len(logfile) == 1 && logfile[0] != "" {
		logfilepath = logfile[0]
	} else if m.Config != nil {
		logfilepath = m.Config.viper.GetString("log.file")
	}

	ConfigureLogger(m.Logger, logfilepath)
}

// Microservice contains information and
type Microservice struct {
	Config           *Configuration
	Client           *c8y.Client
	AgentID          string
	MicroserviceHost string
	Scheduler        *Scheduler
	Logger           *zap.Logger
}

// Scheduler to control cronjob tasks
type Scheduler struct {
	cronjob *cron.Cron
}

// NewScheduler creates a new scheduler to control cronjob tasks
func NewScheduler() *Scheduler {
	return &Scheduler{
		cronjob: cron.New(),
	}
}

// Start the scheduler so all configured cronjob tasks will run at their defined intervals
func (s *Scheduler) Start() {
	s.cronjob.Start()
}

// Stop the schedule. No more cronjob tasks will be triggered until Start() is called again. All of the job definitions will still be defined
func (s *Scheduler) Stop() {
	s.cronjob.Stop()
}

// AddFunc adds a task to a the scheduler at a specified interval
func (s *Scheduler) AddFunc(spec string, cmd func()) error {
	id, err := s.cronjob.AddFunc(spec, cmd)
	if err != nil {
		zap.S().Errorf("Could not create task scheduler. spec='%s', err='%s'", spec, err)
		return err
	}
	zap.S().Infof("Added task [id=%v, schedule=%s] to scheduler", id, spec)
	return nil
}

// TestClientConnection tests if the microservice client has connection to the Cumulocty host
func (m *Microservice) TestClientConnection() error {
	// Print out the service users
	for _, user := range m.Client.ServiceUsers {
		zap.S().Infof("user: %s, tenant: %s, password: ******************", user.Username, user.Tenant)
	}

	_, _, err := m.Client.Inventory.GetDevices(
		m.WithServiceUser(),
		&c8y.PaginationOptions{
			PageSize: 5,
		},
	)

	if err != nil {
		zap.S().Errorf("Could not get a list of devices. %s", err)
	}
	return err
}

/* Getters, Setters */

// SetMicroserviceHost sets the microservice host to a non-default Cumulocity host
// Useful when the microservice is not deployed in Cumulocity (i.e. local host, or external docker server)
func (m *Microservice) SetMicroserviceHost(host string) {
	_, err := url.Parse(host)
	if err != nil {
		panic(fmt.Errorf("Invalid microservice host. %s", err))
	}
	m.MicroserviceHost = host
}

/* Contexts */

// WithServiceUser returns the default service user (i.e. the first tenant).
// Can be used when using a PER_TENANT microservice as there will only ever be one tenant
func (m *Microservice) WithServiceUser(tenant ...string) context.Context {
	if len(tenant) > 1 {
		panic(fmt.Errorf("Context only accepts 1 tenant"))
	}
	if len(tenant) == 0 {
		return m.Client.Context.ServiceUserContext("", true)
	}
	return m.Client.Context.ServiceUserContext(tenant[0], true)
}
