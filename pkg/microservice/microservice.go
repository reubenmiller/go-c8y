package microservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"

	cron "gopkg.in/robfig/cron.v2"
	"resty.dev/v3"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Options contains the additional microservice options
type Options struct {
	// List of supported operations
	SupportedOperations AgentSupportedOperations
	AgentInformation    AgentInformation

	// HTTPClient is an optional custom HTTP client used to control transport, proxy, and TLS settings.
	HTTPClient *http.Client
}

// NewDefaultMicroservice returns a new microservice instance.
// Bootstrap credentials are read automatically from the C8Y_BOOTSTRAP_TENANT,
// C8Y_BOOTSTRAP_USER and C8Y_BOOTSTRAP_PASSWORD environment variables.
// The Cumulocity host is read from C8Y_BASEURL.
//
// NOTE: The microservice agent is not registered automatically. Call
// RegisterMicroserviceAgent() after setting any default configuration values
// via Config.SetDefault().
func NewDefaultMicroservice(opts Options) *Microservice {
	config := NewConfiguration()
	config.InitConfiguration()

	// c8y_Configuration is always included so operators can update the config
	supportedOperations := AgentSupportedOperations{"c8y_Configuration"}
	supportedOperations.AddOperations(opts.SupportedOperations)

	ms := &Microservice{
		Config:              config,
		Scheduler:           NewScheduler(),
		SupportedOperations: supportedOperations,
		AgentInformation:    opts.AgentInformation,
	}

	// Build a bootstrap client from C8Y_BOOTSTRAP_* environment variables.
	// This client is used for application-level calls (e.g. listing subscriptions).
	bootstrapOpts := api.ClientOptions{BaseURL: config.GetHost()}
	if opts.HTTPClient != nil {
		bootstrapOpts.Transport = opts.HTTPClient.Transport
	}
	ms.Client = NewBootstrapClient(bootstrapOpts)

	// Retrieve current application metadata.
	currentApplication := ms.Client.Microservices.CurrentMicroservice.Get(context.Background())
	if currentApplication.Err != nil {
		slog.Error("Failed to get current application information", "err", currentApplication.Err)
	} else {
		ms.Application = &currentApplication.Data
	}

	// Fetch service users (one per subscribed tenant) so per-tenant API calls
	// are available immediately.
	if err := ms.RefreshServiceUsers(context.Background()); err != nil {
		slog.Warn("Failed to load service users – are the bootstrap credentials correct?", "err", err)
	}

	// Verify connectivity using the first available service user.
	if err := ms.TestClientConnection(); err != nil {
		slog.Error("Could not connect to Cumulocity. If running locally, check bootstrap credentials.", "err", err)
	}

	ms.RealtimeClientCache = NewRealtimeClientCache(config.GetHost())

	return ms
}

// NewMicroservice creates a Microservice with a custom host and optional client
// option overrides. Unlike NewDefaultMicroservice it does not call
// Config.InitConfiguration() or RefreshServiceUsers() – the caller is
// responsible for doing so.
// Bootstrap credentials are read from the C8Y_BOOTSTRAP_* environment variables.
func NewMicroservice(host string, clientOpts ...func(*api.ClientOptions)) *Microservice {
	opts := api.ClientOptions{}
	for _, o := range clientOpts {
		o(&opts)
	}
	if host != "" {
		opts.BaseURL = host
	}
	client := NewBootstrapClient(opts)

	return &Microservice{
		Client:              client,
		Config:              NewConfiguration(),
		Scheduler:           NewScheduler(),
		RealtimeClientCache: NewRealtimeClientCache(host),
	}
}

// NewBootstrapClient returns an *api.Client pre-configured for use inside a
// Cumulocity microservice. It fills in any missing fields from the standard
// microservice environment variables:
//
//   - BaseURL  → C8Y_BASEURL
//   - Auth     → C8Y_BOOTSTRAP_TENANT / C8Y_BOOTSTRAP_USER / C8Y_BOOTSTRAP_PASSWORD
//
// Values already set in opts are never overwritten, so callers retain full
// control. The service-user auth middleware is always registered so that
// contexts produced by ServiceUserContext() work correctly with the returned
// client.
//
// Use this when you want the raw *api.Client without the full Microservice
// wrapper (no agent registration, scheduler, or configuration management).
func NewBootstrapClient(opts api.ClientOptions) *api.Client {
	if opts.BaseURL == "" {
		opts.BaseURL = microservices.GetBootstrapBaseURLFromEnvironment()
	}
	if opts.Auth.Tenant == "" && opts.Auth.Username == "" && opts.Auth.Password == "" {
		tenant, username, password := microservices.GetBootstrapUserFromEnvironment()
		opts.Auth = authentication.AuthOptions{
			Tenant:   tenant,
			Username: username,
			Password: password,
		}
	}
	client := api.NewClient(opts)
	client.UseTenantInUsername = true
	client.HTTPClient.AddRequestMiddleware(middlewareServiceUserAuth())
	return client
}

