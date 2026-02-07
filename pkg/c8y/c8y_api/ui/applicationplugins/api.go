package applicationplugins

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

// Note: Application plugins are managed by modifying the host application object itself
// via PUT /application/applications/{id}, not through dedicated plugin endpoints.
// This service wraps those operations for convenience.
var (
	ApiApplication = "/application/applications/{id}"
)

var ParamId = "id"

type Service struct {
	core.Service
}

func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// PluginReference represents a reference to an installed plugin
type PluginReference struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// ApplicationWithPlugins wraps application data focusing on plugin management
type ApplicationWithPlugins struct {
	jsondoc.Facade
}

func NewApplicationWithPlugins(data []byte) ApplicationWithPlugins {
	return ApplicationWithPlugins{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (a ApplicationWithPlugins) ID() string {
	return a.Get("id").String()
}

func (a ApplicationWithPlugins) Plugins() []map[string]interface{} {
	plugins := a.Get("applicationBuilder.plugins").Array()
	result := make([]map[string]interface{}, len(plugins))
	for i, p := range plugins {
		// Convert gjson.Result to map[string]interface{}
		result[i] = p.Value().(map[string]interface{})
	}
	return result
}

// PluginReferenceResult wraps plugin reference data
type PluginReferenceResult struct {
	jsondoc.Facade
}

func NewPluginReference(data []byte) PluginReferenceResult {
	return PluginReferenceResult{jsondoc.Facade{JSONDoc: jsondoc.New(data)}}
}

func (p PluginReferenceResult) ID() string {
	return p.Get("id").String()
}

func (p PluginReferenceResult) Name() string {
	return p.Get("name").String()
}

func (p PluginReferenceResult) Version() string {
	return p.Get("version").String()
}

type ListOptions struct {
	pagination.PaginationOptions
}

type PluginReferenceIterator = pagination.Iterator[PluginReferenceResult]

// getApplication fetches the current application object
func (s *Service) getApplication(ctx context.Context, applicationID string) op.Result[ApplicationWithPlugins] {
	return core.Execute(ctx, s.getApplicationB(applicationID), NewApplicationWithPlugins)
}

func (s *Service) getApplicationB(applicationID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, applicationID).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

// updateApplication updates the application with new data
func (s *Service) updateApplication(ctx context.Context, applicationID string, body map[string]interface{}) op.Result[ApplicationWithPlugins] {
	return core.Execute(ctx, s.updateApplicationB(applicationID, body), NewApplicationWithPlugins)
}

func (s *Service) updateApplicationB(applicationID string, body map[string]interface{}) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, applicationID).
		SetBody(body).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

// List retrieves all plugins installed in an application
// This fetches the application and returns it - use Plugins() method to extract plugin references
func (s *Service) List(ctx context.Context, applicationID string, opt ListOptions) op.Result[ApplicationWithPlugins] {
	return s.getApplication(ctx, applicationID)
}

// ListAll is not applicable for application plugins as they're part of the application object
// Use List() to get the application and extract plugins with app.Plugins()
func (s *Service) ListAll(ctx context.Context, applicationID string, opts ListOptions) op.Result[ApplicationWithPlugins] {
	return s.List(ctx, applicationID, opts)
}

// Install adds a plugin reference to an application
// This fetches the application, adds the plugin, and updates it
func (s *Service) Install(ctx context.Context, applicationID string, pluginID string) op.Result[ApplicationWithPlugins] {
	// Get current application state
	appResult := s.getApplication(ctx, applicationID)
	if appResult.Err != nil {
		return appResult
	}

	// Get current plugins
	currentPluginsResult := appResult.Data.Get("applicationBuilder.plugins").Array()

	// Convert to []interface{}
	currentPlugins := make([]interface{}, len(currentPluginsResult))
	for i, p := range currentPluginsResult {
		currentPlugins[i] = p.Value()
	}

	// Add new plugin reference
	newPlugin := map[string]interface{}{
		"id": pluginID,
	}
	updatedPlugins := append(currentPlugins, newPlugin)

	// Update application with new plugin list
	body := map[string]interface{}{
		"applicationBuilder": map[string]interface{}{
			"plugins": updatedPlugins,
		},
	}

	return s.updateApplication(ctx, applicationID, body)
}

// Update updates installed plugin references in an application
// This replaces all plugins with the provided list
func (s *Service) Update(ctx context.Context, applicationID string, pluginRefs []PluginReference) op.Result[ApplicationWithPlugins] {
	// Convert plugin references to the format expected by the API
	plugins := make([]map[string]interface{}, len(pluginRefs))
	for i, ref := range pluginRefs {
		plugin := map[string]interface{}{}
		if ref.ID != "" {
			plugin["id"] = ref.ID
		}
		if ref.Name != "" {
			plugin["name"] = ref.Name
		}
		if ref.Version != "" {
			plugin["version"] = ref.Version
		}
		plugins[i] = plugin
	}

	body := map[string]interface{}{
		"applicationBuilder": map[string]interface{}{
			"plugins": plugins,
		},
	}

	return s.updateApplication(ctx, applicationID, body)
}

// Replace replaces all installed plugins in an application
// This is the same as Update but more explicit in intent
func (s *Service) Replace(ctx context.Context, applicationID string, pluginRefs []PluginReference) op.Result[ApplicationWithPlugins] {
	return s.Update(ctx, applicationID, pluginRefs)
}

// Delete removes a plugin reference from an application
// This fetches the application, removes the plugin, and updates it
func (s *Service) Delete(ctx context.Context, applicationID string, pluginID string) (*resty.Response, error) {
	// Get current application state
	appResult := s.getApplication(ctx, applicationID)
	if appResult.Err != nil {
		return nil, appResult.Err
	}

	// Get current plugins and filter out the one to delete
	currentPluginsResult := appResult.Data.Get("applicationBuilder.plugins").Array()
	updatedPlugins := []interface{}{}

	for _, p := range currentPluginsResult {
		// Get the id from the gjson.Result
		id := p.Get("id").String()
		if id != "" && id != pluginID {
			updatedPlugins = append(updatedPlugins, p.Value())
		}
	}

	// Update application with filtered plugin list
	body := map[string]interface{}{
		"applicationBuilder": map[string]interface{}{
			"plugins": updatedPlugins,
		},
	}

	updateResult := s.updateApplication(ctx, applicationID, body)
	if updateResult.Err != nil {
		return nil, updateResult.Err
	}

	// Return a success response (updateResult doesn't have Response field, so create one)
	return &resty.Response{}, nil
}
