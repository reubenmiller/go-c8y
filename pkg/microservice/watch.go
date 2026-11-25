package microservice

import (
	"context"
	"log/slog"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
)

// DefaultWatchInterval is how often WatchSubscriptions polls by default.
const DefaultWatchInterval = 30 * time.Second

// Isolation modes of a microservice (C8Y_MICROSERVICE_ISOLATION).
const (
	IsolationMultiTenant = "MULTI_TENANT"
	IsolationPerTenant   = "PER_TENANT"
)

// WatchOptions configures WatchSubscriptions.
type WatchOptions struct {
	// Interval between polls. Default DefaultWatchInterval (30s).
	Interval time.Duration

	// OnTenantAdded is called for each newly subscribed tenant, including all
	// tenants found on the first poll. ctx carries the tenant's service user.
	OnTenantAdded func(ctx context.Context, user model.ServiceUser) error

	// OnTenantRemoved is called for each tenant that is no longer subscribed.
	OnTenantRemoved func(ctx context.Context, tenant string) error

	// OnCredentialsChanged is called when a still-subscribed tenant's service
	// user changed (for example after an unsubscribe/subscribe cycle). ctx
	// carries the new credentials.
	OnCredentialsChanged func(ctx context.Context, user model.ServiceUser) error

	// OnError receives polling errors and callback errors. Watching continues.
	OnError func(err error)
}

// WatchSubscriptions polls the microservice's subscriptions until ctx is
// cancelled, keeps the cached service users current, and reports changes
// through the callbacks. Callbacks run one at a time, sorted by tenant ID;
// start goroutines in them for long-running work. It returns nil when ctx is
// cancelled.
//
// While watching, GetServiceUser no longer refreshes on a cache miss, so an
// unsubscribed tenant stays unknown until it subscribes again.
//
// Each replica of a microservice watches independently.
func (m *Microservice) WatchSubscriptions(ctx context.Context, opts WatchOptions) error {
	interval := opts.Interval
	if interval <= 0 {
		interval = DefaultWatchInterval
	}
	onError := opts.OnError
	if onError == nil {
		onError = func(err error) { slog.Warn("Subscription watch error", "err", err) }
	}

	m.watching.Store(true)
	defer m.watching.Store(false)

	known := map[string]model.ServiceUser{}
	poll := func() {
		users, err := m.fetchServiceUsers(ctx)
		if err != nil {
			if ctx.Err() == nil {
				onError(err)
			}
			return
		}
		m.SetServiceUsers(users)

		current := make(map[string]model.ServiceUser, len(users))
		for _, u := range users {
			current[u.Tenant] = u
		}
		for _, tenant := range sortedTenants(current) {
			u := current[tenant]
			prev, existed := known[tenant]
			switch {
			case !existed && opts.OnTenantAdded != nil:
				if err := opts.OnTenantAdded(api.WithServiceUser(ctx, u), u); err != nil {
					onError(err)
				}
			case existed && prev != u && opts.OnCredentialsChanged != nil:
				if err := opts.OnCredentialsChanged(api.WithServiceUser(ctx, u), u); err != nil {
					onError(err)
				}
			}
		}
		for _, tenant := range sortedTenants(known) {
			if _, ok := current[tenant]; ok {
				continue
			}
			m.forgetTenant(tenant)
			if opts.OnTenantRemoved != nil {
				if err := opts.OnTenantRemoved(ctx, tenant); err != nil {
					onError(err)
				}
			}
		}
		known = current
	}

	poll()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			poll()
		}
	}
}

// fetchServiceUsers returns the service users of all subscribed tenants. In
// PER_TENANT isolation it returns the single service user from C8Y_TENANT,
// C8Y_USER and C8Y_PASSWORD without calling the platform.
func (m *Microservice) fetchServiceUsers(ctx context.Context) ([]model.ServiceUser, error) {
	if isolationMode() == IsolationPerTenant {
		return []model.ServiceUser{{
			Tenant:   os.Getenv("C8Y_TENANT"),
			Username: os.Getenv("C8Y_USER"),
			Password: os.Getenv("C8Y_PASSWORD"),
		}}, nil
	}
	result := m.Client.Microservices.Current.ListUsers(ctx)
	if result.Err != nil {
		return nil, result.Err
	}
	var users []model.ServiceUser
	for item, err := range result.Items() {
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
	return users, nil
}

// isolationMode returns C8Y_MICROSERVICE_ISOLATION, defaulting to MULTI_TENANT.
func isolationMode() string {
	if v := strings.ToUpper(strings.TrimSpace(os.Getenv("C8Y_MICROSERVICE_ISOLATION"))); v != "" {
		return v
	}
	return IsolationMultiTenant
}

// forgetTenant drops per-tenant caches for a removed tenant.
func (m *Microservice) forgetTenant(tenant string) {
	m.settingsCache.Delete(tenant)
}

func sortedTenants(users map[string]model.ServiceUser) []string {
	out := make([]string, 0, len(users))
	for t := range users {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}
