package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"
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
// lengte, lenlt, lenlte, includes, notincludes, version.
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

// Condition compiles a single (property, operator, value) triple into a
// Predicate. This is the programmatic equivalent of one Parse expression for
// callers that have already tokenized and coerced filter values (e.g.
// go-c8y-cli's --filter handling). String values compare as strings; numeric
// and boolean values compare with their native semantics.
func Condition(property, operator string, value any) (Predicate, error) {
	operator = strings.ToLower(strings.TrimSpace(operator))
	switch v := value.(type) {
	case string:
		return compileCondition(property, operator, v, true)
	case bool:
		return compileCondition(property, operator, strconv.FormatBool(v), false)
	case int:
		return compileCondition(property, operator, strconv.Itoa(v), false)
	case int64:
		return compileCondition(property, operator, strconv.FormatInt(v, 10), false)
	case float32:
		return compileCondition(property, operator, strconv.FormatFloat(float64(v), 'f', -1, 64), false)
	case float64:
		return compileCondition(property, operator, strconv.FormatFloat(v, 'f', -1, 64), false)
	default:
		return nil, fmt.Errorf("filter: unsupported value type %T", value)
	}
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

// getter resolves the filtered property of a document.
type getter = func(jsondoc.JSONDoc) gjson.Result

// condValue is a filter value with go-c8y-cli's numeric/boolean coercion
// applied (skipped for quoted values, mirroring AddRawFilters).
type condValue struct {
	raw    string
	num    float64
	isNum  bool
	b      bool
	isBool bool
}

func coerceValue(value string, quoted bool) condValue {
	cv := condValue{raw: value}
	if quoted {
		return cv
	}
	if v, err := strconv.ParseFloat(value, 64); err == nil {
		cv.num, cv.isNum = v, true
	} else if v, err := strconv.ParseBool(value); err == nil {
		cv.b, cv.isBool = v, true
	}
	return cv
}

func compileCondition(property, operator, value string, quoted bool) (Predicate, error) {
	switch operator {
	case "has", "keyin":
		return Exists(keyPath(property, value)), nil
	case "hasnot", "nothas", "missing", "keynotin":
		return Not(Exists(keyPath(property, value))), nil
	}

	get := pathGetter(property)
	cv := coerceValue(value, quoted)

	switch operator {
	case "eq", "=", "==":
		return compileEquals(get, cv), nil
	case "neq", "!=", "<>":
		return Not(compileEquals(get, cv)), nil
	case "gt", ">", "gte", ">=", "lt", "<", "lte", "<=":
		if !cv.isNum {
			return nil, fmt.Errorf("filter: operator %q requires a numeric value, got %q", operator, value)
		}
		return numericCompare(get, operator, cv.num), nil
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
		return compileLike(get, operator, value)
	case "match", "-match", "notmatch", "-notmatch":
		return compileMatch(get, operator, value)
	case "datelt", "datelte", "olderthan", "dategt", "dategte", "newerthan":
		return compileDateCompare(get, operator, value)
	case "leneq", "lenneq", "lengt", "lengte", "lenlt", "lenlte":
		if !cv.isNum {
			return nil, fmt.Errorf("filter: operator %q requires a numeric value, got %q", operator, value)
		}
		return lengthCompare(get, operator, int(cv.num)), nil
	case "version":
		return compileVersion(get, value)
	case "includes", "notincludes":
		return compileIncludes(get, operator, cv), nil
	}
	return nil, fmt.Errorf("filter: unsupported operator %q", operator)
}

func compileEquals(get getter, want condValue) Predicate {
	switch {
	case want.isNum:
		return func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && v.Float() == want.num }
	case want.isBool:
		return func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && v.Bool() == want.b }
	default:
		return func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && v.String() == want.raw }
	}
}

func compileLike(get getter, operator, pattern string) (Predicate, error) {
	re, err := wildcardToRegexp(pattern)
	if err != nil {
		return nil, fmt.Errorf("filter: invalid wildcard pattern %q: %w", pattern, err)
	}
	p := func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && re.MatchString(v.String()) }
	if strings.Contains(operator, "notlike") {
		return Not(p), nil
	}
	return p, nil
}

func compileMatch(get getter, operator, expr string) (Predicate, error) {
	re, err := regexp.Compile("(?ims)" + expr)
	if err != nil {
		return nil, fmt.Errorf("filter: invalid regex %q: %w", expr, err)
	}
	p := func(d jsondoc.JSONDoc) bool { v := get(d); return v.Exists() && re.MatchString(v.String()) }
	if strings.Contains(operator, "notmatch") {
		return Not(p), nil
	}
	return p, nil
}

// compileDateCompare compiles the date operators. References may be absolute
// dates or relative expressions such as "-25h" or "'2026-01-01' + 1d";
// relative references are resolved against now once at compile time.
func compileDateCompare(get getter, operator, value string) (Predicate, error) {
	ref, err := parseTimestamp(value)
	if err != nil {
		return nil, fmt.Errorf("filter: invalid date %q: %w", value, err)
	}
	return func(d jsondoc.JSONDoc) bool {
		v := get(d)
		if !v.Exists() {
			return false
		}
		ts, err := parseTimestamp(v.String())
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
}

func lengthCompare(get getter, operator string, want int) Predicate {
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
	}
}

// compileVersion compiles a semver constraint check, e.g.
// "version >=1.2.0, <2.0.0". String values which are not valid versions
// (including "") are treated as 0.0.0; non-string values never match
// (mirroring go-c8y-cli's legacy gojsonq macro).
func compileVersion(get getter, constraint string) (Predicate, error) {
	constraints, err := version.NewConstraint(constraint)
	if err != nil {
		return nil, fmt.Errorf("filter: invalid version constraint %q: %w", constraint, err)
	}
	fallback := version.Must(version.NewVersion("0.0.0"))
	return func(d jsondoc.JSONDoc) bool {
		v := get(d)
		if v.Type != gjson.String {
			return false
		}
		current, err := version.NewVersion(v.String())
		if err != nil {
			current = fallback
		}
		return constraints.Check(current)
	}, nil
}

func compileIncludes(get getter, operator string, want condValue) Predicate {
	p := func(d jsondoc.JSONDoc) bool {
		found := false
		get(d).ForEach(func(_, item gjson.Result) bool {
			if want.isNum && item.Type == gjson.Number {
				found = item.Float() == want.num
			} else {
				found = item.String() == want.raw
			}
			return !found
		})
		return found
	}
	if operator == "notincludes" {
		return Not(p)
	}
	return p
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
