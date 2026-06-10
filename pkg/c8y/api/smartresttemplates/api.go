// Package smartresttemplates manages SmartREST 2.0 template collections.
//
// A collection is stored as an inventory managed object of type
// c8y_SmartRest2Template and registered under the external identity type
// c8y_SmartRest2DeviceIdentifier, with the collection name as the external ID
// (the X-Id devices reference when sending SmartREST 2.0 messages). The
// upsert uses that external identity to decide between create and update, and
// compares the request/response template lists order-insensitively so a
// collection exported from the platform can be re-imported without spurious
// updates.
package smartresttemplates

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/identity"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// ManagedObjectType is the fixed inventory type of SmartREST 2.0 template collections
const ManagedObjectType = "c8y_SmartRest2Template"

// ExternalIDType is the external identity type under which a collection is
// registered (the external ID is the collection name)
const ExternalIDType = "c8y_SmartRest2DeviceIdentifier"

// FragmentTemplates holds the request and response templates of a collection
const FragmentTemplates = "com_cumulocity_model_smartrest_csv_CsvSmartRestTemplate"

// FragmentExternalID is the bookkeeping fragment the platform stores on
// template collections, mirroring the external identity
const FragmentExternalID = "__externalId"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
		identity:       identity.NewService(s),
	}
}

// Service provides api to manage SmartREST 2.0 template collections
type Service struct {
	core.Service
	managedObjects *managedobjects.Service
	identity       *identity.Service
}

// CreateOptions describe the desired state of a template collection
type CreateOptions struct {
	// Name of the collection; also registered as the external identity
	// (the X-Id devices reference)
	Name string

	// RequestTemplates and ResponseTemplates define the collection. The
	// order of the templates does not matter: the lists are sorted by msgId
	// before writing and comparing, so reordering never triggers an update.
	RequestTemplates  []map[string]any
	ResponseTemplates []map[string]any

	// Fragments are custom top-level fields merged into the body before the
	// standard fields (name, type, __externalId and the template fragment),
	// which always win. They are part of the desired state: upserts compare
	// them against the existing object to decide whether an update is needed.
	Fragments []model.Fragment

	// Annotations are written together with the body on create and on every
	// real update, but are excluded from the upsert change detection so they
	// never trigger an update by themselves (e.g. sync timestamps, provenance
	// metadata).
	Annotations []model.Fragment
}

// desiredBody returns the managed-fields body used for change detection.
// Template lists are sorted so the comparison is order-insensitive.
func (opt CreateOptions) desiredBody() map[string]any {
	body := make(map[string]any)
	for _, fr := range opt.Fragments {
		if fr != nil {
			body[fr.FragmentKey()] = fr
		}
	}
	body["name"] = opt.Name
	body["type"] = ManagedObjectType
	body[FragmentExternalID] = opt.Name
	body[FragmentTemplates] = map[string]any{
		"requestTemplates":  sortedTemplates(opt.RequestTemplates),
		"responseTemplates": sortedTemplates(opt.ResponseTemplates),
	}
	return body
}

// annotate merges the annotation fragments into a body
func (opt CreateOptions) annotate(body map[string]any) map[string]any {
	for _, fr := range opt.Annotations {
		if fr != nil {
			body[fr.FragmentKey()] = fr
		}
	}
	return body
}

// sortedTemplates returns a copy of the templates sorted by msgId (then name)
// so that the template order in the input never matters. The result is never
// nil so it marshals to [] like the platform's export format.
func sortedTemplates(templates []map[string]any) []map[string]any {
	sorted := make([]map[string]any, len(templates))
	copy(sorted, templates)
	sort.SliceStable(sorted, func(i, j int) bool {
		return templateSortKey(sorted[i]) < templateSortKey(sorted[j])
	})
	return sorted
}

// templateSortKey builds the sort key of a template. msgId is unique within a
// collection; name breaks ties for malformed input.
func templateSortKey(template map[string]any) string {
	return fmt.Sprint(template["msgId"]) + "\x00" + fmt.Sprint(template["name"])
}

