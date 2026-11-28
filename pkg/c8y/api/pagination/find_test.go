package pagination

import (
	"errors"
	"iter"
	"testing"
)

func iterOf(items []string, failAt int) *Iterator[string] {
	return &Iterator[string]{items: func(yield func(string, error) bool) {
		for i, item := range items {
			if i == failAt {
				yield("", errors.New("page failed"))
				return
			}
			if !yield(item, nil) {
				return
			}
		}
	}}
}

var _ iter.Seq2[string, error] = iterOf(nil, -1).Items()

func TestFindFirst(t *testing.T) {
	got, found, err := FindFirst(iterOf([]string{"a", "bb", "cc"}, -1), func(s string) bool { return len(s) == 2 })
	if err != nil || !found || got != "bb" {
		t.Errorf("FindFirst = %q, %v, %v; want bb, true, nil", got, found, err)
	}

	_, found, err = FindFirst(iterOf([]string{"a"}, -1), func(string) bool { return false })
	if err != nil || found {
		t.Errorf("FindFirst without match = %v, %v", found, err)
	}

	_, found, err = FindFirst(iterOf([]string{"a", "b"}, 1), func(s string) bool { return s == "b" })
	if err == nil || found {
		t.Errorf("FindFirst after a failed page = %v, %v; want the error", found, err)
	}

	failed := &Iterator[string]{err: errors.New("cannot start"), items: func(func(string, error) bool) {}}
	if _, _, err := FindFirst(failed, func(string) bool { return true }); err == nil {
		t.Error("FindFirst ignored the error of the iterator")
	}
}

func TestFindWildcard(t *testing.T) {
	it := func() *Iterator[string] { return iterOf([]string{"alpha", "beta-1", "beta-2"}, -1) }
	keys := func(s string) []string { return []string{s, "/" + s} }

	for pattern, want := range map[string]string{"beta-*": "beta-1", "/alp*": "alpha", "beta-?": "beta-1", "alpha": "alpha"} {
		got, found, err := FindWildcard(it(), pattern, keys)
		if err != nil || !found || got != want {
			t.Errorf("FindWildcard(%q) = %q, %v, %v; want %q", pattern, got, found, err, want)
		}
	}
	if _, found, _ := FindWildcard(it(), "gamma*", keys); found {
		t.Error("FindWildcard matched a pattern without a match")
	}
}