// serviceUserAuthKey is an unexported context key used to carry per-request
// service-user credentials without leaking into other packages.
type serviceUserAuthKey struct{}

// middlewareServiceUserAuth returns a resty middleware that overrides the
// client-level bootstrap credentials when a ServiceUser has been embedded in
// the request context via ServiceUserContext().
// It runs early so that resty's own auth machinery sees the header already set.
func middlewareServiceUserAuth() resty.RequestMiddleware {
	return func(_ *resty.Client, r *resty.Request) error {
		user, ok := r.Context().Value(serviceUserAuthKey{}).(model.ServiceUser)
		if !ok || user.Tenant == "" {
			return nil
		}
		// TODO: Can the basic auth tokens be exchanged for a token instead?
		r.SetAuthToken("")
		r.SetAuthScheme("Basic")
		// Set per-request basic auth using tenantID/username format.
		r.SetBasicAuth(authentication.JoinTenantUser(user.Tenant, user.Username), user.Password)
		// Also update the {tenantID} path parameter used in tenant-scoped URLs.
		r.SetPathParam("tenantID", user.Tenant)
		return nil
	}
}

// Microservice represents a running Cumulocity microservice.
type Microservice struct {
	// Application holds metadata about the microservice application.
	Application *jsonmodels.Microservice

	// Config holds the microservice configuration (backed by Viper).
	Config *Configuration

	// Client is the single shared API client authenticated with the bootstrap
	// credentials. Pass a context from ServiceUserContext() to make calls on
	// behalf of a specific tenant's service user.
	Client *api.Client

	// ServiceUsers is the cached list of tenant/service-user pairs.
	// Refresh with RefreshServiceUsers().
	ServiceUsers []model.ServiceUser

	// AgentID is the managed-object ID of this microservice's agent representation.
	AgentID string

	// MicroserviceHost overrides the host used for outbound microservice requests.
	MicroserviceHost string

	Scheduler           *Scheduler
	SupportedOperations AgentSupportedOperations
	AgentInformation    AgentInformation
	Hooks               Hooks
	RealtimeClientCache *RealtimeClientCache

	mu sync.RWMutex // protects ServiceUsers
}

// RefreshServiceUsers fetches the current list of subscribed service users from
// /application/currentApplication/subscriptions using the bootstrap client.
// Existing cached per-tenant clients are invalidated and will be recreated
// on the next call to TenantClient().
func (m *Microservice) RefreshServiceUsers(ctx context.Context) error {
	result := m.Client.Microservices.CurrentMicroservice.ListUsers(ctx)
	if result.Err != nil {
		return result.Err
	}

	var users []model.ServiceUser
	for item, err := range op.Iter2(result) {
		if err != nil {
			slog.Warn("Error reading service user", "err", err)
			continue
		}
		users = append(users, model.ServiceUser{
			Tenant:   item.Tenant(),
			Username: item.Username(),
			Password: item.Password(),
		})
	}

	m.mu.Lock()
	m.ServiceUsers = users
	m.mu.Unlock()

	slog.Info("Loaded service users", "count", len(users))
	return nil
}

// ServiceUserContext returns a context that carries the service-user credentials
// for the given tenant. Pass it to any m.Client API call to have that request
// execute as the tenant's service user instead of the bootstrap user.
//
// If no tenant is specified the first available service user is used.
// If the tenant is not found in the local cache a single refresh of
// /application/currentApplication/subscriptions is attempted before giving up.
// If still not found, a plain context.Background() is returned so the call
// proceeds with bootstrap credentials.
//
// Example – MULTI_TENANT microservice iterating all subscribed tenants:
//
//	for _, user := range ms.ServiceUsers {
//		ctx := ms.ServiceUserContext(user.Tenant)
//		result := ms.Client.Devices.List(ctx, devices.ListOptions{})
//	}
func (m *Microservice) ServiceUserContext(tenant ...string) context.Context {
	// An empty targetTenant means "any" – findServiceUserContext will return
	// the first available service user, which is the correct behaviour when
	// the caller does not specify a tenant.
	targetTenant := ""
	if len(tenant) > 0 {
		targetTenant = tenant[0]
	}
	if ctx, ok := m.findServiceUserContext(targetTenant); ok {
		return ctx
	}
	// Tenant not in cache – refresh once and retry.
	slog.Debug("Service user not found in cache, refreshing subscriptions", "tenant", targetTenant)
	if err := m.RefreshServiceUsers(context.Background()); err != nil {
		slog.Warn("Failed to refresh service users", "err", err)
	}
	if ctx, ok := m.findServiceUserContext(targetTenant); ok {
		return ctx
	}
	slog.Warn("No service user found for tenant, using bootstrap credentials", "tenant", targetTenant)
	return context.Background()
}

// findServiceUserContext searches the cached ServiceUsers list and returns a
// context containing matching credentials. The second return value is false
// when no match is found.
//
// If targetTenant is empty the first service user in the list is returned,
// which covers the common single-tenant case and the no-arg ServiceUserContext
// call.
func (m *Microservice) findServiceUserContext(targetTenant string) (context.Context, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.ServiceUsers {
		if targetTenant == "" || user.Tenant == targetTenant {
			return context.WithValue(context.Background(), serviceUserAuthKey{}, user), true
		}
	}
	return nil, false
}

