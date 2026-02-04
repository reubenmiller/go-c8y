package managedobjects

import (
	"context"
	"fmt"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/identity"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
)

func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.createB(body), jsonmodels.NewManagedObject)
}

func (s *Service) Get(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.ManagedObject] {
	// Resolve ID (supports "name:device", "externalId:type:id", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		// Use background context for resolution so it doesn't inherit the deferred flag
		// This allows lookups (like List) to actually execute
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}

	return core.ExecuteReturnResult(ctx, s.getB(resolvedID, opt), jsonmodels.NewManagedObject, meta)
}

func (s *Service) Update(ctx context.Context, ID string, body any) op.Result[jsonmodels.ManagedObject] {
	// Resolve ID (supports "name:device", "externalId:type:id", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}

	return core.ExecuteReturnResult(ctx, s.updateB(resolvedID, body), jsonmodels.NewManagedObject, meta)
}

func (s *Service) Delete(ctx context.Context, ID string, opt DeleteOptions) op.Result[jsonmodels.ManagedObject] {
	// Resolve ID (supports "name:device", "externalId:type:id", etc.)
	// If deferred execution is enabled, we still need to resolve the ID first
	// But do it in a normal context so the resolution actually completes
	resolutionCtx := ctx
	if ctxhelpers.IsDeferredExecution(ctx) {
		resolutionCtx = context.Background()
	}

	meta := make(map[string]any)
	resolvedID, err := s.ResolveID(resolutionCtx, ID, meta)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}

	return core.ExecuteReturnResult(ctx, s.deleteB(resolvedID, opt), jsonmodels.NewManagedObject, meta)
}

// GetOrCreateByName searches by name and optionally type, creating if not found
func (s *Service) GetOrCreateByName(ctx context.Context, name, objType string, body map[string]any) op.Result[jsonmodels.ManagedObject] {
	return op.Result[jsonmodels.ManagedObject]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.ManagedObject] {
		query := model.NewInventoryQuery().
			AddFilterEqStr("name", name).
			AddFilterEqStr("type", objType).
			Build()
		return s.getOrCreateWithQuery(execCtx, body, query)
	}).WithMeta("operation", "getOrCreateByName").
		ExecuteOrDefer(ctx)
}

// GetOrCreateByFragment searches for objects with a specific fragment property
func (s *Service) GetOrCreateByFragment(ctx context.Context, fragment string, body map[string]any) op.Result[jsonmodels.ManagedObject] {
	return op.Result[jsonmodels.ManagedObject]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.ManagedObject] {
		if fragment == "" {
			return op.Failed[jsonmodels.ManagedObject](fmt.Errorf("fragment must be set"), false)
		}
		query := model.NewInventoryQuery().
			HasFragment(fragment).
			Build()
		return s.getOrCreateWithQuery(execCtx, body, query)
	}).WithMeta("operation", "getOrCreateByFragment").
		ExecuteOrDefer(ctx)
}

// GetOrCreateWith provides a generic query-based lookup
// Example queries:
//   - "name eq 'device01' and type eq 'c8y_Device'"
//   - "has(c8y_IsDevice) and c8y_Serial eq '12345'"
//   - "fragmentType eq 'c8y_CustomFragment'"
func (s *Service) GetOrCreateWith(ctx context.Context, body map[string]any, query string) op.Result[jsonmodels.ManagedObject] {
	return op.Result[jsonmodels.ManagedObject]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.ManagedObject] {
		query_ := model.NewInventoryQuery().
			AddFilterPart(query).
			Build()
		return s.getOrCreateWithQuery(execCtx, body, query_)
	}).WithMeta("operation", "getOrCreateWith").
		ExecuteOrDefer(ctx)
}

// getOrCreateWithQuery is the internal implementation
func (s *Service) getOrCreateWithQuery(ctx context.Context, body map[string]any, query string) op.Result[jsonmodels.ManagedObject] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.ManagedObject], bool) {
		searchOpts := ListOptions{}
		searchOpts.PaginationOptions.PageSize = 1
		searchOpts.Query = query

		listResult := s.List(ctx, searchOpts)
		if listResult.Err != nil {
			return listResult, false
		}

		// Check if any items were found
		for item := range listResult.Data.Iter() {
			found := jsonmodels.NewManagedObject(item.Bytes())
			result := op.OK(found)
			result.HTTPStatus = listResult.HTTPStatus
			result.RequestID = listResult.RequestID
			result.Meta["query"] = query
			return result, true
		}

		return op.Result[jsonmodels.ManagedObject]{}, false
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.ManagedObject] {
		createResult := s.Create(ctx, body)
		// Preserve the original result (including Failed status if creation failed)
		createResult.Meta["query"] = query
		return createResult
	}

	// Execute get-or-create pattern (automatically sets Meta["found"])
	return op.GetOrCreateR(ctx, finder, creator)
}

