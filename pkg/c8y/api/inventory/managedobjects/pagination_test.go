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
		opts  any
		want  want
	}{
		{"unset (zero value) -> id keyset", "", "", nil, wantID},
		{"explicit auto -> id keyset", pagination.StrategyAuto, "", nil, wantID},
		{"auto with conflicting orderby -> offset", pagination.StrategyAuto, "$filter=(type eq 'x') $orderby=name", nil, wantOffset},
		{"auto with _id orderby -> id keyset", pagination.StrategyAuto, "$orderby=_id asc", nil, wantID},
		{"explicit offset", pagination.StrategyOffset, "", nil, wantOffset},
		{"explicit id", pagination.StrategyIDKeyset, "", nil, wantID},
		{"explicit id with conflicting orderby -> error", pagination.StrategyIDKeyset, "$orderby=name", nil, wantErr},
		{"time keyset not applicable -> error", pagination.StrategyTimeKeyset, "", nil, wantErr},
		{"unknown -> error", pagination.StrategyKind("bogus"), "", nil, wantErr},
		{"auto with type filter -> offset", pagination.StrategyAuto, "", ListOptions{Type: "x"}, wantOffset},
		{"auto with fragmentType filter -> offset", "", "", ListOptions{FragmentType: "c8y_IsDevice"}, wantOffset},
		{"auto with query-only filter -> id keyset", "", "$filter=(name eq 'x')", ListOptions{Query: "$filter=(name eq 'x')"}, wantID},
		{"explicit id with type filter -> error", pagination.StrategyIDKeyset, "", ListOptions{Type: "x"}, wantErr},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := ResolveListStrategy(tc.kind, tc.query, tc.opts)
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
