// Package applicationplugins manages the UI plugins installed in a hosted
// application.
//
// The installed plugins are the application's config.remotes: a map from
// "<plugin contextPath>@<version>" to the list of the plugin's modules, e.g.
//
//	{"config": {"remotes": {"my-plugin@1.2.0": ["MyPluginModule"]}}}
//
// Entries whose key has no "@<version>" are not plugins managed here and are
// always kept as they are.
package applicationplugins

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/ui/plugins"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

var (
	ApiApplication = "/application/applications/{id}"
)

var ParamID = "id"

// VersionLatest selects the newest version of a plugin (its "latest" tag).
const VersionLatest = "latest"

const availabilityShared = "SHARED"

type Service struct {
	svc          httpcore.Service
	applications *applications.Service
	plugins      *plugins.Service
}

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:          *s,
		applications: applications.NewService(s),
		plugins:      plugins.NewService(s),
	}
}

// InstalledPlugin is one plugin installed in an application.
type InstalledPlugin struct {
	ContextPath string
	Version     string
	Modules     []string
}

// Key is the config.remotes key of the plugin: "<contextPath>@<version>".
func (p InstalledPlugin) Key() string {
	return p.ContextPath + "@" + p.Version
}

// ApplicationWithPlugins is a hosted application.
type ApplicationWithPlugins struct {
	jsondoc.Facade
}

