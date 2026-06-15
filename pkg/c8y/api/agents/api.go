// Package agents provides access to Cumulocity agents. An agent is a managed
// object carrying the com_cumulocity_model_Agent fragment (in addition to
// c8y_IsDevice), so this service wraps the managed-objects API and scopes reads
// and name resolution to that fragment — a plain managed-object lookup would
// also match ordinary devices.
package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

const (
	// FragmentIsAgent marks a managed object as an agent.
	FragmentIsAgent = "com_cumulocity_model_Agent"
	// FragmentIsDevice marks a managed object as a device (agents are devices).
	FragmentIsDevice = "c8y_IsDevice"
)

// ManagedObjectIterator iterates agents.
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// GetOptions and DeleteOptions reuse the managed-object options.
type (
	GetOptions    = managedobjects.GetOptions
	DeleteOptions = managedobjects.DeleteOptions
)

// Service interacts with agents.
type Service struct {
	core.Service

	managedObjects *managedobjects.Service
}

// NewService creates an agents service.
func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
	}
}

// AgentRef is a typed reference to an agent, resolved by ResolveID. Construct
// it with ByID or ByName, or cast a dynamic string with AgentRef(value).
type AgentRef string

// ByID creates a direct-ID reference (no lookup).
func ByID(id string) AgentRef { return AgentRef(id) }

// ByName creates a reference resolved by agent name (wildcards allowed).
func ByName(name string) AgentRef { return AgentRef("name:" + name) }

// ListOptions filters agents. The Query field is the fully-built q expression
// (e.g. from model.InventoryQuery); restrict it to agents with ScopeToAgents
// when the caller has not already done so.
type ListOptions struct {
	Type         string `url:"type,omitempty"`
	FragmentType string `url:"fragmentType,omitempty"`
	Query        string `url:"q"`

	managedobjects.GetOptions
	pagination.PaginationOptions
}

// List returns a page of managed objects matching opt's query. It is thin (the
// query is passed through as-is, like devices.List); the resolver and
// ScopeToAgents are responsible for restricting results to agents.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.listB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator over every agent matching opts. By default it uses
// the _id keyset optimisation; set ListOptions.Strategy to override.
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

// Get returns an agent by reference (id, name:, or query:).
func (s *Service) Get(ctx context.Context, ref AgentRef, opt GetOptions) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Get(ctx, id, opt)
}

// Update updates an agent by reference.
func (s *Service) Update(ctx context.Context, ref AgentRef, body any) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Update(ctx, id, body)
}

// Delete deletes an agent by reference.
func (s *Service) Delete(ctx context.Context, ref AgentRef, opt DeleteOptions) op.Result[core.NoContent] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}
	return s.managedObjects.Delete(ctx, id, opt)
}

// Create creates an agent managed object. The c8y_IsDevice and
// com_cumulocity_model_Agent fragments are added when the body (a map) does not
// already set them; other body shapes are passed through unchanged (the caller
// is responsible for including the fragments).
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.ManagedObject] {
	if m, ok := body.(map[string]any); ok {
		if _, ok := m[FragmentIsDevice]; !ok {
			m[FragmentIsDevice] = map[string]any{}
		}
		if _, ok := m[FragmentIsAgent]; !ok {
			m[FragmentIsAgent] = map[string]any{}
		}
		body = m
	}
	return s.managedObjects.Create(ctx, body)
}

// CreateAgent creates a new agent with the given name, including the
// c8y_IsDevice and com_cumulocity_model_Agent fragments.
func (s *Service) CreateAgent(ctx context.Context, name string) op.Result[jsonmodels.ManagedObject] {
	return s.managedObjects.Create(ctx, jsonmodels.NewAgent(name))
}

// ResolveID resolves an agent reference to an id. A plain string is used as-is;
// "name:<name>" and "query:<q>" are looked up scoped to agents.
func (s *Service) ResolveID(ctx context.Context, ref string, meta map[string]any) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty agent reference")
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
			HasFragment(FragmentIsAgent).
			AddFilterEqStr("name", value).
			AddOrderBy("name").
			Build(), value, meta)
	case "query":
		return s.lookupByQuery(ctx, ScopeToAgents(value), value, meta)
	default:
		return "", fmt.Errorf("unknown agent resolver scheme: %q", scheme)
	}
}

// lookupByQuery returns the id of the first agent matching query. label is the
// original reference, for error messages and metadata.
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
	return "", fmt.Errorf("agent not found: %s", label)
}

// ScopeToAgents builds a query that restricts results to agents, combining the
// com_cumulocity_model_Agent fragment check with the given raw filter expression
// (not an already-built $filter string). An empty filter yields the bare
// fragment check.
func ScopeToAgents(filter string) string {
	fragment := fmt.Sprintf("has(%s)", FragmentIsAgent)
	return model.NewInventoryQuery().AddFilterPart(fragment, filter).Build()
}
