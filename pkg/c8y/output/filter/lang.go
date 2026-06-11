package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/araddon/dateparse"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// Parse compiles go-c8y-cli style --filter expressions into a single
// Predicate (multiple expressions are ANDed). Expressions have the form
//
//	<property> <operator> <value>   e.g. "name like linux*"
//	<operator> <value>              e.g. "has c8y_Mobile"
//
// following the same tokenization rules as go-c8y-cli: values may be quoted
// with single or double quotes to prevent numeric/boolean coercion, and an
// expression with no recognized operator defaults to "contains".
//
// Supported operators: eq (=, ==), neq (!=, <>), gt (>), gte (>=), lt (<),
// lte (<=), contains, strictcontains, startswith, endswith, like, notlike,
// match, notmatch, has (keyIn), hasnot/nothas/missing (keyNotIn), datelt,
// datelte, olderthan, dategt, dategte, newerthan, leneq, lenneq, lengt,
// lengte, lenlt, lenlte, includes, notincludes.
//
// All pattern and date compilation happens here; the returned predicate
// performs only path lookups and comparisons per document.
func Parse(expressions ...string) (Predicate, error) {
	preds := make([]Predicate, 0, len(expressions))
	for _, expr := range expressions {
		p, err := parseExpression(expr)
		if err != nil {
			return nil, err
		}
		preds = append(preds, p)
	}
	if len(preds) == 1 {
		return preds[0], nil
	}
	return And(preds...), nil
}

func parseExpression(expr string) (Predicate, error) {
	property := ""
	operator := ""
	value := ""

	fields := splitQuoted(expr, ' ', 3)
	switch len(fields) {
	case 0:
		return nil, fmt.Errorf("filter: empty expression")
	case 1:
		// A bare value matches the whole document (go-c8y-cli's default
		// "contains" operator).
		operator = "contains"
		value = fields[0]
	case 2:
		operator = fields[0]
		value = fields[1]
	default:
		property = strings.TrimSpace(fields[0])
		operator = fields[1]
		value = fields[2]
	}

	quoted := isQuoted(value)
	if quoted {
		value = strings.Trim(value, "\"'")
	}

	return compileCondition(property, strings.ToLower(operator), value, quoted)
}

func compileCondition(property, operator, value string, quoted bool) (Predicate, error) {
	switch operator {
	case "has", "keyin":
		return Exists(keyPath(property, value)), nil
	case "hasnot", "nothas", "missing", "keynotin":
		return Not(Exists(keyPath(property, value))), nil
	}

	get := pathGetter(property)

	// Numeric / boolean coercion (skipped for quoted values), mirroring
	// go-c8y-cli's AddRawFilters.
	var numValue float64
	isNum := false
	var boolValue, isBool bool
	if !quoted {
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			numValue, isNum = v, true
		} else if v, err := strconv.ParseBool(value); err == nil {
			boolValue, isBool = v, true
		}
	}

	switch operator {
	case "eq", "=", "==":
		switch {
		case isNum:
			return func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && v.Float() == numValue }, nil
		case isBool:
			return func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && v.Bool() == boolValue }, nil
		default:
			return func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && v.String() == value }, nil
		}
	case "neq", "!=", "<>":
		eq, err := compileCondition(property, "eq", value, quoted)
		if err != nil {
			return nil, err
		}
		return Not(eq), nil
	case "gt", ">", "gte", ">=", "lt", "<", "lte", "<=":
		if !isNum {
			return nil, fmt.Errorf("filter: operator %q requires a numeric value, got %q", operator, value)
		}
		return numericCompare(get, operator, numValue), nil
	case "contains":
		needle := strings.ToLower(value)
		return func(d jsondoc.JSONDoc) bool {
			return strings.Contains(strings.ToLower(get(d).String()), needle)
		}, nil
	case "strictcontains":
		return func(d jsondoc.JSONDoc) bool {
			return strings.Contains(get(d).String(), value)
		}, nil
	case "startswith":
		return func(d jsondoc.JSONDoc) bool { return strings.HasPrefix(get(d).String(), value) }, nil
	case "endswith":
		return func(d jsondoc.JSONDoc) bool { return strings.HasSuffix(get(d).String(), value) }, nil
	case "like", "-like", "notlike", "-notlike":
		re, err := wildcardToRegexp(value)
		if err != nil {
			return nil, fmt.Errorf("filter: invalid wildcard pattern %q: %w", value, err)
		}
		p := func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && re.MatchString(v.String()) }
		if strings.Contains(operator, "notlike") {
			return Not(p), nil
		}
		return p, nil
	case "match", "-match", "notmatch", "-notmatch":
		re, err := regexp.Compile("(?ims)" + value)
		if err != nil {
			return nil, fmt.Errorf("filter: invalid regex %q: %w", value, err)
		}
		p := func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && re.MatchString(v.String()) }
		if strings.Contains(operator, "notmatch") {
			return Not(p), nil
		}
		return p, nil
	case "datelt", "datelte", "olderthan", "dategt", "dategte", "newerthan":
		ref, err := dateparse.ParseAny(value)
		if err != nil {
			return nil, fmt.Errorf("filter: invalid date %q: %w", value, err)
		}
		return func(d jsondoc.JSONDoc) bool {
			v := get(d)
			if !v.Exists() {
				return false
			}
			ts, err := dateparse.ParseAny(v.String())
			if err != nil {
				return false
			}
			switch operator {
			case "datelt":
				return ts.Before(ref)
			case "datelte", "olderthan":
				return ts.Before(ref) || ts.Equal(ref)
			case "dategt":
				return ts.After(ref)
			default: // dategte, newerthan
				return ts.After(ref) || ts.Equal(ref)
			}
		}, nil
	case "leneq", "lenneq", "lengt", "lengte", "lenlt", "lenlte":
		if !isNum {
			return nil, fmt.Errorf("filter: operator %q requires a numeric value, got %q", operator, value)
		}
		want := int(numValue)
		return func(d jsondoc.JSONDoc) bool {
			v := get(d)
			if !v.Exists() {
				return false
			}
			n := valueLength(v)
			switch operator {
			case "leneq":
				return n == want
			case "lenneq":
				return n != want
			case "lengt":
				return n > want
			case "lengte":
				return n >= want
			case "lenlt":
				return n < want
			default:
				return n <= want
			}
		}, nil
	case "includes", "notincludes":
		p := func(d jsondoc.JSONDoc) bool {
			found := false
			get(d).ForEach(func(_, item gjson.Result) bool {
				if isNum && item.Type == gjson.Number {
					found = item.Float() == numValue
				} else {
					found = item.String() == value
				}
				return !found
			})
			return found
		}
		if operator == "notincludes" {
			return Not(p), nil
		}
		return p, nil
	}
	return nil, fmt.Errorf("filter: unsupported operator %q", operator)
}

