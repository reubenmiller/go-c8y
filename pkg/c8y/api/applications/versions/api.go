package versions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"resty.dev/v3"
)

var (
	ApiVersions      = "/application/applications/{id}/versions"
	ApiVersionByID   = "/application/applications/{id}/versions/{version}"
	ApiVersionByName = "/application/applications/{id}/versions"
)

var ParamId = "id"
var ParamVersion = "version"

const ResultProperty = "applicationVersions"

// Service for managing application versions
type Service struct {
	core.Service
}

func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// ListOptions contains options for listing versions
type ListOptions struct {
	pagination.PaginationOptions
}

type UploadFileOptions = core.UploadFileOptions

// CreateOptions contains options for creating application versions
type CreateOptions struct {
	Version string   `json:"version,omitempty"`
	Tags    []string `json:"tags"`

	File UploadFileOptions `json:"-"`
}

type VersionIterator = pagination.Iterator[jsonmodels.ApplicationVersion]

// List retrieves all versions for a specific application
func (s *Service) List(ctx context.Context, applicationID string, opt ListOptions) op.Result[jsonmodels.ApplicationVersion] {
	return core.ExecuteCollection(ctx, s.listB(applicationID, opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplicationVersion)
}

func (s *Service) listB(applicationID string, opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetPathParam(ParamId, applicationID).
		SetURL(ApiVersions)

	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// ListAll returns an iterator for all versions
func (s *Service) ListAll(ctx context.Context, applicationID string, opts ListOptions) *VersionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.ApplicationVersion] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, applicationID, o)
		},
		jsonmodels.NewApplicationVersion,
	)
}

// ListByVersion retrieves a specific version of an application by version string
func (s *Service) ListByVersion(ctx context.Context, applicationID string, version string) op.Result[jsonmodels.ApplicationVersion] {
	return core.ExecuteCollection(ctx, s.listByVersionB(applicationID, version), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplicationVersion)
}

func (s *Service) listByVersionB(applicationID string, version string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, applicationID).
		SetQueryParam("version", version).
		SetURL(ApiVersionByName)

	return core.NewTryRequest(s.Client, req)
}

// ListByTag retrieves a specific version of an application by tag
func (s *Service) ListByTag(ctx context.Context, applicationID string, tag string) op.Result[jsonmodels.ApplicationVersion] {
	return core.ExecuteCollection(ctx, s.listByTagB(applicationID, tag), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewApplicationVersion)
}

func (s *Service) listByTagB(applicationID string, tag string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, applicationID).
		SetQueryParam("tag", tag).
		SetURL(ApiVersionByName)

	return core.NewTryRequest(s.Client, req)
}

// Create creates a new version of an application
func (s *Service) Create(ctx context.Context, applicationID string, opts CreateOptions) op.Result[jsonmodels.ApplicationVersion] {
	return core.Execute(ctx, s.createB(applicationID, opts), jsonmodels.NewApplicationVersion)
}

func (s *Service) createB(applicationID string, opts CreateOptions) *core.TryRequest {
	filename := opts.File.FilePath
	if filename == "" {
		filename = "application.zip"
	}

	applicationVersion, _ := json.Marshal(opts)

	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, applicationID).
		SetMultipartField("applicationBinary", filename, types.MimeTypeApplicationOctetStream, opts.File.Reader).
		SetMultipartField("applicationVersion", "", types.MimeTypeApplicationJSON, bytes.NewReader(applicationVersion)).
		SetURL(ApiVersions)

	return core.NewTryRequest(s.Client, req)
}

// CreateFromFile creates a new version from a file path or URL
func (s *Service) CreateFromFile(ctx context.Context, applicationID string, filenameOrURL string, version string, tags []string) op.Result[jsonmodels.ApplicationVersion] {
	var file *os.File
	var err error
	var shouldCleanup bool

	if isURL(filenameOrURL) {
		// Download from URL to temp file
		file, err = os.CreateTemp("", "application-*.zip")
		if err != nil {
			return op.Result[jsonmodels.ApplicationVersion]{
				Err: fmt.Errorf("could not create temp file: %w", err),
			}
		}
		shouldCleanup = true

		resp, downloadErr := http.Get(filenameOrURL)
		if downloadErr != nil {
			file.Close()
			os.Remove(file.Name())
			return op.Result[jsonmodels.ApplicationVersion]{
				Err: fmt.Errorf("failed to download from url: %w", downloadErr),
			}
		}
		defer resp.Body.Close()

		if _, writeErr := io.Copy(file, resp.Body); writeErr != nil {
			file.Close()
			os.Remove(file.Name())
			return op.Result[jsonmodels.ApplicationVersion]{
				Err: fmt.Errorf("failed to write downloaded content: %w", writeErr),
			}
		}

		// Seek back to start for reading
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			file.Close()
			os.Remove(file.Name())
			return op.Result[jsonmodels.ApplicationVersion]{
				Err: fmt.Errorf("failed to seek file: %w", seekErr),
			}
		}
	} else {
		// Open local file
		file, err = os.Open(filenameOrURL)
		if err != nil {
			return op.Result[jsonmodels.ApplicationVersion]{
				Err: fmt.Errorf("failed to open file: %w", err),
			}
		}
	}

	// Ensure cleanup
	if shouldCleanup {
		defer func() {
			file.Close()
			os.Remove(file.Name())
		}()
	} else {
		defer file.Close()
	}

	opts := CreateOptions{
		Version: version,
		Tags:    tags,
		File: UploadFileOptions{
			FilePath: filenameOrURL,
			Reader:   file,
		},
	}

	return s.Create(ctx, applicationID, opts)
}

// Update updates the tags for a specific application version
func (s *Service) Update(ctx context.Context, applicationID string, version string, tags []string) op.Result[jsonmodels.ApplicationVersion] {
	return core.Execute(ctx, s.updateB(applicationID, version, tags), jsonmodels.NewApplicationVersion)
}

func (s *Service) updateB(applicationID string, version string, tags []string) *core.TryRequest {
	body := map[string]interface{}{
		"tags": tags,
	}

	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, applicationID).
		SetPathParam(ParamVersion, version).
		SetBody(body).
		SetURL(ApiVersionByID)

	return core.NewTryRequest(s.Client, req)
}

// DeleteByVersion removes a specific version of an application by version string
func (s *Service) DeleteByVersion(ctx context.Context, applicationID string, version string) (*resty.Response, error) {
	return core.ExecuteResponseOnly(ctx, s.deleteByVersionB(applicationID, version))
}

func (s *Service) deleteByVersionB(applicationID string, version string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, applicationID).
		SetQueryParam("version", version).
		SetURL(ApiVersionByName)

	return core.NewTryRequest(s.Client, req)
}

// DeleteByTag removes a specific version of an application by tag
func (s *Service) DeleteByTag(ctx context.Context, applicationID string, tag string) (*resty.Response, error) {
	return core.ExecuteResponseOnly(ctx, s.deleteByTagB(applicationID, tag))
}

func (s *Service) deleteByTagB(applicationID string, tag string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, applicationID).
		SetQueryParam("tag", tag).
		SetURL(ApiVersionByName)

	return core.NewTryRequest(s.Client, req)
}

// isURL checks if a string is a URL
func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
