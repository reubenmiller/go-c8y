// Package configuration provides access to the Cumulocity configuration
// repository. A configuration file is a managed object of type
// c8y_ConfigurationDump carrying the configuration's name, configurationType,
// url and an optional deviceType filter, so this service wraps the
// managed-objects API and scopes reads and name resolution to that type — a
// plain managed-object lookup would also match unrelated objects.
//
// Create and Update optionally upload a binary file: the file is stored as an
// inventory binary, its url is written onto the configuration managed object and
// the binary is linked back as a child addition (mirroring the v1 CLI's
// binary-upload + add-child-addition behaviour).
package configuration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects/childadditions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/sjson"
	"resty.dev/v3"
)

const (
	// TypeConfiguration is the type of a configuration-file managed object.
	TypeConfiguration = "c8y_ConfigurationDump"
	// FragmentGlobal marks the configuration as globally available.
	FragmentGlobal = "c8y_Global"
)

// ManagedObjectIterator iterates configuration files.
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// GetOptions reuses the managed-object detail options.
type GetOptions = managedobjects.GetOptions

// DeleteOptions reuses the managed-object delete options (forceCascade).
type DeleteOptions = managedobjects.DeleteOptions

// UploadFileOptions describes an optional binary file uploaded with a
// create/update.
type UploadFileOptions = core.UploadFileOptions

// Service interacts with the configuration repository.
type Service struct {
	core.Service

	managedObjects *managedobjects.Service
	binaries       *binaries.Service
	childAdditions *childadditions.Service
}

// NewService creates a configuration service.
func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
		binaries:       binaries.NewService(s),
		childAdditions: childadditions.NewService(s),
	}
}

// ConfigurationRef is a typed reference to a configuration file, resolved by
// ResolveID. Construct it with ByID or ByName, or cast a dynamic string with
// ConfigurationRef(value).
type ConfigurationRef string

// ByID creates a direct-ID reference (no lookup).
func ByID(id string) ConfigurationRef { return ConfigurationRef(id) }

// ByName creates a reference resolved by configuration name (wildcards allowed).
func ByName(name string) ConfigurationRef { return ConfigurationRef("name:" + name) }

// ListOptions filters configuration files. The Query field is the fully-built q
// expression (e.g. from model.InventoryQuery); restrict it to configuration
// files with ScopeToConfiguration when the caller has not already done so.
type ListOptions struct {
	Type         string `url:"type,omitempty"`
	FragmentType string `url:"fragmentType,omitempty"`
	Query        string `url:"q"`

	managedobjects.GetOptions
	pagination.PaginationOptions
}

// List returns a page of managed objects matching opt's query. It is thin (the
// query is passed through as-is, like devices.List); the resolver and
// ScopeToConfiguration are responsible for restricting results to configuration
// files.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.listB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator over every configuration file matching opts. By
// default it uses the _id keyset optimisation; set ListOptions.Strategy to
// override.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ManagedObjectIterator {
	strategy, err := managedobjects.ResolveListStrategy(opts.Strategy, opts.Query)
	if err != nil {
		return pagination.NewErrorIterator[jsonmodels.ManagedObject](err)
	}
	return pagination.PaginateWith(
		ctx,
		pagination.PageRequest{PaginationOptions: opts.PaginationOptions},
		strategy,
		func(req pagination.PageRequest) op.Result[jsonmodels.ManagedObject] {
			o := opts
			o.PaginationOptions = req.PaginationOptions
			if req.AfterID != "" {
				o.Query = model.WithIDCursor(opts.Query, req.AfterID)
			}
			return s.List(ctx, o)
		},
		jsonmodels.NewManagedObject,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// Get returns a configuration file by reference (id, name:, or query:).
func (s *Service) Get(ctx context.Context, ref ConfigurationRef, opt GetOptions) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Get(ctx, id, opt)
}

// CreateRaw creates a configuration managed object from a pre-built body. The
// c8y_ConfigurationDump type and the c8y_Global fragment are added when the body
// (a map) does not already set them; other body shapes (e.g. the CLI's
// json.RawMessage) are passed through unchanged (the caller's body template is
// responsible for including them).
func (s *Service) CreateRaw(ctx context.Context, body any) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.Create(ctx, withDefaults(body))
}

// UpdateRaw updates a configuration file by reference from a pre-built body.
func (s *Service) UpdateRaw(ctx context.Context, ref ConfigurationRef, body any) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Update(ctx, id, body)
}

// CreateOptions creates a configuration file with an optional binary upload.
type CreateOptions struct {
	Body any
	File UploadFileOptions
}

// Create creates a configuration managed object. When a file is supplied it is
// uploaded as an inventory binary first, its url is written onto the body, and
// the binary is linked back as a child addition of the new configuration —
// mirroring the v1 CLI's binary-upload + add-child-addition behaviour. With no
// file it is exactly CreateRaw. The upload + create + link run inside a deferred
// executor so confirmation and dry-run behave correctly.
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.ManagedObject] {
	if opt.File.IsZero() {
		return s.CreateRaw(ctx, opt.Body)
	}
	return op.Result[jsonmodels.ManagedObject]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.ManagedObject] {
		body, binaryID, err := s.uploadAndSetURL(execCtx, opt.Body, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.ManagedObject](err, true)
		}
		res := s.managedObjects.Create(execCtx, withDefaults(body))
		if res.Err != nil {
			return res
		}
		s.linkBinary(execCtx, res.Data.ID(), binaryID)
		return res
	}).WithMeta("operation", "create").ExecuteOrDefer(ctx)
}

