package microservice

import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
)

// DefaultSettingsTTL is how long per-tenant settings are cached by default
// (the same as the Java SDK).
const DefaultSettingsTTL = 10 * time.Minute

type settingsEntry struct {
	settings jsonmodels.MicroserviceSettings
	expires  time.Time
}

// Settings returns the microservice's settings as seen by the tenant in ctx
// (GET /application/currentApplication/settings): subscribers can override
// settings the owner marked editable. The request uses the tenant's service
// user. Results are cached per tenant for SettingsTTL (default 10 minutes)
// and dropped when WatchSubscriptions sees the tenant unsubscribe.
//
// Keys starting with "credentials." are returned as the platform sends them;
// it decrypts them only for the owning service user.
func (m *Microservice) Settings(ctx context.Context) (jsonmodels.MicroserviceSettings, error) {
	tenant := TenantFromContext(ctx)
	if v, ok := m.settingsCache.Load(tenant); ok {
		e := v.(settingsEntry)
		if time.Now().Before(e.expires) {
			return e.settings, nil
		}
	}

	reqCtx := ctx
	if auth, ok := authentication.AuthFromContext(ctx); !ok || !m.isServiceUser(auth) {
		if tenant != "" {
			reqCtx = m.WithServiceUser(ctx, tenant)
		}
	}
	res := m.Client.Microservices.Current.ListSettings(reqCtx)
	if res.Err != nil {
		return jsonmodels.MicroserviceSettings{}, res.Err
	}
	ttl := m.SettingsTTL
	if ttl <= 0 {
		ttl = DefaultSettingsTTL
	}
	m.settingsCache.Store(tenant, settingsEntry{settings: res.Data, expires: time.Now().Add(ttl)})
	return res.Data, nil
}

// isServiceUser reports whether auth are the cached service-user credentials
// of its tenant.
func (m *Microservice) isServiceUser(auth authentication.AuthOptions) bool {
	u, ok := m.lookupServiceUser(auth.Tenant)
	return ok && auth.Tenant != "" && u.Username == auth.Username && u.Password == auth.Password
}
