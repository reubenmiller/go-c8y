package source

import "testing"

func TestParseRef(t *testing.T) {
	cases := map[string]string{
		"":                     "",
		"12345":                "12345",
		"dev01":                "name:dev01",
		"name:dev01":           "name:dev01",
		"ext:c8y_Serial:ABC":   "ext:c8y_Serial:ABC",
		"query:name eq 'x'":    "query:name eq 'x'",
		"id:1":                 "id:1",
		"83001,1.0.0":          "83001",
		"my app, other":        "name:my app",
		" 42 ":                 "42",
		"weird:name":           "name:weird:name",
		"contextPath:cockpit":  "name:contextPath:cockpit",
	}
	for in, want := range cases {
		if got := ParseRef(in); got != want {
			t.Errorf("ParseRef(%q) = %q, want %q", in, got, want)
		}
	}
	if got := ParseRef("contextPath:cockpit", "contextPath"); got != "contextPath:cockpit" {
		t.Errorf("extra scheme not kept: %q", got)
	}
}
