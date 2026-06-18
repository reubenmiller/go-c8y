package plugins

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/matcher"
	"resty.dev/v3"
)

var (
	ApiPlugins = "/application/applications"
	ApiPlugin  = "/application/applications/{id}"
)

var ParamID = "id"

const ResultProperty = "applications"
const CumulocityUIManifestFile = "cumulocity.json"
const ApplicationTagLatest = "latest"
const ApplicationTypeHosted = "HOSTED"

type Service struct {
	core.Service

	// Resolver lookup function (name -> id), scoped to UI plugins
	lookupByName    func(ctx context.Context, name string) (string, map[string]any, error)
	customResolvers map[string]source.Resolver
}

func NewService(s *core.Service) *Service {
	service := &Service{
		Service:         *s,
		customResolvers: make(map[string]source.Resolver),
	}

	// Setup lookup function for name-based resolution.
	//
	// A UI plugin is a HOSTED application that has versions, so the lookup is
	// scoped to that subtype (mirroring the v1 uiplugin fetcher). The reference
	// is matched against both the plugin name and its contextPath, since the
	// contextPath is commonly used to reference a plugin.
	service.lookupByName = func(ctx context.Context, name string) (string, map[string]any, error) {
		opts := ListOptions{
			Type: ApplicationTypeHosted,
			PaginationOptions: pagination.PaginationOptions{
				MaxItems: 2000,
			},
		}

		it := service.ListAll(ctx, opts)
		if it.Err() != nil {
			return "", nil, it.Err()
		}

		for item := range it.Items() {
			matchedName, _ := matcher.MatchWithWildcards(item.Name(), name)
			matchedPath, _ := matcher.MatchWithWildcards(item.ContextPath(), name)
			if matchedName || matchedPath {
				return item.ID(), map[string]any{
					"id":   item.ID(),
					"name": item.Name(),
					"type": item.Type(),
				}, nil
			}
		}

		return "", nil, core.ErrNotFound("ui plugin not found with name: %s", name)
	}

	return service
}

// Plugin represents a UI plugin/extension
type Plugin struct {
	ID              string    `json:"id,omitempty"`
	Name            string    `json:"name,omitempty"`
	Key             string    `json:"key,omitempty"`
	Type            string    `json:"type,omitempty"`
	ContextPath     string    `json:"contextPath,omitempty"`
	Availability    string    `json:"availability,omitempty"`
	ActiveVersionID string    `json:"activeVersionId,omitempty"`
	Manifest        *Manifest `json:"manifest,omitempty"`
}

type Manifest struct {
	Package   string `json:"package,omitempty"`
	IsPackage *bool  `json:"isPackage,omitempty"`
}

func (m *Manifest) WithIsPackage(v bool) *Manifest {
	m.IsPackage = &v
	return m
}

func (m *Manifest) WithPackage(v string) *Manifest {
	m.Package = v
	return m
}

// ManifestFile represents the contents of cumulocity.json in a plugin zip
type ManifestFile struct {
	Name        string `json:"name,omitempty"`
	Key         string `json:"key,omitempty"`
	ContextPath string `json:"contextPath,omitempty"`
	Package     string `json:"package,omitempty"`
	IsPackage   bool   `json:"isPackage,omitempty"`
	Version     string `json:"version,omitempty"`

	Author                  string              `json:"author"`
	Description             string              `json:"description,omitempty"`
	License                 string              `json:"license"`
	Remotes                 map[string][]string `json:"remotes"`
	RequiredPlatformVersion string              `json:"requiredPlatformVersion"`
}

type ListOptions struct {
	Name         string `url:"name,omitempty"`
	Owner        string `url:"owner,omitempty"`
	Availability string `url:"availability,omitempty"`
	ProvidedFor  string `url:"providedFor,omitempty"`
	Subscriber   string `url:"subscriber,omitempty"`
	Tenant       string `url:"tenant,omitempty"`
	Type         string `url:"type,omitempty"`
	User         string `url:"user,omitempty"`
	HasVersions  bool   `url:"hasVersions,omitempty"`
	pagination.PaginationOptions
}

