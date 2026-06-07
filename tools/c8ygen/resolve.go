package main

import "strings"

// refName returns the final path segment of a JSON pointer ref, e.g.
// "#/components/parameters/queryParam_alarm_source" -> "queryParam_alarm_source".
func refName(ref string) string {
	if ref == "" {
		return ""
	}
	parts := strings.Split(ref, "/")
	return parts[len(parts)-1]
}

// resolveParam follows a $ref into components/parameters one level. Parameters in this
// spec are not chained, so single-level resolution is sufficient.
func (o *OAS) resolveParam(p Parameter) Parameter {
	if p.Ref == "" {
		return p
	}
	if target, ok := o.Components.Parameters[refName(p.Ref)]; ok {
		return target
	}
	return p
}

// resolveSchema follows a $ref into components/schemas one level, returning the target
// schema. Used to resolve property-level refs (e.g. desc_self) to their concrete type.
func (o *OAS) resolveSchema(s Schema) Schema {
	if s.Ref == "" {
		return s
	}
	if target, ok := o.Components.Schemas[refName(s.Ref)]; ok {
		return target
	}
	return s
}

// goScalarType maps a resolved schema to a Go type and the url-tag option suffix.
// It returns ("", "", false) for types the pilot generator does not emit (objects,
// nested arrays, unmapped). overrides allows a resource to pin a Go type for a field.
func (o *OAS) goScalarType(s Schema) (goType string, urlOpts string, ok bool) {
	s = o.resolveSchema(s)
	switch s.Type {
	case "string":
		if s.Format == "date-time" {
			return "time.Time", ",omitempty,omitzero", true
		}
		return "string", ",omitempty", true
	case "integer":
		return "int", ",omitempty", true
	case "number":
		return "float64", ",omitempty", true
	case "boolean":
		return "bool", ",omitempty", true
	case "array":
		if s.Items == nil {
			return "", "", false
		}
		item := o.resolveSchema(*s.Items)
		if item.Type == "string" {
			return "[]string", ",omitempty", true
		}
		return "", "", false
	default:
		return "", "", false
	}
}
