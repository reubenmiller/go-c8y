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

// New returns a new microservice instance without contacting the Cumulocity
// platform. Bootstrap credentials are read automatically from the
// C8Y_BOOTSTRAP_TENANT, C8Y_BOOTSTRAP_USER and C8Y_BOOTSTRAP_PASSWORD
// environment variables, and the Cumulocity host from C8Y_BASEURL.
//
// Call Bootstrap() afterwards to load the application metadata and the
// per-tenant service users:
//
//	ms := microservice.New(microservice.Options{})
//	if err := ms.Bootstrap(context.Background()); err != nil {
//		log.Fatal(err)
//	}
func New(opts Options) *Microservice {
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
		RealtimeClientCache: NewRealtimeClientCache(config.GetHost()),
	}

	// Build a bootstrap client from C8Y_BOOTSTRAP_* environment variables.
	// This client is used for application-level calls (e.g. listing subscriptions).
	bootstrapOpts := api.ClientOptions{BaseURL: config.GetHost()}
	if opts.HTTPClient != nil {
		bootstrapOpts.Transport = opts.HTTPClient.Transport
	}
	ms.Client = NewBootstrapClient(bootstrapOpts)

	return ms
}

// Bootstrap loads the current application metadata and the service users for
// all subscribed tenants. It should be called once after New() before serving
// requests. The returned error is non-nil when the service users could not be
// loaded (e.g. wrong bootstrap credentials); a failure to read the application
// metadata is logged but not fatal.
func (m *Microservice) Bootstrap(ctx context.Context) error {
	// Retrieve current application metadata.
	currentApplication := m.Client.Microservices.CurrentMicroservice.Get(ctx)
	if currentApplication.Err != nil {
		slog.Warn("Failed to get current application information", "err", currentApplication.Err)
	} else {
		m.Application = &currentApplication.Data
	}

	// Fetch service users (one per subscribed tenant) so per-tenant API calls
	// are available immediately.
	if err := m.RefreshServiceUsers(ctx); err != nil {
		return fmt.Errorf("failed to load service users (are the bootstrap credentials correct?): %w", err)
	}
	return nil
}

// NewDefaultMicroservice returns a new microservice instance and immediately
// bootstraps it (loads application metadata and service users) and verifies
// platform connectivity. Errors during bootstrap are logged but do not prevent
// the microservice from being returned.
//
// NOTE: The microservice agent is not registered automatically. Call
// RegisterMicroserviceAgent() after setting any default configuration values
// via Config.SetDefault().
//
// Deprecated: Use New() followed by Bootstrap() for explicit error handling.
func NewDefaultMicroservice(opts Options) *Microservice {
	ms := New(opts)

	if err := ms.Bootstrap(context.Background()); err != nil {
		slog.Warn("Failed to bootstrap microservice", "err", err)
	}

	// Verify connectivity using the first available service user.
	if err := ms.TestClientConnection(); err != nil {
		slog.Error("Could not connect to Cumulocity. If running locally, check bootstrap credentials.", "err", err)
	}

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
// control. Per-tenant calls are made by passing a context produced by
// api.WithServiceUser() (or Microservice.WithServiceUser()) to any API call.
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
	return client
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
// Use WithServiceUser to derive from an existing context (preserving
// cancellation and deadlines) instead of context.Background().
func (m *Microservice) ServiceUserContext(tenant ...string) context.Context {
	return m.WithServiceUser(context.Background(), tenant...)
}

// WithServiceUser returns a copy of the parent context carrying the
// service-user credentials for the given tenant, preserving any cancellation,
// deadline or values already present on the parent.
//
// If no tenant is specified the first available service user is used.
// If the tenant is not found in the local cache a single refresh of
// /application/currentApplication/subscriptions is attempted before giving up;
// if still not found the parent context is returned unchanged so the call
// proceeds with bootstrap credentials.
//
// Example – MULTI_TENANT microservice making a call for a specific tenant:
//
//	ctx := ms.WithServiceUser(ctx, "t12345")
//	result := ms.Client.Devices.List(ctx, devices.ListOptions{})
func (m *Microservice) WithServiceUser(parent context.Context, tenant ...string) context.Context {
	// An empty targetTenant means "any" – GetServiceUser will return the first
	// available service user, which is the correct behaviour when the caller
	// does not specify a tenant.
	targetTenant := ""
	if len(tenant) > 0 {
		targetTenant = tenant[0]
	}
	if user, ok := m.GetServiceUser(targetTenant); ok {
		return api.WithServiceUser(parent, user)
	}
	slog.Warn("No service user found for tenant, using bootstrap credentials", "tenant", targetTenant)
	return parent
}

// GetServiceUser returns the cached service user for the given tenant. When
// tenant is empty the first available service user is returned. If the tenant
// is not found in the local cache, a single refresh of the application
// subscriptions is attempted before giving up.
func (m *Microservice) GetServiceUser(tenant string) (model.ServiceUser, bool) {
	if user, ok := m.lookupServiceUser(tenant); ok {
		return user, true
	}
	// Tenant not in cache – refresh once and retry (e.g. the tenant subscribed
	// after the service users were last loaded).
	if m.Client == nil {
		return model.ServiceUser{}, false
	}
	slog.Debug("Service user not found in cache, refreshing subscriptions", "tenant", tenant)
	if err := m.RefreshServiceUsers(context.Background()); err != nil {
		slog.Warn("Failed to refresh service users", "err", err)
		return model.ServiceUser{}, false
	}
	return m.lookupServiceUser(tenant)
}

// lookupServiceUser searches the cached ServiceUsers list. When targetTenant is
// empty the first service user in the list is returned, which covers the common
// single-tenant case.
func (m *Microservice) lookupServiceUser(targetTenant string) (model.ServiceUser, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.ServiceUsers {
		if targetTenant == "" || user.Tenant == targetTenant {
			return user, true
		}
	}
	return model.ServiceUser{}, false
}

// SetServiceUsers replaces the cached service-user list. Mainly useful in tests
// or when service users are managed externally.
func (m *Microservice) SetServiceUsers(users []model.ServiceUser) {
	m.mu.Lock()
	m.ServiceUsers = users
	m.mu.Unlock()
}

// ForEachTenant runs fn once for every subscribed tenant. The context passed to
// fn carries that tenant's service-user credentials, so any m.Client API call
// made with it executes as that tenant (equivalent to the Java SDK's
// MicroserviceSubscriptionsService.runForEachTenant).
//
// The iteration stops at the first error returned by fn or when ctx is
// cancelled.
//
// Example – periodically count devices per tenant:
//
//	ms.ForEachTenant(ctx, func(ctx context.Context, user model.ServiceUser) error {
//		result := ms.Client.Devices.List(ctx, devices.ListOptions{})
//		return result.Err
//	})
func (m *Microservice) ForEachTenant(ctx context.Context, fn func(ctx context.Context, user model.ServiceUser) error) error {
	m.mu.RLock()
	users := make([]model.ServiceUser, len(m.ServiceUsers))
	copy(users, m.ServiceUsers)
	m.mu.RUnlock()

	for _, user := range users {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fn(api.WithServiceUser(ctx, user), user); err != nil {
			return err
		}
	}
	return nil
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