func numericCompare(get func(jsondoc.JSONDoc) gjson.Result, operator string, want float64) Predicate {
	return func(d jsondoc.JSONDoc) bool {
		v := get(d)
		if !v.Exists() {
			return false
		}
		n := v.Float()
		switch operator {
		case "gt", ">":
			return n > want
		case "gte", ">=":
			return n >= want
		case "lt", "<":
			return n < want
		default: // lte, <=
			return n <= want
		}
	}
}

// pathGetter returns an accessor for a property path. An empty or "." path
// refers to the whole document.
func pathGetter(property string) func(jsondoc.JSONDoc) gjson.Result {
	if property == "" || property == "." {
		return func(d jsondoc.JSONDoc) gjson.Result { return d.Get() }
	}
	return func(d jsondoc.JSONDoc) gjson.Result { return d.Get(property) }
}

// keyPath builds the lookup path for key-existence operators, which take the
// key as the expression value: "has c8y_Mobile" or "has c8y_Hardware.model".
func keyPath(property, key string) string {
	if property == "" || property == "." {
		return key
	}
	return property + "." + key
}

func valueLength(v gjson.Result) int {
	if v.IsArray() || v.IsObject() {
		n := 0
		v.ForEach(func(_, _ gjson.Result) bool { n++; return true })
		return n
	}
	return len(v.String())
}

// wildcardToRegexp converts a go-c8y-cli wildcard pattern to a regular
// expression with identical semantics to pkg/matcher.ConvertWildcardToRegex:
// '*' matches any sequence, matching is case-insensitive and anchored.
func wildcardToRegexp(pattern string) (*regexp.Regexp, error) {
	pattern = strings.ReplaceAll(pattern, "\\", "\\\\")
	pattern = strings.ReplaceAll(pattern, ".", "\\.")
	pattern = strings.ReplaceAll(pattern, "*", ".*")
	return regexp.Compile("(?ims)^" + pattern + "$")
}

// splitQuoted splits s on sep outside of single/double quoted sections,
// joining any fields beyond maxSplit back together (same behavior as
// go-c8y-cli's splitFilter).
func splitQuoted(s string, sep rune, maxSplit int) []string {
	quoted := false
	openedQuote := ' '
	fields := strings.FieldsFunc(s, func(r rune) bool {
		if r == '"' || r == '\'' {
			if !quoted {
				openedQuote = r
				quoted = true
			} else if r == openedQuote {
				quoted = false
			}
		}
		return !quoted && r == sep
	})
	if len(fields) > maxSplit {
		out := fields[0 : maxSplit-1]
		out = append(out, strings.Join(fields[maxSplit-1:], string(sep)))
		return out
	}
	return fields
}

func isQuoted(v string) bool {
	return (strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`)) ||
		(strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'"))
}
