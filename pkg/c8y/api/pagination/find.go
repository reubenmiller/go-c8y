package pagination

import "github.com/reubenmiller/go-c8y/v2/pkg/matcher"

// FindFirst returns the first item of it for which match is true. found is
// false when no item matches. The error is the error of the iterator, or of
// the first page which failed to load; items after a failed page are not
// searched.
//
//	app, found, err := pagination.FindFirst(it, func(a jsonmodels.Application) bool {
//		return a.Name() == "cockpit"
//	})
func FindFirst[T any](it *Iterator[T], match func(T) bool) (item T, found bool, err error) {
	if err := it.Err(); err != nil {
		return item, false, err
	}
	for candidate, err := range it.Items() {
		if err != nil {
			return item, false, err
		}
		if match(candidate) {
			return candidate, true, nil
		}
	}
	return item, false, nil
}

// FindWildcard returns the first item of it where one of the values returned by
// keys matches pattern. The pattern supports the "*" and "?" wildcards (see
// [matcher.MatchWithWildcards]), so it is the client-side counterpart of a
// name lookup which the server cannot filter.
func FindWildcard[T any](it *Iterator[T], pattern string, keys func(T) []string) (item T, found bool, err error) {
	return FindFirst(it, func(candidate T) bool {
		for _, key := range keys(candidate) {
			if ok, _ := matcher.MatchWithWildcards(key, pattern); ok {
				return true
			}
		}
		return false
	})
}