// GetOrCreateByExternalIDOptions options for GetOrCreateByExternalID
type GetOrCreateByExternalIDOptions struct {
	ExternalID     string // External ID to lookup
	ExternalIDType string // External ID type (defaults to "c8y_Serial")
	Body           any    // Managed object body to create if not found
}

// GetOrCreateByExternalID looks up a managed object by external identity,
// creating both the managed object and identity if not found.
//
// This is useful for device provisioning workflows where devices are identified
// by serial numbers or other external identifiers.
//
// Flow:
//  1. Try to get managed object by external identity
//  2. If found, return it with Status=OK and Meta["found"]=true
//  3. If not found:
//     - Create the managed object
//     - Assign the external identity
//     - Return with Status=Created and Meta["found"]=false
//
// Example:
//
//	result := client.ManagedObjects.GetOrCreateByExternalID(ctx,
//	    GetOrCreateByExternalIDOptions{
//	        ExternalID: "device-serial-12345",
//	        ExternalIDType: "c8y_Serial",
//	        Body: map[string]any{
//	            "name": "My Device",
//	            "type": "c8y_Device",
//	            "c8y_IsDevice": map[string]any{},
//	        },
//	    },
//	)

// externalIDState tracks the workflow state for GetOrCreateByExternalID
type externalIDState struct {
	// Input parameters
	externalID     string
	externalIDType string
	body           any
	service        *Service

	// Workflow state
	managedObject jsonmodels.ManagedObject
	found         bool
	created       bool
	duplicate     bool

	// Error tracking
	err        error
	httpStatus int
}

func (s *Service) GetOrCreateByExternalID(
	ctx context.Context,
	opts GetOrCreateByExternalIDOptions,
) op.Result[jsonmodels.ManagedObject] {
	return op.Result[jsonmodels.ManagedObject]{}.WithExecutor(func(execCtx context.Context) op.Result[jsonmodels.ManagedObject] {
		// Default external ID type
		if opts.ExternalIDType == "" {
			opts.ExternalIDType = "c8y_Serial"
		}

		return s.executeGetOrCreateByExternalID(execCtx, opts)
	}).WithMeta("operation", "getOrCreateByExternalID").
		ExecuteOrDefer(ctx)
}

