package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"gopkg.in/yaml.v3"
)

// serviceUserRequiredRoles are the permissions requested for the placeholder
// microservice's service users. They cover everything the sync sections do
// inside a target tenant (inventory repositories, tenant options).
var serviceUserRequiredRoles = []string{
	"ROLE_INVENTORY_READ",
	"ROLE_INVENTORY_ADMIN",
	"ROLE_INVENTORY_CREATE",
	"ROLE_OPTION_MANAGEMENT_READ",
	"ROLE_OPTION_MANAGEMENT_ADMIN",
	"ROLE_APPLICATION_MANAGEMENT_READ",
	"ROLE_TENANT_MANAGEMENT_READ",
}

// resolveTargetCredentials obtains credentials for every target that is not
// the current tenant and stores them on the targets in place
func (s *Syncer) resolveTargetCredentials(ctx context.Context, targets []Target, spec *TargetsSpec) error {
	needed := false
	for _, target := range targets {
		if !target.Current && target.Auth == nil {
			needed = true
			break
		}
	}
	if !needed {
		return nil
	}

	switch spec.CredentialsMode() {
	case CredentialsModeSessions:
		return s.resolveSessionCredentials(targets, spec)
	default:
		return s.resolveServiceUserCredentials(ctx, targets, spec)
	}
}

// resolveServiceUserCredentials provisions a placeholder microservice (owned
// by the current tenant), subscribes it to every target tenant and uses the
// resulting per-tenant service users as credentials. This is the same pattern
// go-c8y-cli uses for "microservice service user" sessions.
//
// Side effects (idempotent, left in place between runs): one MICROSERVICE
// application in the current tenant and one subscription per target tenant.
func (s *Syncer) resolveServiceUserCredentials(ctx context.Context, targets []Target, spec *TargetsSpec) error {
	name := DefaultCredentialsApplication
	if spec.Credentials != nil && spec.Credentials.Application != "" {
		name = spec.Credentials.Application
	}

	// Ensure the placeholder microservice exists with the required roles
	existing, found := s.Client.Microservices.FindFirst(ctx, microservices.ListOptions{Name: name})
	if existing.Err != nil {
		return fmt.Errorf("failed to look up credentials application %q: %w", name, existing.Err)
	}

	var appID, selfLink string
	rolesUpToDate := false
	if found {
		appID = existing.Data.ID()
		selfLink = existing.Data.Self()
		rolesUpToDate = stringSetEqual(existing.Data.RequiredRoles(), serviceUserRequiredRoles)
	} else {
		app := model.NewMicroservice(name)
		app.ContextPath = name
		created := s.Client.Microservices.Create(ctx, app)
		if created.Err != nil {
			return fmt.Errorf("failed to create credentials application %q: %w", name, created.Err)
		}
		appID = created.Data.ID()
		selfLink = created.Data.Self()
		fmt.Printf("🔑 Created credentials application %q (%s)\n", name, appID)
	}

	if !rolesUpToDate {
		update := s.Client.Microservices.Update(ctx, appID, &model.Microservice{
			RequiredRoles: serviceUserRequiredRoles,
		})
		if update.Err != nil {
			return fmt.Errorf("failed to update required roles of credentials application %q: %w", name, update.Err)
		}
	}

	// Subscribe the placeholder to every target tenant (409 = already subscribed)
	for _, target := range targets {
		if target.Current || target.Auth != nil {
			continue
		}
		result := s.Client.Microservices.Subscribe(ctx, target.TenantID, selfLink)
		if result.Err != nil {
			return fmt.Errorf("failed to subscribe credentials application %q to tenant %s: %w", name, target.TenantID, result.Err)
		}
	}

	// Fetch the per-tenant service users via the application bootstrap user
	bootstrap := s.Client.Microservices.BootstrapUser.Get(ctx, appID)
	if bootstrap.Err != nil {
		return fmt.Errorf("failed to get bootstrap user of credentials application %q: %w", name, bootstrap.Err)
	}

	bootstrapClient := api.NewClient(api.ClientOptions{
		BaseURL: s.Client.BaseURL.String(),
		Auth: authentication.AuthOptions{
			Tenant:   bootstrap.Data.Tenant(),
			Username: bootstrap.Data.Username(),
			Password: bootstrap.Data.Password(),
		},
	})
	bootstrapClient.UseTenantInUsername = true

	// Service users for freshly subscribed tenants can take a moment to
	// appear, so retry a few times before giving up
	users := make(map[string]authentication.AuthOptions)
	var missing []string
	for attempt := range 5 {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}

		result := bootstrapClient.Microservices.CurrentMicroservice.ListUsers(ctx)
		if result.Err != nil {
			return fmt.Errorf("failed to list service users of credentials application %q: %w", name, result.Err)
		}
		for user, err := range op.Iter2(result) {
			if err != nil {
				return fmt.Errorf("failed to read service users of credentials application %q: %w", name, err)
			}
			users[user.Tenant()] = authentication.AuthOptions{
				Tenant:   user.Tenant(),
				Username: user.Username(),
				Password: user.Password(),
			}
		}

		missing = missing[:0]
		for _, target := range targets {
			if target.Current || target.Auth != nil {
				continue
			}
			if _, ok := users[target.TenantID]; !ok {
				missing = append(missing, target.TenantID)
			}
		}
		if len(missing) == 0 {
			break
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("no service user available for tenant(s) %s: check that the subscription of application %q succeeded",
			strings.Join(missing, ", "), name)
	}

	for i := range targets {
		if targets[i].Current || targets[i].Auth != nil {
			continue
		}
		auth := users[targets[i].TenantID]
		targets[i].Auth = &auth
	}
	return nil
}

