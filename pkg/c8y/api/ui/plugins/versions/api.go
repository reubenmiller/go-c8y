package versions

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	ApiVersions      = "/application/applications/{id}/versions"
	ApiVersionByID   = "/application/applications/{id}/versions/{version}"
	ApiVersionByName = "/application/applications/{id}/versions"
)

var ParamID = "id"
var ParamVersion = "version"

const ResultProperty = "versions"

type Service struct {
	core.Service
}

func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// Version represents a UI plugin version
type Version struct {
	Version  string   `json:"version,omitempty"`
	BinaryID string   `json:"binaryId,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}

type ListOptions struct {
	pagination.PaginationOptions
}

type CreateOptions struct {
	Version  string
	Tags     []string
	Filename string
	Reader   io.Reader
}

type VersionIterator = pagination.Iterator[jsonmodels.UIPluginVersion]

// List retrieves all versions for a specific plugin
func (s *Service) List(ctx context.Context, pluginID string, opt ListOptions) op.Result[jsonmodels.UIPluginVersion] {
	return core.ExecuteCollection(ctx, s.listB(pluginID, opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewUIPluginVersion)
}

func (s *Service) ListAll(ctx context.Context, pluginID string, opts ListOptions) *VersionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.UIPluginVersion] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, pluginID, o)
		},
		jsonmodels.NewUIPluginVersion,
	)
}

func (s *Service) listB(pluginID string, opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetPathParam(ParamID, pluginID).
		SetURL(ApiVersions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get retrieves a specific version of a plugin by version string
func (s *Service) Get(ctx context.Context, pluginID string, version string) op.Result[jsonmodels.UIPluginVersion] {
	return core.Execute(ctx, s.getB(pluginID, version), jsonmodels.NewUIPluginVersion)
}

func (s *Service) getB(pluginID string, version string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, pluginID).
		SetQueryParam("version", version).
		SetURL(ApiVersionByName)
	return core.NewTryRequest(s.Client, req)
}

// Create uploads a new version of a plugin from a file path or io.Reader
func (s *Service) Create(ctx context.Context, pluginID string, opt CreateOptions) op.Result[jsonmodels.UIPluginVersion] {
	return core.Execute(ctx, s.createB(pluginID, opt), jsonmodels.NewUIPluginVersion)
}

func (s *Service) createB(pluginID string, opt CreateOptions) *core.TryRequest {

	// build version meta information
	applicationVersion := map[string]any{}
	if opt.Version != "" {
		applicationVersion["version"] = opt.Version
	}
	if len(opt.Tags) > 0 {
		applicationVersion["tags"] = opt.Tags
	}
	fields := []*resty.MultipartField{}

	applicationVersionJSON, _ := json.Marshal(applicationVersion)
	fields = append(fields, &resty.MultipartField{
		Name:   "applicationVersion",
		Reader: bytes.NewBuffer(applicationVersionJSON),
	})

	// Add file from path or reader
	if opt.Filename != "" {
		fields = append(fields, &resty.MultipartField{
			Name:     "applicationBinary",
			FilePath: opt.Filename,
		})
	} else if opt.Reader != nil {
		fields = append(fields, &resty.MultipartField{
			Name:     "applicationBinary",
			Reader:   opt.Reader,
			FileName: "plugin.zip",
		})
	}

	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamID, pluginID).
		SetMultipartFields(fields...).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiVersions)
	return core.NewTryRequest(s.Client, req)
}

// Update updates the tags for a specific plugin version
func (s *Service) Update(ctx context.Context, pluginID string, version string, tags []string) op.Result[jsonmodels.UIPluginVersion] {
	return core.Execute(ctx, s.updateB(pluginID, version, tags), jsonmodels.NewUIPluginVersion)
}

func (s *Service) updateB(pluginID string, version string, tags []string) *core.TryRequest {
	body := map[string]any{
		"tags": tags,
	}
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, pluginID).
		SetPathParam(ParamVersion, version).
		SetBody(body).
		SetURL(ApiVersionByID)
	return core.NewTryRequest(s.Client, req)
}

// Delete removes a specific version of a plugin
func (s *Service) Delete(ctx context.Context, pluginID string, version string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(pluginID, version)).IgnoreNotFound()
}

func (s *Service) deleteB(pluginID string, version string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, pluginID).
		SetQueryParam("version", version).
		SetURL(ApiVersionByName)
	return core.NewTryRequest(s.Client, req)
}
