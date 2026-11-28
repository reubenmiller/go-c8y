package microservice

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pipeline"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
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
//	ms, err := microservice.New(microservice.Options{})
//	if err != nil {
//		log.Fatal(err)
//	}
//	if err := ms.Bootstrap(context.Background()); err != nil {
//		log.Fatal(err)
//	}
//
// It returns an error when the host is missing or not a valid URL.
func New(opts Options) (*Microservice, error) {
	config := NewConfiguration()
	config.InitConfiguration()

	// c8y_Configuration is always included so operators can update the config
	supportedOperations := AgentSupportedOperations{"c8y_Configuration"}
	supportedOperations.AddOperations(opts.SupportedOperations)

	ms := &Microservice{
		Config:              config,
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
	client, err := NewBootstrapClient(bootstrapOpts)
	if err != nil {
		return nil, err
	}
	ms.Client = client
	ms.installClientMiddleware()

	return ms, nil
}

// Bootstrap loads the current application metadata and the service users for
// all subscribed tenants. It should be called once after New() before serving
// requests. The returned error is non-nil when the service users could not be
// loaded (e.g. wrong bootstrap credentials); a failure to read the application
// metadata is logged but not fatal.
func (m *Microservice) Bootstrap(ctx context.Context) error {
	// Retrieve current application metadata.
	currentApplication := m.Client.Microservices.Current.Get(ctx)
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

// NewMicroservice creates a Microservice with a custom host and optional client
// option overrides. Unlike New it does not call Config.InitConfiguration() or
// RefreshServiceUsers() – the caller is responsible for doing so.
// Bootstrap credentials are read from the C8Y_BOOTSTRAP_* environment variables.
// It returns an error when the host is missing or not a valid URL.
func NewMicroservice(host string, clientOpts ...func(*api.ClientOptions)) (*Microservice, error) {
	opts := api.ClientOptions{}
	for _, o := range clientOpts {
		o(&opts)
	}
	if host != "" {
		opts.BaseURL = host
	}
	client, err := NewBootstrapClient(opts)
	if err != nil {
		return nil, err
	}

	ms := &Microservice{
		Client:              client,
		Config:              NewConfiguration(),
		RealtimeClientCache: NewRealtimeClientCache(host),
	}
	ms.installClientMiddleware()
	return ms, nil
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
// It returns an error when the base URL is missing or not a valid URL.
func NewBootstrapClient(opts api.ClientOptions) (*api.Client, error) {
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
	return api.NewClient(opts)
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

	SupportedOperations AgentSupportedOperations
	AgentInformation    AgentInformation
	Hooks               Hooks
	RealtimeClientCache *RealtimeClientCache

	mu sync.RWMutex // protects ServiceUsers

	// callerCache caches validated callers (see TenantContextOptions.Validation).
	callerCache     *callerCache
	callerCacheOnce sync.Once

	// SettingsTTL is how long Settings caches each tenant's settings.
	// Default DefaultSettingsTTL (10 minutes).
	SettingsTTL time.Duration

	// settingsCache maps tenant -> settingsEntry.
	settingsCache sync.Map

	// watching is true while WatchSubscriptions runs.
	watching atomic.Bool

	// lastRefresh rate-limits refreshes triggered by 401 responses (unix nanos).
	lastRefresh atomic.Int64
}

// RefreshServiceUsers fetches the current list of subscribed service users from
// /application/currentApplication/subscriptions using the bootstrap client.
// Existing cached per-tenant clients are invalidated and will be recreated
// on the next call to TenantClient().
func (m *Microservice) RefreshServiceUsers(ctx context.Context) error {
	users, err := m.fetchServiceUsers(ctx)
	if err != nil {
		return err
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
	// after the service users were last loaded). While WatchSubscriptions runs
	// it keeps the cache current, so a miss means the tenant is not subscribed.
	if m.Client == nil || m.watching.Load() {
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
// Without opts it runs one tenant at a time and returns the first error from
// fn. Pass pipeline.Options to run tenants concurrently (Workers) and control
// error handling (MaxErrors, ErrorThreshold, OnError).
//
// Example – periodically count devices per tenant:
//
//	ms.ForEachTenant(ctx, func(ctx context.Context, user model.ServiceUser) error {
//		result := ms.Client.Devices.List(ctx, devices.ListOptions{})
//		return result.Err
//	})
func (m *Microservice) ForEachTenant(ctx context.Context, fn func(ctx context.Context, user model.ServiceUser) error, opts ...pipeline.Options) error {
	m.mu.RLock()
	users := make([]model.ServiceUser, len(m.ServiceUsers))
	copy(users, m.ServiceUsers)
	m.mu.RUnlock()

	if len(opts) > 0 {
		return pipeline.ForEach(ctx, pipeline.FromSlice(users), opts[0], func(ctx context.Context, user model.ServiceUser) error {
			return fn(api.WithServiceUser(ctx, user), user)
		})
	}

	// Default: one tenant at a time, stop at the first error.
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
	client, err := realtime.NewClient(nil, realtime.ClientOptions{
		Host:     s.host,
		Tenant:   user.Tenant,
		Username: user.Username,
		Password: user.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create realtime client: %w", err)
	}
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

// SetHost overrides the host used for outbound requests, for example when the
// microservice is not deployed in Cumulocity (local development). It returns an
// error, leaving the host unchanged, when host is not a valid URL.
func (m *Microservice) SetHost(host string) error {
	if _, err := url.Parse(host); err != nil {
		return fmt.Errorf("invalid microservice host: %w", err)
	}
	m.MicroserviceHost = host
	return nil
}

// WithBootstrapUserCredentials returns the bootstrap user credentials stored on
// the microservice bootstrap client.
func (m *Microservice) WithBootstrapUserCredentials() model.ServiceUser {
	return model.ServiceUser{
		Tenant:   m.Client.Auth().Tenant,
		Username: m.Client.Auth().Username,
		Password: m.Client.Auth().Password,
	}
}

// WithServiceUserCredentials returns the service-user credentials for the given
// tenant. If no tenant is specified the first available service user is returned.
// Only the first tenant argument is used; further arguments are ignored (this
// used to panic).
func (m *Microservice) WithServiceUserCredentials(tenant ...string) model.ServiceUser {
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
