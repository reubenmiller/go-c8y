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