// NewRealtimeClientCache returns a new realtime client cache where realtime
// clients can be reused across subscription notifications.
func NewRealtimeClientCache(host string) *RealtimeClientCache {
	return &RealtimeClientCache{
		host:    host,
		clients: map[string]*realtime.Client{},
	}
}

// RealtimeClientCache caches per-tenant realtime clients.
type RealtimeClientCache struct {
	host    string
	clients map[string]*realtime.Client
}

// NewRealtimeClient returns (or creates) a realtime.Client for the given service user.
func (m *Microservice) NewRealtimeClient(user model.ServiceUser) (*realtime.Client, error) {
	return m.RealtimeClientCache.LoadOrNewClient(user)
}

// SetClient adds a realtime client to the cache, keyed by the service user's tenant.
func (s *RealtimeClientCache) SetClient(user model.ServiceUser, client *realtime.Client) {
	s.clients[user.Tenant] = client
}

// LoadOrNewClient returns a realtime.Client for the given service user, creating
// and caching one if it does not yet exist.
func (s *RealtimeClientCache) LoadOrNewClient(user model.ServiceUser) (*realtime.Client, error) {
	if client, err := s.GetClient(user); err == nil {
		return client, nil
	}
	client := realtime.NewClient(nil, realtime.ClientOptions{
		Host:     s.host,
		Tenant:   user.Tenant,
		Username: user.Username,
		Password: user.Password,
	})
	if client != nil {
		s.SetClient(user, client)
		return client, nil
	}
	return nil, errors.New("failed to create realtime client")
}

// GetClient returns the cached realtime.Client for the given service user, or
// an error if none exists.
func (s *RealtimeClientCache) GetClient(user model.ServiceUser) (*realtime.Client, error) {
	slog.Info("Get realtime client for tenant", "tenant", user.Tenant)
	slog.Info("Total realtime clients in cache", "total", len(s.clients))
	if v, ok := s.clients[user.Tenant]; ok {
		return v, nil
	}
	return nil, errors.New("no realtime client found for tenant")
}

// Hooks contains lifecycle callback functions for the microservice.
type Hooks struct {
	OnConfigurationUpdateFunc func(Configuration)
}

// Scheduler controls cron-job tasks.
type Scheduler struct {
	cronjob *cron.Cron
}

// NewScheduler creates a new Scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		cronjob: cron.New(),
	}
}

// Start activates the scheduler so all configured tasks run at their intervals.
func (s *Scheduler) Start() {
	s.cronjob.Start()
}

// Stop pauses the scheduler. Existing job definitions are preserved.
func (s *Scheduler) Stop() {
	s.cronjob.Stop()
}

// AddFunc registers a function to run on the given cron schedule.
func (s *Scheduler) AddFunc(spec string, cmd func()) error {
	id, err := s.cronjob.AddFunc(spec, cmd)
	if err != nil {
		slog.Error("Could not create task scheduler", "spec", spec, "err", err)
		return err
	}
	slog.Info("Added task to scheduler", "id", id, "schedule", spec)
	return nil
}

// TestClientConnection verifies that the microservice can reach Cumulocity by
// listing a small number of devices using the first available service user.
func (m *Microservice) TestClientConnection() error {
	result := m.Client.Devices.List(
		m.ServiceUserContext(),
		devices.ListOptions{
			PaginationOptions: pagination.NewPaginationOptions(1),
		},
	)

	if result.Err != nil {
		slog.Error("Could not get a list of devices", "err", result.Err)
	}
	return result.Err
}

/* Getters, Setters */

// SetMicroserviceHost overrides the host used for outbound requests.
// Useful when the microservice is not deployed in Cumulocity (e.g. local dev).
func (m *Microservice) SetMicroserviceHost(host string) {
	_, err := url.Parse(host)
	if err != nil {
		panic(fmt.Errorf("invalid microservice host: %s", err))
	}
	m.MicroserviceHost = host
}

// WithBootstrapUserCredentials returns the bootstrap user credentials stored on
// the microservice bootstrap client.
func (m *Microservice) WithBootstrapUserCredentials() model.ServiceUser {
	return model.ServiceUser{
		Tenant:   m.Client.Auth.Tenant,
		Username: m.Client.Auth.Username,
		Password: m.Client.Auth.Password,
	}
}

// WithServiceUserCredentials returns the service-user credentials for the given
// tenant. If no tenant is specified the first available service user is returned.
func (m *Microservice) WithServiceUserCredentials(tenant ...string) model.ServiceUser {
	if len(tenant) > 1 {
		panic("WithServiceUserCredentials accepts at most one tenant argument")
	}
	targetTenant := ""
	if len(tenant) > 0 {
		targetTenant = tenant[0]
	}
	for _, user := range m.ServiceUsers {
		if user.Tenant == targetTenant || targetTenant == "" {
			return user
		}
	}
	return model.ServiceUser{}
}
