package dtm

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// DefinitionKind is the path segment of a definition type.
type DefinitionKind string

const (
	KindProperties   DefinitionKind = "properties"
	KindAssets       DefinitionKind = "assets"
	KindMeasurements DefinitionKind = "measurements"
	KindEvents       DefinitionKind = "events"
	KindAlarms       DefinitionKind = "alarms"
)

// Domain entities a property definition can apply to (contexts and
// applicableTo).
const (
	ContextMeasurement  = "measurement"
	ContextEvent        = "event"
	ContextAlarm        = "alarm"
	ContextAudit        = "audit"
	ContextOperation    = "operation"
	ContextAsset        = "asset"
	ContextTenantOption = "tenant_option"
)

// DefinitionIterator iterates definitions across pages.
type DefinitionIterator = pagination.Iterator[jsonmodels.DigitalTwinDefinition]

// DefinitionsService groups the five definition types. They share one
// method set; options that only apply to one kind say so.
type DefinitionsService struct {
	Properties   *DefinitionService
	Assets       *DefinitionService
	Measurements *DefinitionService
	Events       *DefinitionService
	Alarms       *DefinitionService
}

func newDefinitionsService(s *httpcore.Service) *DefinitionsService {
	return &DefinitionsService{
		Properties:   &DefinitionService{svc: *s, kind: KindProperties},
		Assets:       &DefinitionService{svc: *s, kind: KindAssets},
		Measurements: &DefinitionService{svc: *s, kind: KindMeasurements},
		Events:       &DefinitionService{svc: *s, kind: KindEvents},
		Alarms:       &DefinitionService{svc: *s, kind: KindAlarms},
	}
}

// DefinitionService reads the definitions of one kind.
type DefinitionService struct {
	svc  httpcore.Service
	kind DefinitionKind
}

// Kind returns the definition kind this service reads.
func (s *DefinitionService) Kind() DefinitionKind { return s.kind }

// GetDefinitionOptions selects a definition.
type GetDefinitionOptions struct {
	// ApplicableTo selects the property definition that applies to this
	// domain entity (for example ContextAsset). Without it the API returns
	// the property definition that applies to no entity. Property
	// definitions only.
	ApplicableTo string `url:"applicableTo,omitempty"`
}

// Get returns the definition with the given identifier.
func (s *DefinitionService) Get(ctx context.Context, identifier string, opt GetDefinitionOptions) op.Result[jsonmodels.DigitalTwinDefinition] {
	if identifier == "" {
		return op.Failed[jsonmodels.DigitalTwinDefinition](fmt.Errorf("empty definition identifier"), false)
	}
	if s.kind != KindProperties {
		opt.ApplicableTo = ""
	}
	return httpcore.Execute(ctx, s.getB(identifier, opt), jsonmodels.NewDigitalTwinDefinition)
}

func (s *DefinitionService) getB(identifier string, opt GetDefinitionOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamKind, string(s.kind)).
		SetPathParam(ParamIdentifier, identifier).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiDefinition)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListDefinitionsOptions filters definitions. Filters are combined with AND;
// values within one filter are combined with OR.
type ListDefinitionsOptions struct {
	// Query is a DTM query, for example "$filter=(jsonSchema.title eq 'X')".
	// A bare filter expression is wrapped with "$filter=(...)".
	Query       string   `url:"query,omitempty"`
	Identifiers []string `url:"identifiers,comma,omitempty"`
	Titles      []string `url:"titles,comma,omitempty"`
	Tags        []string `url:"tags,comma,omitempty"`

	// Origin filters property definitions by c8y_Origin. Property
	// definitions only.
	Origin string `url:"origin,omitempty"`
	// ApplicableTo filters property definitions by domain entity. Property
	// definitions only.
	ApplicableTo string `url:"applicableTo,omitempty"`
	// OnlyRoots returns asset definitions that are not used inside another
	// asset definition. Asset definitions only.
	OnlyRoots bool `url:"onlyRoots,omitempty"`

	pagination.PaginationOptions
}

// scoped drops the options the endpoint of this kind does not accept, so a
// shared options value can be reused across kinds.
func (s *DefinitionService) scoped(opt ListDefinitionsOptions) ListDefinitionsOptions {
	if s.kind != KindProperties {
		opt.Origin = ""
		opt.ApplicableTo = ""
	}
	if s.kind != KindAssets {
		opt.OnlyRoots = false
	}
	opt.Query = normalizeQuery(opt.Query)
	return opt
}

// List returns one page of definitions.
func (s *DefinitionService) List(ctx context.Context, opt ListDefinitionsOptions) op.Result[jsonmodels.DigitalTwinDefinition] {
	return httpcore.ExecuteCollection(ctx, s.listB(s.scoped(opt)), ResultPropertyDefinitions, types.ResponseFieldStatistics, jsonmodels.NewDigitalTwinDefinition)
}

func (s *DefinitionService) listB(opt ListDefinitionsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamKind, string(s.kind)).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiDefinitions)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultPropertyDefinitions)
}

// ListAll returns an iterator over all definitions matching opt.
func (s *DefinitionService) ListAll(ctx context.Context, opt ListDefinitionsOptions) *DefinitionIterator {
	return pagination.Paginate(ctx, opt.PaginationOptions, func(p pagination.PaginationOptions) op.Result[jsonmodels.DigitalTwinDefinition] {
		o := opt
		o.PaginationOptions = p
		return s.List(ctx, o)
	}, jsonmodels.NewDigitalTwinDefinition)
}

// CountDefinitionsOptions filters a definition count.
type CountDefinitionsOptions struct {
	Identifiers []string `url:"identifiers,comma,omitempty"`
	Titles      []string `url:"titles,comma,omitempty"`
	Tags        []string `url:"tags,comma,omitempty"`
	// ApplicableTo is for property definitions only.
	ApplicableTo string `url:"applicableTo,omitempty"`
	// OnlyRoots is for asset definitions only.
	OnlyRoots bool `url:"onlyRoots,omitempty"`
}

// Count returns the number of definitions matching opt.
func (s *DefinitionService) Count(ctx context.Context, opt CountDefinitionsOptions) op.Result[int64] {
	if s.kind != KindProperties {
		opt.ApplicableTo = ""
	}
	if s.kind != KindAssets {
		opt.OnlyRoots = false
	}
	return httpcore.Execute(ctx, s.countB(opt), parseCount)
}

func (s *DefinitionService) countB(opt CountDefinitionsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamKind, string(s.kind)).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiDefinitionsCount)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// TagsOptions limits which definitions the tags are collected from. Empty
// options return the tags of all definitions of the kind.
type TagsOptions struct {
	Identifiers []string `url:"identifiers,comma,omitempty"`
	Titles      []string `url:"titles,comma,omitempty"`
}

// Tags returns the distinct tags used by definitions of this kind. Asset
// definitions have no tags endpoint, so Tags fails for them without sending
// a request.
func (s *DefinitionService) Tags(ctx context.Context, opt TagsOptions) op.Result[[]string] {
	if s.kind == KindAssets {
		return op.Failed[[]string](fmt.Errorf("the DTM API has no tags endpoint for asset definitions"), false)
	}
	return httpcore.Execute(ctx, s.tagsB(opt), parseStringArray)
}

func (s *DefinitionService) tagsB(opt TagsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamKind, string(s.kind)).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiDefinitionsTags)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func parseStringArray(b []byte) []string {
	items := gjson.ParseBytes(b).Array()
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.String())
	}
	return out
}