// normalizeTemplateOrder sorts the template lists of an existing managed
// object document with the same key as sortedTemplates, so the desired-state
// comparison is order-insensitive on both sides. Any parsing failure returns
// the document unchanged (the comparison then reports a difference, which
// results in an update rather than a missed one).
func normalizeTemplateOrder(existing []byte) []byte {
	var doc map[string]any
	if err := json.Unmarshal(existing, &doc); err != nil {
		return existing
	}
	fragment, ok := doc[FragmentTemplates].(map[string]any)
	if !ok {
		return existing
	}
	for _, key := range []string{"requestTemplates", "responseTemplates"} {
		items, ok := fragment[key].([]any)
		if !ok {
			// A null or missing list counts as empty, matching an empty
			// desired list
			fragment[key] = []any{}
			continue
		}
		sort.SliceStable(items, func(i, j int) bool {
			return anyTemplateSortKey(items[i]) < anyTemplateSortKey(items[j])
		})
		fragment[key] = items
	}
	normalized, err := json.Marshal(doc)
	if err != nil {
		return existing
	}
	return normalized
}

func anyTemplateSortKey(value any) string {
	if template, ok := value.(map[string]any); ok {
		return templateSortKey(template)
	}
	return fmt.Sprint(value)
}

// ListOptions to filter the template collections by
type ListOptions struct {
	pagination.PaginationOptions
}

// List the SmartREST 2.0 template collections in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.SmartRestTemplateCollection] {
	return core.ExecuteCollection(ctx, s.listB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSmartRestTemplateCollection)
}

// SmartRestTemplateCollectionIterator provides iteration over template collections
type SmartRestTemplateCollectionIterator = pagination.Iterator[jsonmodels.SmartRestTemplateCollection]

// ListAll returns an iterator for all template collections
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SmartRestTemplateCollectionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.SmartRestTemplateCollection] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewSmartRestTemplateCollection,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	listOpts := managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddOrderBy("name").
			AddFilterEqStr("type", ManagedObjectType).
			Build(),
		PaginationOptions: opt.PaginationOptions,
	}
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(listOpts)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// Get retrieves a template collection by name, resolved via its external identity
func (s *Service) Get(ctx context.Context, name string) op.Result[jsonmodels.SmartRestTemplateCollection] {
	return op.Result[jsonmodels.SmartRestTemplateCollection]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SmartRestTemplateCollection] {
		ref := s.identity.Get(execCtx, identity.IdentityOptions{Type: ExternalIDType, ExternalID: name})
		if ref.Err != nil {
			return op.Failed[jsonmodels.SmartRestTemplateCollection](ref.Err, ref.IsRetryable())
		}
		existing := s.managedObjects.Get(execCtx, ref.Data.ManagedObjectID(), managedobjects.GetOptions{})
		if existing.Err != nil {
			return op.Failed[jsonmodels.SmartRestTemplateCollection](existing.Err, existing.IsRetryable())
		}
		result := op.OK(jsonmodels.NewSmartRestTemplateCollection(existing.Data.Bytes()))
		result.HTTPStatus = existing.HTTPStatus
		return result
	}).WithMeta("operation", "get").
		ExecuteOrDefer(ctx)
}

// Create a template collection and register its external identity. When the
// identity cannot be registered the managed object is removed again (best
// effort) so a retry does not leave a duplicate behind.
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.SmartRestTemplateCollection] {
	return op.Result[jsonmodels.SmartRestTemplateCollection]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SmartRestTemplateCollection] {
		return s.create(execCtx, opt)
	}).WithMeta("operation", "create").
		ExecuteOrDefer(ctx)
}

