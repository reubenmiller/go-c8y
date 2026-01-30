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

func (b *InventoryQuery) AddFilterPart(v string) *InventoryQuery {
	if v != "" {
		b.Filter = append(b.Filter, v)
	}
	return b
}

func (b *InventoryQuery) AddFilterEqStr(k string, v any) *InventoryQuery {
	if v != "" {
		switch value := v.(type) {
		case string:
			b.Filter = append(b.Filter, fmt.Sprintf("(%s eq '%v')", k, value))
		default:
			b.Filter = append(b.Filter, fmt.Sprintf("(%s eq %v)", k, value))
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