// UpdateOptions updates a configuration file with an optional binary upload.
type UpdateOptions struct {
	Body any
	File UploadFileOptions
}

// Update updates a configuration file by reference. When a file is supplied it is
// uploaded as an inventory binary first, its url is written onto the body, and
// the binary is linked back as a child addition. With no file it is exactly
// UpdateRaw.
func (s *Service) Update(ctx context.Context, ref ConfigurationRef, opt UpdateOptions) op.Result[jsonmodels.ManagedObject] {
	if opt.File.IsZero() {
		return s.UpdateRaw(ctx, ref, opt.Body)
	}
	return op.Result[jsonmodels.ManagedObject]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.ManagedObject] {
		id, err := s.ResolveID(execCtx, string(ref), nil)
		if err != nil {
			return op.Failed[jsonmodels.ManagedObject](err, false)
		}
		body, binaryID, err := s.uploadAndSetURL(execCtx, opt.Body, opt.File)
		if err != nil {
			return op.Failed[jsonmodels.ManagedObject](err, true)
		}
		res := s.managedObjects.Update(execCtx, id, body)
		if res.Err != nil {
			return res
		}
		s.linkBinary(execCtx, id, binaryID)
		return res
	}).WithMeta("operation", "update").ExecuteOrDefer(ctx)
}

// Delete deletes a configuration file by reference. forceCascade (DeleteOptions)
// also removes any related binaries.
func (s *Service) Delete(ctx context.Context, ref ConfigurationRef, opt DeleteOptions) op.Result[core.NoContent] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}
	return s.managedObjects.Delete(ctx, id, opt)
}

// ResolveID resolves a configuration reference to an id. A plain string is used
// as-is; "name:<name>" and "query:<q>" are looked up scoped to configuration
// files.
func (s *Service) ResolveID(ctx context.Context, ref string, meta map[string]any) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty configuration reference")
	}
	scheme, value, hasScheme := strings.Cut(ref, ":")
	if !hasScheme {
		return ref, nil
	}
	// Resolve for real even under dry run / deferred execution.
	ctx = contexthelpers.ResolutionContext(ctx)
	switch scheme {
	case "id":
		return value, nil
	case "name":
		return s.lookupByQuery(ctx, model.NewInventoryQuery().
			AddFilterEqStr("type", TypeConfiguration).
			AddFilterEqStr("name", value).
			AddOrderBy("name").
			Build(), value, meta)
	case "query":
		return s.lookupByQuery(ctx, ScopeToConfiguration(value), value, meta)
	default:
		return "", fmt.Errorf("unknown configuration resolver scheme: %q", scheme)
	}
}

// lookupByQuery returns the id of the first configuration file matching query.
// label is the original reference, for error messages and metadata.
func (s *Service) lookupByQuery(ctx context.Context, query, label string, meta map[string]any) (string, error) {
	opt := ListOptions{Query: query}
	opt.PageSize = 1
	result := s.List(ctx, opt)
	if result.Err != nil {
		return "", result.Err
	}
	for item := range op.Iter(result) {
		if meta != nil {
			meta["id"] = item.ID()
			meta["name"] = item.Name()
			meta["source"] = "name"
		}
		return item.ID(), nil
	}
	return "", fmt.Errorf("configuration not found: %s", label)
}

// ScopeToConfiguration builds a query that restricts results to configuration
// files, combining the c8y_ConfigurationDump type check with the given raw
// filter expression (not an already-built $filter string). An empty filter
// yields the bare type check.
func ScopeToConfiguration(filter string) string {
	typeFilter := fmt.Sprintf("type eq '%s'", TypeConfiguration)
	return model.NewInventoryQuery().AddFilterPart(typeFilter, filter).Build()
}

// withDefaults adds the configuration type and the c8y_Global fragment to a map
// body when absent; non-map bodies are returned unchanged (the caller's body
// template sets them).
func withDefaults(body any) any {
	m, ok := body.(map[string]any)
	if !ok {
		return body
	}
	if _, ok := m["type"]; !ok {
		m["type"] = TypeConfiguration
	}
	if _, ok := m[FragmentGlobal]; !ok {
		m[FragmentGlobal] = map[string]any{}
	}
	return m
}

// uploadAndSetURL uploads the binary file and writes its self url onto the body,
// returning the updated body and the uploaded binary's id.
func (s *Service) uploadAndSetURL(ctx context.Context, body any, file UploadFileOptions) (any, string, error) {
	res := s.binaries.Create(ctx, file)
	if res.IsError() {
		return body, "", fmt.Errorf("failed to upload binary: %w", res.Err)
	}
	out, err := setURL(body, res.Data.Self())
	if err != nil {
		return body, "", err
	}
	return out, res.Data.ID(), nil
}

// setURL writes url onto the body, handling the CLI's json.RawMessage / a map /
// a nil body.
func setURL(body any, url string) (any, error) {
	switch b := body.(type) {
	case nil:
		return map[string]any{"url": url}, nil
	case map[string]any:
		b["url"] = url
		return b, nil
	case json.RawMessage:
		return sjson.SetBytes(b, "url", url)
	case []byte:
		return sjson.SetBytes(b, "url", url)
	default:
		return body, nil
	}
}

// linkBinary links an uploaded binary back to the configuration as a child
// addition. Best-effort: a missing id (e.g. under dry run) or a link failure does
// not fail the create/update, matching the v1 post-action.
func (s *Service) linkBinary(ctx context.Context, moID, binaryID string) {
	if moID == "" || binaryID == "" {
		return
	}
	_ = s.childAdditions.Assign(ctx, moID, binaryID)
}
