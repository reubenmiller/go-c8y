package devicegroups

import "testing"

func TestRefConstructors(t *testing.T) {
	if got := ByID("123"); got != "123" {
		t.Errorf("ByID = %q, want 123", got)
	}
	if got := ByName("My Group"); got != "name:My Group" {
		t.Errorf("ByName = %q, want name:My Group", got)
	}
}

func TestScopeToGroups(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		want   string
	}{
		{"empty", "", "$filter=(has(c8y_IsDeviceGroup))"},
		{"with filter", "(name eq 'foo')", "$filter=(has(c8y_IsDeviceGroup) and (name eq 'foo'))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScopeToGroups(tt.filter); got != tt.want {
				t.Errorf("ScopeToGroups(%q) = %q, want %q", tt.filter, got, tt.want)
			}
		})
	}
}
