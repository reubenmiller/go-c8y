package dtm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// Definition is a definition to create or update. All kinds use
// Identifier, Tags and JSONSchema; Contexts is for property definitions and
// Composition for asset definitions. Extra holds custom fragments.
//
// The server adds fields to the stored jsonSchema ($schema,
// additionalProperties, and for asset definitions properties and required
// built from Composition). Describe the structure of an asset definition in
// Composition, not in JSONSchema.properties.
type Definition struct {
	Identifier  string         `json:"identifier"`
	Contexts    []string       `json:"contexts,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	JSONSchema  map[string]any `json:"jsonSchema,omitempty"`
	Composition *Composition   `json:"composition,omitempty"`
	Extra       map[string]any `json:"-"`
}

// MarshalJSON merges Extra into the object.
func (d Definition) MarshalJSON() ([]byte, error) {
	type plain Definition
	b, err := json.Marshal(plain(d))
	if err != nil || len(d.Extra) == 0 {
		return b, err
	}
	m := jsonObject(b)
	for k, v := range d.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// Composition lists the properties and sub-assets an asset definition
// allows.
type Composition struct {
	AllowedProperties    []AllowedItem `json:"allowedProperties,omitempty"`
	AllowedSubAssets     []AllowedItem `json:"allowedSubAssets,omitempty"`
	AdditionalProperties *bool         `json:"additionalProperties,omitempty"`
	AdditionalSubAssets  *bool         `json:"additionalSubAssets,omitempty"`
}

// AllowedItem references a property or asset definition by identifier.
// MinOccurs and MaxOccurs are strings in the API, for example "0", "1" or
// "*". Empty values use the server default.
type AllowedItem struct {
	Identifier string `json:"identifier"`
	MinOccurs  string `json:"minOccurs,omitempty"`
	MaxOccurs  string `json:"maxOccurs,omitempty"`
}

// definitionKey reads the identifier and contexts from a body.
func definitionKey(body map[string]any) (string, []string, error) {
	id, _ := body["identifier"].(string)
	if id == "" {
		return "", nil, fmt.Errorf("definition identifier is required")
	}
	var contexts []string
	if raw, ok := body["contexts"].([]any); ok {
		for _, c := range raw {
			if s, ok := c.(string); ok {
				contexts = append(contexts, s)
			}
		}
	}
	return id, contexts, nil
}

// Create creates a definition (POST /definitions/{kind}). body is a
// Definition, a map, or any struct that marshals to the definition JSON.
//
// A 409 for a title that another definition uses wraps ErrDuplicateTitle.
func (s *DefinitionService) Create(ctx context.Context, body any) op.Result[jsonmodels.DigitalTwinDefinition] {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(ParamKind, string(s.kind)).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiDefinitions)
	return classifyDefinitionError(httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), jsonmodels.NewDigitalTwinDefinition))
}

// UpdateDefinitionOptions selects the definition to update.
type UpdateDefinitionOptions struct {
	// Contexts selects the property definition that applies to exactly
	// these domain entities. Property definitions only; empty selects the
	// definition that applies to no entity.
	Contexts []string
}

type contextsQuery struct {
	Contexts []string `url:"contexts,comma,omitempty"`
}

// Update replaces the definition identifier with body. The PUT replaces the
// whole jsonSchema, including the fields the server generated when the
// definition was created (for example the required "source" of a
// measurement definition). Use Upsert to merge the desired schema with the
// stored one.
//
// Property definitions are addressed by identifier and opt.Contexts; the
// other kinds take the identifier from the body, so Update sets it.
func (s *DefinitionService) Update(ctx context.Context, identifier string, body any, opt UpdateDefinitionOptions) op.Result[jsonmodels.DigitalTwinDefinition] {
	if identifier == "" {
		return op.Failed[jsonmodels.DigitalTwinDefinition](fmt.Errorf("empty definition identifier"), false)
	}
	m, err := toMap(body)
	if err != nil {
		return op.Failed[jsonmodels.DigitalTwinDefinition](err, false)
	}
	m["identifier"] = identifier

	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamKind, string(s.kind)).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(m)
	if s.kind == KindProperties {
		req.SetPathParam(ParamIdentifier, identifier).
			SetQueryParamsFromValues(httpcore.QueryParameters(contextsQuery{opt.Contexts})).
			SetURL(ApiDefinition)
	} else {
		req.SetURL(ApiDefinitions)
	}
	return classifyDefinitionError(httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), jsonmodels.NewDigitalTwinDefinition))
}

// DeleteDefinitionOptions selects the definition to delete.
type DeleteDefinitionOptions struct {
	// Contexts selects the property definition that applies to exactly
	// these domain entities. Property definitions only.
	Contexts []string
}

// Delete deletes a definition. A definition that does not exist returns
// StatusSkipped without an error, so Delete is safe in defer. A definition
// still referenced by another (HTTP 409) returns an error wrapping
// ErrDefinitionInUse.
func (s *DefinitionService) Delete(ctx context.Context, identifier string, opt DeleteDefinitionOptions) op.Result[core.NoContent] {
	if identifier == "" {
		return op.Failed[core.NoContent](fmt.Errorf("empty definition identifier"), false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamKind, string(s.kind)).
		SetPathParam(ParamIdentifier, identifier).
		SetURL(ApiDefinition)
	if s.kind == KindProperties {
		req.SetQueryParamsFromValues(httpcore.QueryParameters(contextsQuery{opt.Contexts}))
	}
	res := httpcore.ExecuteNoContent(ctx, httpcore.NewTryRequest(s.svc.Client, req), map[string]any{"identifier": identifier})
	return skipIfNotFound(classifyDefinitionError(res))
}

// Find returns the definition with the identifier and contexts of body, as
// used by GetOrCreate and Upsert. For property definitions the contexts
// must match exactly (in any order). The bool is false when there is none.
func (s *DefinitionService) Find(ctx context.Context, identifier string, contexts []string) (op.Result[jsonmodels.DigitalTwinDefinition], bool) {
	if s.kind != KindProperties {
		res := s.Get(ctx, identifier, GetDefinitionOptions{})
		if core.IsNotFound(res.Err) {
			return res, false
		}
		return res, res.Err == nil
	}
	list := s.List(ctx, ListDefinitionsOptions{
		Identifiers:       []string{identifier},
		PaginationOptions: pagination.PaginationOptions{PageSize: 100},
	})
	if list.Err != nil {
		return op.Failed[jsonmodels.DigitalTwinDefinition](list.Err, false).WithHTTPStatus(list.HTTPStatus), false
	}
	for def := range list.Seq() {
		if def.Identifier() == identifier && sameSet(def.Contexts(), contexts) {
			return op.OK(def), true
		}
	}
	return op.NoMatch[jsonmodels.DigitalTwinDefinition](), false
}

// GetOrCreate returns the definition with the identifier (and, for property
// definitions, the contexts) of body, creating it when it does not exist.
// It never updates an existing definition.
//
// Status: StatusCreated (new) or StatusOK (existed, no write).
func (s *DefinitionService) GetOrCreate(ctx context.Context, body any) op.Result[jsonmodels.DigitalTwinDefinition] {
	return op.Result[jsonmodels.DigitalTwinDefinition]{}.WithExecutor(func(ctx context.Context) op.Result[jsonmodels.DigitalTwinDefinition] {
		m, err := toMap(body)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinDefinition](err, false)
		}
		id, contexts, err := definitionKey(m)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinDefinition](err, false)
		}
		if existing, ok := s.Find(ctx, id, contexts); ok {
			return existing.WithMeta("existing", true)
		} else if existing.Err != nil {
			return existing
		}
		return s.Create(ctx, m)
	}).WithMeta("operation", "getOrCreate").ExecuteOrDefer(ctx)
}

// Upsert creates the definition described by body or updates the existing
// one (matched by identifier, and contexts for property definitions) to
// match it.
//
// When every field of body already has the same value on the stored
// definition, Upsert returns StatusSkipped without a write. The comparison
// ignores fields body does not set, treats tags and contexts as sets, and
// matches composition entries by identifier, so the defaults the server
// fills in (for example minOccurs "0") do not count as changes.
//
// On update, the desired jsonSchema is merged over the stored one for all
// kinds except asset definitions (whose schema the server rebuilds from the
// composition), so fields the server generated are kept.
//
// A definition that differs but cannot be edited (shared from the
// enterprise tenant, or a library definition) returns an error wrapping
// ErrReadOnlyDefinition, without a write.
//
// Status: StatusCreated, StatusUpdated or StatusSkipped.
func (s *DefinitionService) Upsert(ctx context.Context, body any) op.Result[jsonmodels.DigitalTwinDefinition] {
	return op.Result[jsonmodels.DigitalTwinDefinition]{}.WithExecutor(func(ctx context.Context) op.Result[jsonmodels.DigitalTwinDefinition] {
		desired, err := toMap(body)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinDefinition](err, false)
		}
		id, contexts, err := definitionKey(desired)
		if err != nil {
			return op.Failed[jsonmodels.DigitalTwinDefinition](err, false)
		}

		existing, found := s.Find(ctx, id, contexts)
		if !found {
			if existing.Err != nil {
				return existing
			}
			return s.Create(ctx, desired)
		}

		changed := definitionChanges(desired, existing.Data.Bytes())
		if len(changed) == 0 {
			return skippedFrom(existing, "no changes detected")
		}
		if existing.Data.IsShared() || !existing.Data.CanEdit() {
			return op.Failed[jsonmodels.DigitalTwinDefinition](
				fmt.Errorf("%w: %s %s differs in %s", ErrReadOnlyDefinition, s.kind, id, strings.Join(changed, ", ")), false)
		}

		update := desired
		if s.kind != KindAssets {
			if stored := jsonObject([]byte(existing.Data.Get("jsonSchema").Raw)); stored != nil {
				if want, ok := desired["jsonSchema"].(map[string]any); ok {
					update = deepMerge(desired, map[string]any{"jsonSchema": deepMerge(stored, want)})
				}
			}
		}
		res := s.Update(ctx, id, update, UpdateDefinitionOptions{Contexts: contexts})
		return res.WithMeta("changed", changed)
	}).WithMeta("operation", "upsert").ExecuteOrDefer(ctx)
}

// definitionChanges returns the top-level fields of desired that differ
// from the existing definition JSON (empty when it matches).
func definitionChanges(desired map[string]any, existing []byte) []string {
	stored := jsonObject(existing)
	if stored == nil {
		return []string{"(unreadable)"}
	}
	var changed []string
	for key, want := range desired {
		have := stored[key]
		switch key {
		case "identifier":
			continue
		case "tags", "contexts":
			if !sameSet(stringSlice(want), stringSlice(have)) {
				changed = append(changed, key)
			}
		case "composition":
			if !compositionMatches(want, have) {
				changed = append(changed, key)
			}
		default:
			if !op.DesiredStateMatches(map[string]any{key: want}, mustJSON(map[string]any{key: have})) {
				changed = append(changed, key)
			}
		}
	}
	slices.Sort(changed)
	return changed
}

// compositionMatches compares a desired composition with a stored one.
// allowedProperties and allowedSubAssets must list the same identifiers;
// each desired entry is a subset match of the stored entry with the same
// identifier. Other fields use a subset match.
func compositionMatches(want, have any) bool {
	w, ok := normalize(want).(map[string]any)
	if !ok {
		return false
	}
	h, _ := normalize(have).(map[string]any)
	if h == nil {
		h = map[string]any{}
	}
	for key, wv := range w {
		hv := h[key]
		switch key {
		case "allowedProperties", "allowedSubAssets":
			if !allowedItemsMatch(wv, hv) {
				return false
			}
		default:
			if !op.DesiredStateMatches(map[string]any{key: wv}, mustJSON(map[string]any{key: hv})) {
				return false
			}
		}
	}
	return true
}

func allowedItemsMatch(want, have any) bool {
	byID := func(v any) map[string]map[string]any {
		out := map[string]map[string]any{}
		items, _ := v.([]any)
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				if id, _ := m["identifier"].(string); id != "" {
					out[id] = m
				}
			}
		}
		return out
	}
	w, h := byID(want), byID(have)
	if len(w) != len(h) {
		return false
	}
	for id, wi := range w {
		hi, ok := h[id]
		if !ok || !op.DesiredStateMatches(wi, mustJSON(hi)) {
			return false
		}
	}
	return true
}

// classifyDefinitionError wraps 409 errors with the matching sentinel.
func classifyDefinitionError[T any](r op.Result[T]) op.Result[T] {
	if !core.ErrHasStatus(r.Err, http.StatusConflict) {
		return r
	}
	msg := r.Err.Error()
	switch {
	case strings.Contains(msg, "still referenced"):
		r.Err = fmt.Errorf("%w: %w", ErrDefinitionInUse, r.Err)
	case strings.Contains(msg, "with title") && !strings.Contains(msg, "with identifier"):
		r.Err = fmt.Errorf("%w: %w", ErrDuplicateTitle, r.Err)
	}
	return r
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// normalize round-trips v through JSON, so Go structs and parsed JSON
// compare the same way.
func normalize(v any) any {
	var out any
	_ = json.Unmarshal(mustJSON(v), &out)
	return out
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

// ComposeItem selects one property definition for Compose.
type ComposeItem struct {
	Identifier string `json:"identifier"`
	// Context selects the property definition that applies to this domain
	// entity (for example ContextAsset). Required when the definition
	// applies to specific entities. The API takes one context here, unlike
	// the contexts list on the definition itself.
	Context string `json:"context,omitempty"`
}

// Compose builds one JSON object schema whose properties are the given
// property definitions (POST /definitions/properties/compose). Nothing is
// stored. Property definitions only.
func (s *DefinitionService) Compose(ctx context.Context, items []ComposeItem) op.Result[jsondoc.JSONDoc] {
	if s.kind != KindProperties {
		return op.Failed[jsondoc.JSONDoc](fmt.Errorf("compose is only available for property definitions"), false)
	}
	if len(items) == 0 {
		return op.Failed[jsondoc.JSONDoc](fmt.Errorf("at least one property definition is required"), false)
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetContentType(types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(items).
		SetURL(ApiDefinitionsCompose)
	return httpcore.Execute(ctx, httpcore.NewTryRequest(s.svc.Client, req), jsondoc.New)
}
