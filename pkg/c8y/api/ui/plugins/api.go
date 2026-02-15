package plugins

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	ApiPlugins = "/application/applications"
	ApiPlugin  = "/application/applications/{id}"
)

var ParamId = "id"

const ResultProperty = "applications"
const CumulocityUIManifestFile = "cumulocity.json"
const ApplicationTagLatest = "latest"
const ApplicationTypeHosted = "HOSTED"

type Service struct {
	core.Service
}

func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
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
	ProviderFor  string `url:"providerFor,omitempty"`
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
func GetManifestContents(zipFilename string, contents interface{}) error {
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

// Get retrieves a specific UI plugin by ID
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.UIPlugin] {
	return core.Execute(ctx, s.getB(id), jsonmodels.NewUIPlugin)
}

func (s *Service) getB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, id).
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

// Update updates a UI plugin's properties
func (s *Service) Update(ctx context.Context, id string, plugin *Plugin) op.Result[jsonmodels.UIPlugin] {
	return core.Execute(ctx, s.updateB(id, plugin), jsonmodels.NewUIPlugin)
}

func (s *Service) updateB(id string, body *Plugin) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, id).
		SetBody(body).
		SetURL(ApiPlugin)
	return core.NewTryRequest(s.Client, req)
}

// Delete deletes a UI plugin
func (s *Service) Delete(ctx context.Context, id string) (*resty.Response, error) {
	return core.ExecuteResponseOnly(ctx, s.deleteB(id))
}

func (s *Service) deleteB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, id).
		SetURL(ApiPlugin)
	return core.NewTryRequest(s.Client, req)
}

// Activate sets a specific version as the active version for a plugin
func (s *Service) Activate(ctx context.Context, appID string, binaryID string) op.Result[jsonmodels.UIPlugin] {
	return core.Execute(ctx, s.updateB(appID, &Plugin{ActiveVersionID: binaryID}), jsonmodels.NewUIPlugin)
}
