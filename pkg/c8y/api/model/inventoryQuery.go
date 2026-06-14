package model

import (
	"fmt"
	"strings"
)

type InventoryQuery struct {
	Filter  []string
	OrderBy []string
	Invert  bool
}

func NewInventoryQuery() *InventoryQuery {
	return &InventoryQuery{}
}

func (b *InventoryQuery) AddOrderBy(key string) *InventoryQuery {
	if key != "" {
		b.OrderBy = append(b.OrderBy, key)
	}
	return b
}

type InventoryQueryCondition interface {
	Enabled() bool
	String() string
}

func FilterHasFragment(name string, enable bool) FilterCondition {
	return FilterCondition{
		Format:  "has(%s)",
		Value:   name,
		Disable: !enable,
	}
}

type FilterCondition struct {
	Format  string
	Value   any
	Disable bool
}

func (c FilterCondition) String() string {
	if c.Format == "" {
		return fmt.Sprintf("%s", c.Value)
	}
	return fmt.Sprintf(c.Format, c.Value)
}

func (c FilterCondition) Enabled() bool {
	return !c.IsEmpty() && !c.Disable
}

func (c FilterCondition) IsEmpty() bool {
	return fmt.Sprintf("%v", c.Value) == ""
}

func (b *InventoryQuery) AddFilterPart(parts ...string) *InventoryQuery {
	for _, v := range parts {
		if v != "" {
			b.Filter = append(b.Filter, v)
		}
	}
	return b
}

func (b *InventoryQuery) HasFragment(v string) *InventoryQuery {
	if v != "" {
		b.AddFilter(FilterHasFragment(v, true))
	}
	return b
}

func (b *InventoryQuery) AddFilterEqStr(k string, v any) *InventoryQuery {
	switch value := v.(type) {
	case string:
		if value != "" {
			b.Filter = append(b.Filter, fmt.Sprintf("(%s eq '%v')", k, value))
		}
	default:
		strValue := fmt.Sprintf("%v", value)
		if strValue != "" {
			b.Filter = append(b.Filter, fmt.Sprintf("(%s eq %v)", k, strValue))
		}
	}
	return b
}

func (b *InventoryQuery) AddFilter(conditions ...InventoryQueryCondition) *InventoryQuery {
	for _, cond := range conditions {
		if cond.Enabled() {
			b.Filter = append(b.Filter, cond.String())
		}
	}
	return b
}

func (b *InventoryQuery) ByGroupID(k string) *InventoryQuery {
	if k != "" {
		b.Filter = append(b.Filter, fmt.Sprintf("bygroupid(%s)", k))
	}
	return b
}

// WithIDCursor rewrites an inventory query so it pages by an ascending _id
// keyset: it ANDs "_id gt '<afterID>'" into the existing filter and forces
// "$orderby=_id asc". An empty afterID is treated as "0" (the start). This is
// the v2 port of the go-c8y-cli v1 managed-object pagination optimisation.
//
//	WithIDCursor("", "0")                       => $filter=(_id gt '0') $orderby=_id asc
//	WithIDCursor("$filter=(type eq 'x')", "42") => $filter=(_id gt '42' and (type eq 'x')) $orderby=_id asc
func WithIDCursor(query, afterID string) string {
	if afterID == "" {
		afterID = "0"
	}
	cursor := fmt.Sprintf("_id gt '%s'", afterID)

	body := extractFilterBody(query)
	var filter string
	switch {
	case body == "":
		filter = fmt.Sprintf("(%s)", cursor)
	case strings.HasPrefix(body, "(") && strings.HasSuffix(body, ")"):
		filter = fmt.Sprintf("(%s and %s)", cursor, body)
	default:
		filter = fmt.Sprintf("(%s and (%s))", cursor, body)
	}
	return fmt.Sprintf("$filter=%s $orderby=_id asc", filter)
}

// QueryHasConflictingOrder reports whether the query carries an explicit
// $orderby that is not _id-based. Such an order is incompatible with the _id
// keyset (which forces "$orderby=_id asc"): Auto falls back to offset to
// preserve it, and an explicit id-keyset request is rejected.
func QueryHasConflictingOrder(query string) bool {
	i := strings.Index(query, "$orderby=")
	if i < 0 {
		return false
	}
	clause := strings.ToLower(strings.TrimSpace(query[i+len("$orderby="):]))
	return clause != "" && clause != "_id" && clause != "_id asc"
}

// extractFilterBody returns the filter expression from an inventory query,
// stripping the "$filter=" prefix and any "$orderby=" suffix. A bare expression
// (no "$filter=") is returned as-is; an orderby-only query yields "".
func extractFilterBody(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	if i := strings.Index(q, "$filter="); i >= 0 {
		rest := q[i+len("$filter="):]
		if j := strings.Index(rest, "$orderby="); j >= 0 {
			rest = rest[:j]
		}
		return strings.TrimSpace(rest)
	}
	if strings.HasPrefix(q, "$orderby=") {
		return ""
	}
	return q
}

func (b *InventoryQuery) Build() string {
	q := &strings.Builder{}
	if len(b.Filter) > 0 {
		q.Write([]byte("$filter="))
		if b.Invert {
			q.Write([]byte("not"))
		}
		q.Write([]byte("("))
		q.WriteString(strings.Join(b.Filter, " and "))
		q.Write([]byte(")"))
	}
	if len(b.OrderBy) > 0 {
		if q.Len() > 0 {
			q.Write([]byte(" "))
		}
		q.Write([]byte("$orderby="))
		q.WriteString(strings.Join(b.OrderBy, ","))
	}
	return q.String()
}
