package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathIdent(t *testing.T) {
	cases := map[string]string{
		"/alarm/alarms":                                      "AlarmAlarms",
		"/alarm/alarms/{id}":                                 "AlarmAlarmsID",
		"/alarm/alarms/count":                                "AlarmAlarmsCount",
		"/inventory/managedObjects/{id}":                     "InventoryManagedObjectsID",
		"/application/applications/{id}/binaries/{binaryId}": "ApplicationApplicationsIDBinariesBinaryID",
		"/.well-known/est/simpleenroll":                      "WellKnownESTSimpleenroll",
		"/":                                                  "Root",
	}
	for path, want := range cases {
		if got := pathIdent(path); got != want {
			t.Errorf("pathIdent(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/alarm/alarms/{id}", "/alarm/alarms/{}"},
		{"/alarm/alarms/{alarmId}", "/alarm/alarms/{}"},
		{"/tenant/loginOptions/{typeOrId}/accessMappings/{id}", "/tenant/loginOptions/{}/accessMappings/{}"},
		{"/tenant/statistics/device/", "/tenant/statistics/device"},
	}
	for _, c := range cases {
		if got := normalizePath(c.in); got != c.want {
			t.Errorf("normalizePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// Two paths differing only in parameter name must normalise equal.
	if normalizePath("/x/{id}") != normalizePath("/x/{deviceId}") {
		t.Errorf("parameter names should normalise equal")
	}
}

func TestLooksLikeAPIPath(t *testing.T) {
	yes := []string{"/alarm/alarms", "/alarm/alarms/{id}", "/inventory/managedObjects/{id}/childDevices"}
	no := []string{"/", "/alarm", "application/json", "/foo.go", "/a b", "https://x/y", "/x/*"}
	for _, p := range yes {
		if !looksLikeAPIPath(p) {
			t.Errorf("looksLikeAPIPath(%q) = false, want true", p)
		}
	}
	for _, p := range no {
		if looksLikeAPIPath(p) {
			t.Errorf("looksLikeAPIPath(%q) = true, want false", p)
		}
	}
}

func TestEnumValueIdent(t *testing.T) {
	cases := map[string]string{
		"CLEARED":          "Cleared",
		"ACKNOWLEDGED":     "Acknowledged",
		"application/json": "ApplicationJSON",
		"c8y_Serial":       "C8ySerial",
		"":                 "Empty",
	}
	for in, want := range cases {
		if got := enumValueIdent(in); got != want {
			t.Errorf("enumValueIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCollectEnumsDedup(t *testing.T) {
	doc := &OAS{}
	doc.Components.Schemas = map[string]Schema{
		"alarm": {Properties: map[string]Schema{
			"severity": {Type: "string", Enum: []any{"CRITICAL", "MAJOR"}},
			"count":    {Type: "integer"},
		}},
	}
	groups := collectEnums(doc)
	if len(groups) != 1 {
		t.Fatalf("expected 1 enum group, got %d", len(groups))
	}
	g := groups[0]
	if g.Name != "AlarmSeverity" {
		t.Errorf("group name = %q, want AlarmSeverity", g.Name)
	}
	if len(g.Values) != 2 || g.Values[0].Ident != "AlarmSeverityCritical" {
		t.Errorf("unexpected values: %+v", g.Values)
	}
}

func TestResolveParamAndScalarType(t *testing.T) {
	doc := &OAS{}
	doc.Components.Parameters = map[string]Parameter{
		"q_src": {Name: "source", In: "query", Schema: Schema{Type: "string"}},
	}
	p := doc.resolveParam(Parameter{Ref: "#/components/parameters/q_src"})
	if p.Name != "source" || p.In != "query" {
		t.Fatalf("resolveParam did not follow ref: %+v", p)
	}

	cases := []struct {
		s       Schema
		goType  string
		urlOpts string
	}{
		{Schema{Type: "string"}, "string", ",omitempty"},
		{Schema{Type: "string", Format: "date-time"}, "time.Time", ",omitempty,omitzero"},
		{Schema{Type: "integer"}, "int", ",omitempty"},
		{Schema{Type: "boolean"}, "bool", ",omitempty"},
		{Schema{Type: "array", Items: &Schema{Type: "string"}}, "[]string", ",omitempty"},
	}
	for _, c := range cases {
		gt, uo, ok := doc.goScalarType(c.s)
		if !ok || gt != c.goType || uo != c.urlOpts {
			t.Errorf("goScalarType(%+v) = %q,%q,%v; want %q,%q,true", c.s, gt, uo, ok, c.goType, c.urlOpts)
		}
	}
	if _, _, ok := doc.goScalarType(Schema{Type: "object"}); ok {
		t.Errorf("object type should not be mapped")
	}
}

func TestOptionFieldsOverridesAndSkips(t *testing.T) {
	doc := &OAS{}
	doc.Paths = map[string]PathItem{
		"/alarm/alarms": {Get: &Operation{Parameters: []Parameter{
			{Name: "pageSize", In: "query", Schema: Schema{Type: "integer"}}, // pagination: skipped
			{Name: "source", In: "query", Schema: Schema{Type: "string"}},    // override type
			{Name: "status", In: "query", Schema: Schema{Type: "array", Items: &Schema{Type: "string"}}},
			{Name: "bodyish", In: "header", Schema: Schema{Type: "string"}}, // non-query: skipped
		}}},
	}
	spec := optionSpec{
		Path: "/alarm/alarms", Method: "GET",
		FieldType: map[string]string{"source": "managedobjects.DeviceRef", "status": "[]model.AlarmStatus"},
	}
	op, _ := doc.operation("/alarm/alarms", "GET")
	fields, _, err := optionFields(doc, spec, op)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("want 2 fields (source, status), got %d: %+v", len(fields), fields)
	}
	byName := map[string]optionField{}
	for _, f := range fields {
		byName[f.Name] = f
	}
	if byName["Source"].Type != "managedobjects.DeviceRef" || byName["Source"].Tag != "source,omitempty" {
		t.Errorf("source override wrong: %+v", byName["Source"])
	}
	if byName["Status"].Type != "[]model.AlarmStatus" {
		t.Errorf("status override wrong: %+v", byName["Status"])
	}
}

func TestRenderModelAccessors(t *testing.T) {
	doc := &OAS{}
	doc.Components.Schemas = map[string]Schema{
		"desc_self": {Type: "string"},
		"alarm": {Properties: map[string]Schema{
			"id":     {Type: "string"},
			"count":  {Type: "integer"},
			"time":   {Type: "string", Format: "date-time"},
			"self":   {Ref: "#/components/schemas/desc_self"},
			"source": {Type: "object"}, // skipped (nested + SkipProps)
		}},
	}
	out, err := renderModel(doc, "test", modelSpec{TypeName: "Alarm", Schema: "alarm", SkipProps: map[string]bool{"source": true}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"func (m Alarm) ID() string { return m.Get(\"id\").String() }",
		"func (m Alarm) Count() int64 { return m.Get(\"count\").Int() }",
		"func (m Alarm) Time() time.Time { return m.Get(\"time\").Time() }",
		"func (m Alarm) Self() string { return m.Get(\"self\").String() }",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("model output missing:\n  %s", want)
		}
	}
	if strings.Contains(out, "func (m Alarm) Source") {
		t.Errorf("source should be skipped (nested object)")
	}
}

func TestLoadOverlay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overlay.yml")
	content := `
resources:
  - package: alarms
    options:
      - type: ListOptions
        path: /alarm/alarms
        method: GET
        doc: |-
          ListOptions doc.
        fields:
          source:
            type: managedobjects.DeviceRef
            doc: Source device.
          status:
            type: "[]model.AlarmStatus"
        embeds:
          - import: x/pagination
            type: pagination.PaginationOptions
        imports:
          - x/model
    models:
      - type: Alarm
        schema: alarm
        skip: [source]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	resources, err := LoadOverlay(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].Pkg != "alarms" {
		t.Fatalf("unexpected resources: %+v", resources)
	}
	opt := resources[0].Options[0]
	if opt.TypeName != "ListOptions" || opt.Path != "/alarm/alarms" || opt.Method != "GET" {
		t.Errorf("option header wrong: %+v", opt)
	}
	if opt.FieldType["source"] != "managedobjects.DeviceRef" || opt.FieldType["status"] != "[]model.AlarmStatus" {
		t.Errorf("field type overrides wrong: %+v", opt.FieldType)
	}
	if opt.FieldDoc["source"] != "Source device." {
		t.Errorf("field doc override wrong: %+v", opt.FieldDoc)
	}
	if len(opt.Embeds) != 1 || opt.Embeds[0].Type != "pagination.PaginationOptions" {
		t.Errorf("embeds wrong: %+v", opt.Embeds)
	}
	m := resources[0].Models[0]
	if m.TypeName != "Alarm" || m.Schema != "alarm" || !m.SkipProps["source"] {
		t.Errorf("model spec wrong: %+v", m)
	}
}

func TestRenderOptionsExtraFields(t *testing.T) {
	doc := &OAS{Paths: map[string]PathItem{
		"/audit/auditRecords": {Get: &Operation{Parameters: []Parameter{
			{Name: "type", In: "query", Schema: Schema{Type: "string"}},
		}}},
	}}
	r := resource{Pkg: "auditrecords", Options: []optionSpec{{
		TypeName: "ListOptions",
		Path:     "/audit/auditRecords",
		Method:   "GET",
		Extra: []extraField{
			{Name: "Revert", Type: "bool", Tag: "revert,omitempty", Doc: "Not in the OAS; server supports it."},
		},
	}}}
	out, err := renderOptions(doc, "test", r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Type string `url:\"type,omitempty\"`") {
		t.Errorf("missing OAS-derived field:\n%s", out)
	}
	if !strings.Contains(out, "Revert bool `url:\"revert,omitempty\"`") {
		t.Errorf("missing extra field:\n%s", out)
	}
	if !strings.Contains(out, "not present in the OpenAPI spec") {
		t.Errorf("missing extra-field marker comment:\n%s", out)
	}
}

func TestLoadOverlayMissingFileIsEmpty(t *testing.T) {
	resources, err := LoadOverlay(filepath.Join(t.TempDir(), "does-not-exist.yml"))
	if err != nil {
		t.Fatalf("missing overlay should not error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("missing overlay should yield no resources, got %d", len(resources))
	}
}

func TestDriftWaiverMatching(t *testing.T) {
	patterns := normalizePatterns([]string{
		"/meta/*",
		"/tenant/loginOptions/{typeOrId}/accessMappings",
		"/.well-known/est/*",
		"/user/passwordReset",
	})
	match := []string{
		"/meta/connect",
		"/meta/subscribe",
		"/tenant/loginOptions/{}/accessMappings", // {typeOrId} normalised to {}
		"/.well-known/est/simpleenroll",
		"/user/passwordReset",
	}
	noMatch := []string{
		"/meta", // prefix needs the trailing slash content
		"/tenant/loginOptions/{}/accessMappings/{}", // more specific, not the exact pattern
		"/user/password",
	}
	for _, p := range match {
		if !matchesAny(p, patterns) {
			t.Errorf("expected %q to be waived", p)
		}
	}
	for _, p := range noMatch {
		if matchesAny(p, patterns) {
			t.Errorf("expected %q NOT to be waived", p)
		}
	}
}

func TestLintWaiversSuppressDrift(t *testing.T) {
	doc := &OAS{Paths: map[string]PathItem{
		"/alarm/alarms":    {Get: &Operation{}},
		"/identity/search": {Post: &Operation{}}, // missing in SDK, will be waived
	}}
	dir := t.TempDir()
	// SDK source with one path literal that is in the OAS, plus one extra (non-OAS).
	src := "package p\nvar a = \"/alarm/alarms\"\nvar b = \"/service/remoteaccess/devices/{id}/configurations\"\n"
	if err := os.WriteFile(filepath.Join(dir, "api.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}

	// Without waivers: drift on both sides.
	res, err := Lint(doc, dir, driftWaivers{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.HasDrift() || len(res.MissingInSDK) != 1 || len(res.ExtraInSDK) != 1 {
		t.Fatalf("expected unwaived drift, got %+v", res)
	}

	// With waivers: both suppressed.
	res, err = Lint(doc, dir, driftWaivers{
		IgnoreMissing: []string{"/identity/search"},
		IgnoreExtra:   []string{"/service/remoteaccess/*"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.HasDrift() {
		t.Errorf("expected no undeclared drift, got %+v", res)
	}
	if res.WaivedCount != 2 {
		t.Errorf("expected 2 waived, got %d", res.WaivedCount)
	}
}

func TestStringEnumSkipsNonString(t *testing.T) {
	if _, ok := stringEnum(Schema{Enum: []any{1, 2, 3}}); ok {
		t.Errorf("numeric enum should be skipped")
	}
	if vals, ok := stringEnum(Schema{Enum: []any{"A", "B"}}); !ok || len(vals) != 2 {
		t.Errorf("string enum should be collected")
	}
}
