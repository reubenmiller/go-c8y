package managedobjects

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
)

func TestResolveListStrategy(t *testing.T) {
	type want int
	const (
		wantID want = iota
		wantOffset
		wantErr
	)

	cases := []struct {
		name  string
		kind  pagination.StrategyKind
		query string
		want  want
	}{
		{"unset (zero value) -> id keyset", "", "", wantID},
		{"explicit auto -> id keyset", pagination.StrategyAuto, "", wantID},
		{"auto with conflicting orderby -> offset", pagination.StrategyAuto, "$filter=(type eq 'x') $orderby=name", wantOffset},
		{"auto with _id orderby -> id keyset", pagination.StrategyAuto, "$orderby=_id asc", wantID},
		{"explicit offset", pagination.StrategyOffset, "", wantOffset},
		{"explicit id", pagination.StrategyIDKeyset, "", wantID},
		{"explicit id with conflicting orderby -> error", pagination.StrategyIDKeyset, "$orderby=name", wantErr},
		{"time keyset not applicable -> error", pagination.StrategyTimeKeyset, "", wantErr},
		{"unknown -> error", pagination.StrategyKind("bogus"), "", wantErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ResolveListStrategy(tc.kind, tc.query)
			switch tc.want {
			case wantErr:
				if err == nil {
					t.Fatalf("expected error, got strategy %T", s)
				}
			case wantID:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if _, ok := s.(pagination.IDKeysetStrategy); !ok {
					t.Fatalf("got %T, want IDKeysetStrategy", s)
				}
			case wantOffset:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if _, ok := s.(pagination.OffsetStrategy); !ok {
					t.Fatalf("got %T, want OffsetStrategy", s)
				}
			}
		})
	}
}
