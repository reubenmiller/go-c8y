package pagination

import "testing"

// StrategyAuto must be a non-empty value so the go-c8y-cli flag completion lists
// it as a real choice ("auto") rather than a blank entry.
func TestStrategyAuto_IsAuto(t *testing.T) {
	if StrategyAuto != "auto" {
		t.Fatalf("StrategyAuto = %q, want \"auto\" (a non-empty value for CLI completion)", StrategyAuto)
	}
}

func TestResolveTimeStrategy(t *testing.T) {
	// Both the explicit "auto" and the empty zero value resolve to the time keyset.
	for _, k := range []StrategyKind{"", StrategyAuto, StrategyTimeKeyset} {
		s, err := ResolveTimeStrategy(k)
		if err != nil {
			t.Fatalf("ResolveTimeStrategy(%q) error: %v", k, err)
		}
		if _, ok := s.(TimeKeysetStrategy); !ok {
			t.Fatalf("ResolveTimeStrategy(%q) = %T, want TimeKeysetStrategy", k, s)
		}
	}

	if s, err := ResolveTimeStrategy(StrategyOffset); err != nil {
		t.Fatalf("ResolveTimeStrategy(offset) error: %v", err)
	} else if _, ok := s.(OffsetStrategy); !ok {
		t.Fatalf("ResolveTimeStrategy(offset) = %T, want OffsetStrategy", s)
	}

	if _, err := ResolveTimeStrategy(StrategyIDKeyset); err == nil {
		t.Error("ResolveTimeStrategy(id) should error — id keyset does not apply to time-series")
	}
	if _, err := ResolveTimeStrategy("bogus"); err == nil {
		t.Error("ResolveTimeStrategy(bogus) should error")
	}
}
