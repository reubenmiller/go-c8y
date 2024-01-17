package microservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"go.uber.org/zap"
	cron "gopkg.in/robfig/cron.v2"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

// Options contains the additional microsevice options
type Options struct {
	// List of supported operations
	SupportedOperations AgentSupportedOperations
	AgentInformation    AgentInformation
}

// NewDefaultMicroservice returns a new microservice instance.
// The bootstrap user will be automatically read from the environment variables
// In addition it will read the configuration, and set configure the zap logger
// NOTE:
// The microservice agent will not be registered automatically, you need to call
// RegisterMicroserviceAgent(). Though before you call it be sure to set your default
// configuration values in the Config.SetDefault()
func NewDefaultMicroservice(opts Options) *Microservice {
	ConfigureLogger(nil, "microservice_bootstrap.log")

	// Read the configuration
	config := NewConfiguration()
	config.InitConfiguration()

	// add the c8y_Configuration operation by default
	supportedOperations := AgentSupportedOperations{"c8y_Configuration"}
	supportedOperations.AddOperations(opts.SupportedOperations)

	ms := &Microservice{
		Config:              config,
		Scheduler:           NewScheduler(),
		SupportedOperations: supportedOperations,
		AgentInformation:    opts.AgentInformation,
	}

	// Init logger using default log.file value provided in settings
	ms.InitializeLogger()

	// Create a Cumulocity client
	client := c8y.NewClientUsingBootstrapUserFromEnvironment(nil, config.GetHost(), false)
	client.UseTenantInUsername = true
	ms.Client = client

	// Get current application
	currentApplication, _, err := client.Application.GetCurrentApplication(context.Background())

	if err != nil {
		zap.S().Errorf("Failed to get current application information. %s", err)
	} else {
		ms.Application = currentApplication
	}

	// Test the Cumulocity Client
	if err := ms.TestClientConnection(); err != nil {
		zap.S().Errorf("Cumulocity client failed to connect to client. If you are running this microservice locally, are you sure you set the bootstrap user credentials correctly?. Error: %s", err)
	}

	ms.RealtimeClientCache = NewRealtimeClientCache(config.GetHost())

	// Register the agent. Don't register the application
	return ms
}

// NewMicroservice create a new microservice where the user can customize the http client and the host
// This function will not initialize the logger (.InitializeLogger()) nor call .Config.InitConfiguration(), it
// is up to the user to call these functions
func NewMicroservice(httpClient *http.Client, host string, skipRealtimeClient bool) *Microservice {
	return &Microservice{
		Client:              c8y.NewClientUsingBootstrapUserFromEnvironment(httpClient, host, skipRealtimeClient),
		Config:              NewConfiguration(),
		Scheduler:           NewScheduler(),
		RealtimeClientCache: NewRealtimeClientCache(host),
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
	Application         *c8y.Application
	Config              *Configuration
	Client              *c8y.Client
	AgentID             string
	MicroserviceHost    string
	Scheduler           *Scheduler
	Logger              *zap.Logger
	SupportedOperations AgentSupportedOperations
	AgentInformation    AgentInformation
	Hooks               Hooks
	RealtimeClientCache *RealtimeClientCache
}

// NewRealtimeClientCache returns a new realtime client cache where realtime clients can be reused for different subscription notifications
func NewRealtimeClientCache(host string) *RealtimeClientCache {
	return &RealtimeClientCache{
		host:    host,
		clients: map[string]*c8y.RealtimeClient{},
	}
}

// RealtimeClientCache is a cache to store the different realtime clients used in the microservice
type RealtimeClientCache struct {
	host    string
	clients map[string]*c8y.RealtimeClient
}

// NewRealtimeClient creates a new realtime client for the given tenant service user
func (m *Microservice) NewRealtimeClient(user c8y.ServiceUser) (*c8y.RealtimeClient, error) {
	return m.RealtimeClientCache.LoadOrNewClient(user)
}

// SetClient adds the given realtime client in the cache stored under the service user's tenant
func (s *RealtimeClientCache) SetClient(user c8y.ServiceUser, client *c8y.RealtimeClient) {
	s.clients[user.Tenant] = client
}

// LoadOrNewClient returns a Realtime client for the given service user.
// If a realtime client already exists for the given service user then it will be returned rather than creating a new client
func (s *RealtimeClientCache) LoadOrNewClient(user c8y.ServiceUser) (*c8y.RealtimeClient, error) {
	if client, err := s.GetClient(user); err == nil {
		return client, nil
	}
	client := c8y.NewRealtimeClient(s.host, nil, user.Tenant, user.Username, user.Password)
	if client != nil {
		s.SetClient(user, client)
		return client, nil
	}
	return nil, errors.New("No existing realtime clients")
}

// GetClient returns a realtime client if it already exists in the cache. If no realtime client already exists for the service user, then an error is returned
func (s *RealtimeClientCache) GetClient(user c8y.ServiceUser) (*c8y.RealtimeClient, error) {
	log.Printf("Get realtime client for tenant %s", user.Tenant)
	log.Printf("Total realtime clients in cache %d", len(s.clients))
	if v, ok := s.clients[user.Tenant]; ok {
		return v, nil
	}
	return nil, errors.New("No realtime client found for tenant")

}

// Hooks contains list of lifecycle hooks that can be used in the microservice
type Hooks struct {
	OnConfigurationUpdateFunc func(Configuration)
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

// WithBootstrapUserCredentials returns the application credentials
func (m *Microservice) WithBootstrapUserCredentials() c8y.ServiceUser {
	return c8y.ServiceUser{
		Tenant:   m.Client.TenantName,
		Username: m.Client.Username,
		Password: m.Client.Password,
	}
}

// WithServiceUserCredentials returns the service user credentials associated with the tenant. If no tenant is given, then the first service user will be returned
func (m *Microservice) WithServiceUserCredentials(tenant ...string) c8y.ServiceUser {
	if len(tenant) > 1 {
		panic(fmt.Errorf("Only accepts 1 tenant"))
	}

	tenantName := ""
	if len(tenant) > 0 {
		tenantName = tenant[0]
	}
	for _, user := range m.Client.ServiceUsers {
		if user.Tenant == tenantName || tenantName == "" {
			return user
		}
	}

	return c8y.ServiceUser{}
}