func NewApplicationWithPlugins(data []byte) ApplicationWithPlugins {
	return ApplicationWithPlugins{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (a ApplicationWithPlugins) ID() string {
	return a.Get("id").String()
}

// Plugins returns the plugins installed in the application (the
// "<contextPath>@<version>" entries of config.remotes), sorted by key.
func (a ApplicationWithPlugins) Plugins() []InstalledPlugin {
	var out []InstalledPlugin
	a.Get("config.remotes").ForEach(func(key, value gjson.Result) bool {
		contextPath, version, ok := strings.Cut(key.String(), "@")
		if !ok {
			return true
		}
		p := InstalledPlugin{ContextPath: contextPath, Version: version, Modules: []string{}}
		for _, m := range value.Array() {
			if name := m.String(); name != "" {
				p.Modules = append(p.Modules, name)
			}
		}
		out = append(out, p)
		return true
	})
	slices.SortFunc(out, func(a, b InstalledPlugin) int { return strings.Compare(a.Key(), b.Key()) })
	return out
}

// get fetches the application. applicationID may be a name ("name:...").
// The lookup is sent also in a dry run.
func (s *Service) get(ctx context.Context, applicationID string) op.Result[ApplicationWithPlugins] {
	lookupCtx := ctxhelpers.ResolutionContext(ctx)
	id, err := s.applications.ResolveID(lookupCtx, applicationID, nil)
	if err != nil {
		return op.Failed[ApplicationWithPlugins](err, false)
	}
	return httpcore.Execute(lookupCtx, s.getB(id), NewApplicationWithPlugins)
}

func (s *Service) getB(applicationID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, applicationID).
		SetURL(ApiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) updateB(applicationID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, applicationID).
		SetBody(body).
		SetURL(ApiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// List returns the application; its Plugins method lists the installed
// plugins. ListPlugins returns them with the details of each plugin.
func (s *Service) List(ctx context.Context, applicationID string) op.Result[ApplicationWithPlugins] {
	return s.get(ctx, applicationID)
}

// PluginDetails describes a plugin installed in an application, with the
// plugin application it refers to.
type PluginDetails struct {
	ID          string         `json:"id,omitempty"`
	Name        string         `json:"name,omitempty"`
	Modules     []string       `json:"modules,omitempty"`
	ContextPath string         `json:"contextPath,omitempty"`
	Description string         `json:"description,omitempty"`
	Version     string         `json:"version,omitempty"`
	Plugin      map[string]any `json:"plugin,omitempty"`
}

// ListPlugins lists the plugins installed in an application with the details
// of each plugin (its id, name, description and the plugin application). The
// result is a collection: one document per plugin. A plugin that no longer
// exists is left out and logged at warn level.
func (s *Service) ListPlugins(ctx context.Context, applicationID string) op.Result[jsondoc.JSONDoc] {
	app := s.get(ctx, applicationID)
	if app.Err != nil {
		return op.Result[jsondoc.JSONDoc]{Err: app.Err, Status: app.Status, HTTPStatus: app.HTTPStatus}
	}
	catalog, err := s.catalog(ctx)
	if err != nil {
		return op.Failed[jsondoc.JSONDoc](err, false)
	}
	details := []PluginDetails{}
	for _, installed := range app.Data.Plugins() {
		plugin, ok := catalog.find(installed.ContextPath)
		if !ok {
			slog.Warn("could not find plugin", "contextPath", installed.ContextPath, "version", installed.Version)
			continue
		}
		raw := map[string]any{}
		_ = json.Unmarshal(plugin.Bytes(), &raw)
		details = append(details, PluginDetails{
			ID:          plugin.ID(),
			Name:        plugin.Name(),
			Modules:     installed.Modules,
			ContextPath: plugin.ContextPath(),
			Description: plugin.Get("manifest.description").String(),
			Version:     installed.Version,
			Plugin:      raw,
		})
	}
	b, err := json.Marshal(details)
	if err != nil {
		return op.Failed[jsondoc.JSONDoc](err, false)
	}
	return op.OK(jsondoc.New(b))
}

// UpdateOptions are the changes to the plugins of an application. Plugins
// are given as "<name or contextPath>[@<version or tag>]"; without a version
// the latest one is used.
type UpdateOptions struct {
	// Install adds these plugins, or changes the version of installed ones.
	Install []string

	// Remove removes these plugins (any version).
	Remove []string

	// RemoveAll removes all installed plugins first (e.g. to replace them
	// with Install).
	RemoveAll bool

	// UpdateAll moves every installed plugin to its latest version.
	UpdateAll bool

	// RemoveInvalid removes installed plugins whose plugin or version no
	// longer exists, instead of failing.
	RemoveInvalid bool

	// Body holds more application fields to change, as a JSON object (e.g.
	// {"config": {"excludedRemotes": [...]}}). Objects are merged into the
	// application's current values. Entries of config.remotes are added to
	// the plugins as given (to add a remote by hand).
	Body any
}

// Update changes the plugins installed in an application and writes the
// application's config back with them. The application and the plugins are
// looked up first (also in a dry run, so the update request is complete).
// Every other config field and every config.remotes entry that is not a
// plugin ("<key>" without "@<version>") is kept.
func (s *Service) Update(ctx context.Context, applicationID string, opt UpdateOptions) op.Result[ApplicationWithPlugins] {
	app := s.get(ctx, applicationID)
	if app.Err != nil {
		return app
	}
	catalog, err := s.catalog(ctx)
	if err != nil {
		return op.Failed[ApplicationWithPlugins](err, false)
	}

	remotes := map[string]InstalledPlugin{} // by contextPath
	var install []string
	if !opt.RemoveAll {
		for _, p := range app.Data.Plugins() {
			switch {
			case opt.UpdateAll:
				install = append(install, p.ContextPath+"@"+VersionLatest)
			case opt.RemoveInvalid:
				install = append(install, p.Key()) // validated below
			default:
				remotes[p.ContextPath] = p
			}
		}
	}
	install = append(install, opt.Install...)

	var errs []error
	for _, ref := range install {
		name, version, _ := strings.Cut(ref, "@")
		plugin, ok := catalog.find(name)
		if !ok {
			if opt.RemoveInvalid {
				slog.Warn("removing reference to orphaned plugin", "remote", ref)
				continue
			}
			errs = append(errs, fmt.Errorf("plugin does not exist. name=%s, tag/version=%s", name, version))
			continue
		}
		installed, err := resolveVersion(plugin, version)
		if err != nil {
			if opt.RemoveInvalid {
				slog.Warn("removing reference to revoked plugin version", "name", plugin.Name(), "version", version)
				continue
			}
			errs = append(errs, err)
			continue
		}
		remotes[installed.ContextPath] = installed
	}
	if len(errs) > 0 {
		return op.Failed[ApplicationWithPlugins](errors.Join(errs...), false)
	}
	for _, ref := range opt.Remove {
		name, _, _ := strings.Cut(ref, "@")
		if plugin, ok := catalog.find(name); ok {
			delete(remotes, plugin.ContextPath())
		} else {
			delete(remotes, name)
		}
	}

	body, err := applicationBody(app.Data, remotes, opt.Body)
	if err != nil {
		return op.Failed[ApplicationWithPlugins](err, false)
	}
	return httpcore.Execute(ctx, s.updateB(app.Data.ID(), body), NewApplicationWithPlugins)
}

// Install adds plugins to an application (see UpdateOptions.Install).
func (s *Service) Install(ctx context.Context, applicationID string, plugins ...string) op.Result[ApplicationWithPlugins] {
	return s.Update(ctx, applicationID, UpdateOptions{Install: plugins})
}

// Replace replaces all plugins of an application with the given ones.
func (s *Service) Replace(ctx context.Context, applicationID string, plugins ...string) op.Result[ApplicationWithPlugins] {
	return s.Update(ctx, applicationID, UpdateOptions{RemoveAll: true, Install: plugins})
}

// Delete removes plugins from an application.
func (s *Service) Delete(ctx context.Context, applicationID string, plugins ...string) op.Result[ApplicationWithPlugins] {
	return s.Update(ctx, applicationID, UpdateOptions{Remove: plugins})
}

// resolveVersion finds the version (or tag) of a plugin. An empty version is
// the latest.
func resolveVersion(plugin jsonmodels.UIPlugin, versionOrTag string) (InstalledPlugin, error) {
	if versionOrTag == "" {
		versionOrTag = VersionLatest
	}
	match := ""
	for _, v := range plugin.Get("applicationVersions").Array() {
		version := v.Get("version").String()
		if version == versionOrTag {
			match = version
			break
		}
		if slices.ContainsFunc(v.Get("tags").Array(), func(t gjson.Result) bool { return t.String() == versionOrTag }) {
			match = version
			break
		}
	}
	if match == "" {
		return InstalledPlugin{}, fmt.Errorf("plugin version not found. plugin=%s, version=%s", plugin.Name(), versionOrTag)
	}
	installed := InstalledPlugin{ContextPath: plugin.ContextPath(), Version: match, Modules: []string{}}
	plugin.Get("manifest.remotes").ForEach(func(_, modules gjson.Result) bool {
		for _, m := range modules.Array() {
			if name := m.String(); name != "" {
				installed.Modules = append(installed.Modules, name)
			}
		}
		return true
	})
	return installed, nil
}

// applicationBody builds the PUT body: the application id and config with
// extra merged in, and config.remotes set to the plugins, the entries that are
// not plugins, and the remotes given in extra.
func applicationBody(app ApplicationWithPlugins, installed map[string]InstalledPlugin, extra any) (map[string]any, error) {
	config := map[string]any{}
	if v := app.Get("config"); v.IsObject() {
		if err := json.Unmarshal([]byte(v.Raw), &config); err != nil {
			return nil, err
		}
	}
	body := map[string]any{"config": config}
	var extraRemotes map[string]any
	if extra != nil {
		b, err := json.Marshal(extra)
		if err != nil {
			return nil, err
		}
		if string(b) != "null" {
			var fields map[string]any
			if err := json.Unmarshal(b, &fields); err != nil {
				return nil, fmt.Errorf("the application fields must be a JSON object: %w", err)
			}
			if c, ok := fields["config"].(map[string]any); ok {
				extraRemotes, _ = c["remotes"].(map[string]any)
				delete(c, "remotes")
			}
			deepMerge(body, fields)
		}
	}
	config, _ = body["config"].(map[string]any)
	if config == nil {
		config = map[string]any{}
		body["config"] = config
	}

	remotes := map[string]any{}
	app.Get("config.remotes").ForEach(func(key, value gjson.Result) bool {
		if !strings.Contains(key.String(), "@") {
			remotes[key.String()] = value.Value()
		}
		return true
	})
	for _, p := range installed {
		remotes[p.Key()] = p.Modules
	}
	// entries given explicitly (e.g. a plugin added by hand) are added last
	for k, v := range extraRemotes {
		remotes[k] = v
	}
	config["remotes"] = remotes
	body["id"] = app.ID()
	return body, nil
}

// deepMerge merges src into dst: objects are merged, other values replace.
func deepMerge(dst, src map[string]any) {
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok := dst[k].(map[string]any); ok {
				deepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
}

// pluginCatalog is the UI plugins the tenant can install: its own and the
// shared ones.
type pluginCatalog struct {
	owned  []jsonmodels.UIPlugin
	shared []jsonmodels.UIPlugin
}

// find returns the plugin with the name or contextPath (case-insensitive),
// preferring the tenant's own plugins over shared ones.
func (c pluginCatalog) find(name string) (jsonmodels.UIPlugin, bool) {
	for _, list := range [][]jsonmodels.UIPlugin{c.owned, c.shared} {
		for _, p := range list {
			if strings.EqualFold(p.Name(), name) || strings.EqualFold(p.ContextPath(), name) {
				return p, true
			}
		}
	}
	return jsonmodels.UIPlugin{}, false
}

// catalog lists the plugins the tenant can use (also in a dry run).
func (s *Service) catalog(ctx context.Context) (pluginCatalog, error) {
	lookupCtx := ctxhelpers.ResolutionContext(ctx)
	var c pluginCatalog
	seen := map[string]bool{}
	for _, opt := range []plugins.ListOptions{
		{Type: plugins.ApplicationTypeHosted},
		{Availability: availabilityShared},
	} {
		opt.PaginationOptions = pagination.PaginationOptions{MaxItems: 2000}
		it := s.plugins.ListAll(lookupCtx, opt)
		for item := range it.Items() {
			if seen[item.ID()] {
				continue
			}
			seen[item.ID()] = true
			if item.Get("availability").String() == availabilityShared {
				c.shared = append(c.shared, item)
			} else {
				c.owned = append(c.owned, item)
			}
		}
		if err := it.Err(); err != nil {
			return c, err
		}
	}
	return c, nil
}
