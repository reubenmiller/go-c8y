package agents

import "testing"

func TestRefConstructors(t *testing.T) {
	if got := ByID("123"); got != "123" {
		t.Errorf("ByID = %q, want 123", got)
	}
	if got := ByName("My Agent"); got != "name:My Agent" {
		t.Errorf("ByName = %q, want name:My Agent", got)
	}
}

func TestScopeToAgents(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		want   string
	}{
		{"empty", "", "$filter=(has(com_cumulocity_model_Agent))"},
		{"with filter", "(name eq 'foo')", "$filter=(has(com_cumulocity_model_Agent) and (name eq 'foo'))"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScopeToAgents(tt.filter); got != tt.want {
				t.Errorf("ScopeToAgents(%q) = %q, want %q", tt.filter, got, tt.want)
			}
		})
	}
}
