package main

import "testing"

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
	// Two paths differing only in parameter name must normalize equal.
	if normalizePath("/x/{id}") != normalizePath("/x/{deviceId}") {
		t.Errorf("parameter names should normalize equal")
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

func TestStringEnumSkipsNonString(t *testing.T) {
	if _, ok := stringEnum(Schema{Enum: []any{1, 2, 3}}); ok {
		t.Errorf("numeric enum should be skipped")
	}
	if vals, ok := stringEnum(Schema{Enum: []any{"A", "B"}}); !ok || len(vals) != 2 {
		t.Errorf("string enum should be collected")
	}
}