// goc8ycliSession is the subset of a go-c8y-cli session file used as
// credentials. Session files are JSON (YAML parses JSON, so YAML session
// files work too).
type goc8ycliSession struct {
	Path     string `yaml:"-"`
	Host     string `yaml:"host"`
	Tenant   string `yaml:"tenant"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// resolveSessionCredentials reads go-c8y-cli session files and matches them
// to the targets by tenant ID and then by domain (against the session host)
func (s *Syncer) resolveSessionCredentials(targets []Target, spec *TargetsSpec) error {
	home := ""
	if spec.Credentials != nil {
		home = spec.Credentials.SessionHome
	}
	if home == "" {
		home = os.Getenv("C8Y_SESSION_HOME")
	}
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to determine the session directory: %w", err)
		}
		home = filepath.Join(userHome, ".cumulocity")
	}

	sessions, err := loadSessionFiles(home)
	if err != nil {
		return err
	}

	var missing []string
	for i := range targets {
		if targets[i].Current || targets[i].Auth != nil {
			continue
		}
		session := matchSession(sessions, targets[i])
		if session == nil {
			missing = append(missing, targets[i].Label())
			continue
		}
		if strings.HasPrefix(session.Password, "{encrypted}") {
			return fmt.Errorf("session %s is encrypted: tenant-sync cannot decrypt go-c8y-cli sessions, store the password unencrypted or use the serviceUser credentials mode", session.Path)
		}
		targets[i].Auth = &authentication.AuthOptions{
			Tenant:   session.Tenant,
			Username: session.Username,
			Password: session.Password,
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("no go-c8y-cli session found in %s for tenant(s): %s", home, strings.Join(missing, ", "))
	}
	return nil
}

// loadSessionFiles parses all readable session files in a directory
func loadSessionFiles(dir string) ([]goc8ycliSession, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read the session directory %s: %w", dir, err)
	}

	var sessions []goc8ycliSession
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".json", ".yaml", ".yml":
		default:
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		session := goc8ycliSession{Path: filepath.Join(dir, entry.Name())}
		if err := yaml.Unmarshal(raw, &session); err != nil {
			continue
		}
		if session.Username == "" || session.Password == "" {
			continue
		}
		sessions = append(sessions, session)
	}

	sort.Slice(sessions, func(i, j int) bool { return sessions[i].Path < sessions[j].Path })
	return sessions, nil
}

// matchSession finds the session for a target: first by tenant ID, then by
// the target domain appearing as the host of the session
func matchSession(sessions []goc8ycliSession, target Target) *goc8ycliSession {
	for i := range sessions {
		if sessions[i].Tenant != "" && sessions[i].Tenant == target.TenantID {
			return &sessions[i]
		}
	}
	if target.Domain == "" {
		return nil
	}
	for i := range sessions {
		if hostMatchesDomain(sessions[i].Host, target.Domain) {
			return &sessions[i]
		}
	}
	return nil
}

// hostMatchesDomain reports whether a session host (which may include a
// scheme, port or trailing slash) refers to the given tenant domain
func hostMatchesDomain(host, domain string) bool {
	host = strings.TrimSpace(strings.ToLower(host))
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimSuffix(strings.SplitN(host, "/", 2)[0], ".")
	host = strings.SplitN(host, ":", 2)[0]
	return host == strings.ToLower(strings.TrimSuffix(domain, "."))
}

// stringSetEqual compares two string slices ignoring order and duplicates
func stringSetEqual(a, b []string) bool {
	setA := make(map[string]bool, len(a))
	for _, v := range a {
		setA[v] = true
	}
	setB := make(map[string]bool, len(b))
	for _, v := range b {
		setB[v] = true
	}
	if len(setA) != len(setB) {
		return false
	}
	for v := range setA {
		if !setB[v] {
			return false
		}
	}
	return true
}