func (s *Service) create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.SmartRestTemplateCollection] {
	created := s.managedObjects.Create(ctx, opt.annotate(opt.desiredBody()))
	if created.Err != nil {
		return op.Failed[jsonmodels.SmartRestTemplateCollection](created.Err, created.IsRetryable())
	}

	identityResult := s.identity.Create(ctx, created.Data.ID(), identity.IdentityOptions{
		Type:       ExternalIDType,
		ExternalID: opt.Name,
	})
	if identityResult.Err != nil {
		// Best-effort rollback so a retry does not create a duplicate
		s.managedObjects.Delete(ctx, created.Data.ID(), managedobjects.DeleteOptions{})
		return op.Failed[jsonmodels.SmartRestTemplateCollection](
			fmt.Errorf("failed to register external identity %s/%s: %w", ExternalIDType, opt.Name, identityResult.Err),
			identityResult.IsRetryable())
	}

	result := op.Created(jsonmodels.NewSmartRestTemplateCollection(created.Data.Bytes()))
	result.HTTPStatus = created.HTTPStatus
	return result
}

// Upsert creates the template collection when its external identity does not
// exist yet, and otherwise updates the existing managed object — but only
// when the desired state differs (annotations excluded, template order
// ignored), so re-applying the same collection performs no writes.
func (s *Service) Upsert(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.SmartRestTemplateCollection] {
	return op.Result[jsonmodels.SmartRestTemplateCollection]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.SmartRestTemplateCollection] {
		if opt.Name == "" {
			return op.Failed[jsonmodels.SmartRestTemplateCollection](fmt.Errorf("name is required"), false)
		}

		ref := s.identity.Get(execCtx, identity.IdentityOptions{Type: ExternalIDType, ExternalID: opt.Name})
		if ref.Err != nil {
			if !core.IsNotFound(ref.Err) {
				return op.Failed[jsonmodels.SmartRestTemplateCollection](
					fmt.Errorf("failed to lookup external identity %s/%s: %w", ExternalIDType, opt.Name, ref.Err),
					ref.IsRetryable())
			}
			created := s.create(execCtx, opt)
			created.Meta["found"] = false
			return created
		}

		existing := s.managedObjects.Get(execCtx, ref.Data.ManagedObjectID(), managedobjects.GetOptions{})
		if existing.Err != nil {
			return op.Failed[jsonmodels.SmartRestTemplateCollection](existing.Err, existing.IsRetryable())
		}

		desired := opt.desiredBody()
		if op.DesiredStateMatches(desired, normalizeTemplateOrder(existing.Data.Bytes())) {
			result := op.Skipped(jsonmodels.NewSmartRestTemplateCollection(existing.Data.Bytes()), "no changes detected")
			result.HTTPStatus = existing.HTTPStatus
			result.Meta["found"] = true
			return result
		}

		updated := s.managedObjects.Update(execCtx, existing.Data.ID(), opt.annotate(desired))
		if updated.Err != nil {
			return op.Failed[jsonmodels.SmartRestTemplateCollection](updated.Err, updated.IsRetryable())
		}
		result := op.Updated(jsonmodels.NewSmartRestTemplateCollection(updated.Data.Bytes()))
		result.HTTPStatus = updated.HTTPStatus
		result.Meta["found"] = true
		return result
	}).WithMeta("operation", "upsert").
		ExecuteOrDefer(ctx)
}

// Delete removes a template collection by name. A missing collection counts
// as the desired state being reached. The external identity is removed
// together with the managed object.
func (s *Service) Delete(ctx context.Context, name string) op.Result[core.NoContent] {
	return op.Result[core.NoContent]{}.WithExecutor(func(execCtx context.Context) op.Result[core.NoContent] {
		ref := s.identity.Get(execCtx, identity.IdentityOptions{Type: ExternalIDType, ExternalID: name})
		if ref.Err != nil {
			if core.IsNotFound(ref.Err) {
				return op.Skipped(core.NoContent{}, "not found")
			}
			return op.Failed[core.NoContent](ref.Err, ref.IsRetryable())
		}
		return s.managedObjects.Delete(execCtx, ref.Data.ManagedObjectID(), managedobjects.DeleteOptions{})
	}).WithMeta("operation", "delete").
		ExecuteOrDefer(ctx)
}