type CreateOptions struct {
	Plugin         *Plugin
	SkipActivation bool
	Version        *Version
}

type Version struct {
	Version string   `json:"version,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

type PluginIterator = pagination.Iterator[jsonmodels.UIPlugin]

// NewPlugin creates a new UI plugin with default settings
func NewPlugin(name string) *Plugin {
	isPackage := true
	return &Plugin{
		Name:        name,
		Key:         name + "-key",
		ContextPath: name,
		Type:        ApplicationTypeHosted,
		Manifest: &Manifest{
			Package:   "plugin",
			IsPackage: &isPackage,
		},
	}
}

// GetManifestContents reads the cumulocity.json manifest from a zip file
func GetManifestContents(zipFilename string, contents any) error {
	reader, err := zip.OpenReader(zipFilename)
	if err != nil {
		return err
	}
	defer reader.Close()

	for _, file := range reader.File {
		if strings.EqualFold(file.Name, CumulocityUIManifestFile) {
			rc, err := file.Open()
			if err != nil {
				return err
			}
			defer rc.Close()

			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				return err
			}

			if err := json.Unmarshal(buf.Bytes(), &contents); err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("manifest file %s not found in zip", CumulocityUIManifestFile)
}

// NewPluginFromFile creates a Plugin from a zip file by reading its manifest
func (s *Service) NewPluginFromFile(filename string) (*Plugin, error) {
	manifestFile := &ManifestFile{}
	if err := GetManifestContents(filename, manifestFile); err != nil {
		return nil, err
	}

	plugin := &Plugin{
		Name:        manifestFile.Name,
		Key:         manifestFile.Key,
		Type:        ApplicationTypeHosted,
		ContextPath: manifestFile.ContextPath,
		Manifest:    &Manifest{},
	}
	plugin.Manifest.WithIsPackage(manifestFile.IsPackage)
	plugin.Manifest.WithPackage(manifestFile.Package)

	return plugin, nil
}

// HasTag checks if a tag exists in a list of tags
func HasTag(tags []string, tag string) bool {
	for _, v := range tags {
		if strings.EqualFold(v, tag) {
			return true
		}
	}
	return false
}

// List returns UI plugins with version support
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.UIPlugin] {
	opt.HasVersions = true
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewUIPlugin)
}

func (s *Service) ListAll(ctx context.Context, opts ListOptions) *PluginIterator {
	opts.HasVersions = true
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.UIPlugin] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewUIPlugin,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiPlugins)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get retrieves a UI plugin by ID or resolver string
// Examples:
//   - Get(ctx, "12345") - direct ID
//   - Get(ctx, "name:my-plugin") - lookup by name or contextPath
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.UIPlugin] {
	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(ctxhelpers.ResolutionContext(ctx), id, meta)
	if err != nil {
		return op.Failed[jsonmodels.UIPlugin](err, false)
	}
	return core.Execute(ctx, s.getB(resolvedID), jsonmodels.NewUIPlugin, meta)
}

func (s *Service) getB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, id).
		SetURL(ApiPlugin)
	return core.NewTryRequest(s.Client, req)
}

// Create creates a new UI plugin
func (s *Service) Create(ctx context.Context, plugin *Plugin) op.Result[jsonmodels.UIPlugin] {
	return core.Execute(ctx, s.createB(plugin), jsonmodels.NewUIPlugin)
}

func (s *Service) createB(body *Plugin) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiPlugins)
	return core.NewTryRequest(s.Client, req)
}

// Update updates a UI plugin's properties by ID or resolver string.
// The body may be a *Plugin or any raw JSON-serializable value (e.g. a
// map[string]any / json.RawMessage built CLI-side from --data/--template).
func (s *Service) Update(ctx context.Context, id string, body any) op.Result[jsonmodels.UIPlugin] {
	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(ctxhelpers.ResolutionContext(ctx), id, meta)
	if err != nil {
		return op.Failed[jsonmodels.UIPlugin](err, false)
	}
	return core.Execute(ctx, s.updateB(resolvedID, body), jsonmodels.NewUIPlugin, meta)
}

func (s *Service) updateB(id string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, id).
		SetBody(body).
		SetURL(ApiPlugin)
	return core.NewTryRequest(s.Client, req)
}

// Delete deletes a UI plugin by ID or resolver string
func (s *Service) Delete(ctx context.Context, id string) op.Result[core.NoContent] {
	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(ctxhelpers.ResolutionContext(ctx), id, meta)
	if err != nil {
		if core.IsNotFound(err) {
			return op.Skipped(core.NoContent{}, "not found")
		}
		return op.Failed[core.NoContent](err, false)
	}
	return core.ExecuteNoContent(ctx, s.deleteB(resolvedID), meta).IgnoreNotFound()
}

func (s *Service) deleteB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, id).
		SetURL(ApiPlugin)
	return core.NewTryRequest(s.Client, req)
}

// Activate sets a specific version as the active version for a plugin
func (s *Service) Activate(ctx context.Context, appID string, binaryID string) op.Result[jsonmodels.UIPlugin] {
	return core.Execute(ctx, s.updateB(appID, &Plugin{ActiveVersionID: binaryID}), jsonmodels.NewUIPlugin)
}

// nameResolver looks up a UI plugin by name (or contextPath)
type nameResolver struct {
	Name   string
	Lookup func(ctx context.Context, name string) (string, map[string]any, error)
}

func (n nameResolver) ResolveID(ctx context.Context) (source.ResolveResult, error) {
	if n.Lookup == nil {
		return source.ResolveResult{}, fmt.Errorf("no lookup function configured for ui plugin name")
	}
	id, meta, err := n.Lookup(ctx, n.Name)
	if err != nil {
		return source.ResolveResult{}, err
	}
	if meta == nil {
		meta = make(map[string]any)
	}
	meta["namePattern"] = n.Name
	meta["source"] = "name"
	return source.ResolveResult{ID: id, Meta: meta}, nil
}

func (n nameResolver) String() string {
	return fmt.Sprintf("name:%s", n.Name)
}

// parseResolver parses a UI plugin reference string into a resolver.
// Supported formats:
//   - "12345" -> direct ID
//   - "id:12345" -> direct ID
//   - "name:my-plugin" -> name/contextPath lookup
//   - "custom:..." -> custom resolver (if registered)
func (s *Service) parseResolver(str string) (source.Resolver, error) {
	if str == "" {
		return nil, fmt.Errorf("empty ui plugin reference string")
	}

	idx := strings.IndexByte(str, ':')
	if idx == -1 {
		// No prefix, treat as direct ID
		return source.ID(str), nil
	}

	prefix := str[:idx]
	value := str[idx+1:]

	if s.customResolvers != nil {
		if resolver, ok := s.customResolvers[prefix]; ok {
			return resolver, nil
		}
	}

	switch prefix {
	case "id":
		return source.ID(value), nil
	case "name":
		return nameResolver{
			Name:   value,
			Lookup: s.lookupByName,
		}, nil
	default:
		return nil, fmt.Errorf("unknown ui plugin resolver scheme: %s", prefix)
	}
}

// ByName creates a name-based reference string for UI plugin lookup.
// The actual lookup is performed when this string is resolved via ResolveID.
func (s *Service) ByName(name string) string {
	return fmt.Sprintf("name:%s", name)
}

// ResolveID resolves a UI plugin ID string that may contain a resolver scheme.
// Plain IDs pass through unchanged (so it is safe under --dry); "name:" refs
// trigger a lookup. Any resolved metadata is merged into meta.
func (s *Service) ResolveID(ctx context.Context, id string, meta map[string]any) (string, error) {
	resolver, err := s.parseResolver(id)
	if err != nil {
		return "", err
	}
	result, err := resolver.ResolveID(ctx)
	if err != nil {
		return "", err
	}
	if meta != nil {
		for k, v := range result.Meta {
			meta[k] = v
		}
	}
	return result.ID, nil
}

// RegisterResolver allows registering custom ID resolvers for use with ResolveID
func (s *Service) RegisterResolver(scheme string, resolver source.Resolver) {
	s.customResolvers[scheme] = resolver
}