func (s *Service) executeGetOrCreateByExternalID(
	ctx context.Context,
	opts GetOrCreateByExternalIDOptions,
) op.Result[jsonmodels.ManagedObject] {

	// Initialize state
	state := &externalIDState{
		externalID:     opts.ExternalID,
		externalIDType: opts.ExternalIDType,
		body:           opts.Body,
		service:        s,
	}

	// Step 1: Lookup by external identity
	lookup := func(ctx context.Context, st *externalIDState) (*externalIDState, error) {
		if st.err != nil {
			return st, st.err
		}

		identResult := st.service.identityService.Get(ctx, identity.IdentityOptions{
			ExternalID: st.externalID,
			Type:       st.externalIDType,
		})

		if identResult.Err != nil {
			// Identity not found - need to create
			return st, nil
		}

		// Identity exists, get the managed object
		moID := identResult.Data.ManagedObjectID()
		getResult := st.service.Get(ctx, moID, GetOptions{})
		if getResult.Err != nil {
			st.err = getResult.Err
			st.httpStatus = getResult.HTTPStatus
			return st, getResult.Err
		}

		st.managedObject = getResult.Data
		st.found = true
		st.httpStatus = getResult.HTTPStatus
		return st, nil
	}

	// Step 2: Create managed object if not found
	create := func(ctx context.Context, st *externalIDState) (*externalIDState, error) {
		if st.err != nil || st.found {
			return st, st.err
		}

		createResult := st.service.Create(ctx, st.body)
		if createResult.Err != nil {
			st.err = createResult.Err
			st.httpStatus = createResult.HTTPStatus
			return st, createResult.Err
		}

		st.managedObject = createResult.Data
		st.created = true
		st.httpStatus = createResult.HTTPStatus
		return st, nil
	}

	// Step 3: Assign external identity
	assignIdentity := func(ctx context.Context, st *externalIDState) (*externalIDState, error) {
		if st.err != nil || st.found {
			return st, st.err
		}

		moID := st.managedObject.ID()
		identResult := st.service.identityService.Create(ctx, moID, identity.IdentityOptions{
			ExternalID: st.externalID,
			Type:       st.externalIDType,
		})

		if identResult.Err == nil {
			// Success
			return st, nil
		}

		// Identity assignment failed
		if identResult.HTTPStatus == 409 {
			// Conflict: identity already exists
			// Delete the newly created MO
			_ = st.service.Delete(ctx, moID, DeleteOptions{})

			// Fetch the existing managed object
			lookupResult := st.service.identityService.Get(ctx, identity.IdentityOptions{
				ExternalID: st.externalID,
				Type:       st.externalIDType,
			})
			if lookupResult.Err != nil {
				st.err = fmt.Errorf("identity conflict but failed to retrieve existing managed object: %w", identResult.Err)
				return st, st.err
			}

			existingMOID := lookupResult.Data.ManagedObjectID()
			getResult := st.service.Get(ctx, existingMOID, GetOptions{})
			if getResult.Err != nil {
				st.err = fmt.Errorf("identity conflict but failed to retrieve existing managed object: %w", getResult.Err)
				return st, st.err
			}

			st.managedObject = getResult.Data
			st.created = false
			st.duplicate = true
			st.httpStatus = 409
			return st, nil
		}

		// Other failure - cleanup the created managed object
		deleteResult := st.service.Delete(ctx, moID, DeleteOptions{})
		if deleteResult.Err != nil {
			st.err = fmt.Errorf("failed to assign identity and cleanup failed: identity error: %w, delete error: %v", identResult.Err, deleteResult.Err)
		} else {
			st.err = fmt.Errorf("failed to assign identity (managed object deleted): %w", identResult.Err)
		}
		st.httpStatus = identResult.HTTPStatus
		return st, st.err
	}

	// Execute pipeline
	pipeline := op.Pipe(lookup, create, assignIdentity)
	finalState, _ := pipeline(ctx, state)

	// Convert state to Result
	if finalState.err != nil {
		return op.Failed[jsonmodels.ManagedObject](finalState.err, true).
			WithHTTPStatus(finalState.httpStatus).
			WithMeta("externalID", finalState.externalID).
			WithMeta("externalIDType", finalState.externalIDType)
	}

	if finalState.duplicate {
		return op.Duplicate(finalState.managedObject, map[string]any{
			"externalID":      finalState.externalID,
			"externalIDType":  finalState.externalIDType,
			"orphanedDeleted": true,
		}).WithHTTPStatus(finalState.httpStatus)
	}

	if finalState.created {
		return op.Created(finalState.managedObject, map[string]any{
			"externalID":       finalState.externalID,
			"externalIDType":   finalState.externalIDType,
			"identityAssigned": true,
			"found":            false,
		}).WithHTTPStatus(finalState.httpStatus)
	}

	return op.OK(finalState.managedObject, map[string]any{
		"externalID":     finalState.externalID,
		"externalIDType": finalState.externalIDType,
		"found":          true,
		"lookupMethod":   "externalIdentity",
	}).WithHTTPStatus(finalState.httpStatus)
}

func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

type ManagedObjectIterator struct {
	items iter.Seq[jsonmodels.ManagedObject]
	err   error
}

func (it *ManagedObjectIterator) Items() iter.Seq[jsonmodels.ManagedObject] {
	return it.items
}

func (it *ManagedObjectIterator) Err() error {
	return it.err
}

func paginateManagedObjects(ctx context.Context, fetch func(page int) op.Result[jsonmodels.ManagedObject], maxItems int64) *ManagedObjectIterator {
	iterator := &ManagedObjectIterator{}

	iterator.items = func(yield func(jsonmodels.ManagedObject) bool) {
		page := 1
		count := int64(0)
		for {
			result := fetch(page)
			if result.Err != nil {
				iterator.err = result.Err
				return
			}
			countBeforeResults := count
			for doc := range result.Data.Iter() {
				if maxItems > 0 && count >= maxItems {
					return
				}
				item := jsonmodels.NewManagedObject(doc.Bytes())
				if !yield(item) {
					return
				}
				count++
			}
			if countBeforeResults == count {
				// No more results
				slog.Info("Stopping pagination as results array is empty")
				return
			}

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}

func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ManagedObjectIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateManagedObjects(ctx, func(page int) op.Result[jsonmodels.ManagedObject] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, opts.GetMaxItems())
}
