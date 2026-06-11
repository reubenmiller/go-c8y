package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLatestCount(t *testing.T) {
	testCases := []struct {
		selector string
		count    int
		ok       bool
	}{
		{"latest-1", 1, true},
		{"latest-5", 5, true},
		{"latest-25", 25, true},
		{"latest", 0, false},
		{"latest-0", 0, false},
		{"latest-", 0, false},
		{"all", 0, false},
		{"v1.2.3", 0, false},
		{"latest-abc", 0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.selector, func(t *testing.T) {
			count, ok := latestCount(tc.selector)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.count, count)
		})
	}
}

func TestFilterReleases(t *testing.T) {
	releases := []githubRelease{
		{TagName: "v2.1.0-rc1", Prerelease: true},
		{TagName: "v2.0.0"},
		{TagName: "v1.9.0-draft", Draft: true},
		{TagName: "v1.9.0"},
		{TagName: "v1.8.0"},
	}

	tags := func(releases []githubRelease) []string {
		var result []string
		for _, release := range releases {
			result = append(result, release.TagName)
		}
		return result
	}

	// No limit, stable releases only: drafts and prereleases are dropped
	assert.Equal(t, []string{"v2.0.0", "v1.9.0", "v1.8.0"}, tags(filterReleases(releases, false, 0)))

	// Latest 2 stable releases
	assert.Equal(t, []string{"v2.0.0", "v1.9.0"}, tags(filterReleases(releases, false, 2)))

	// Latest 2 including prereleases
	assert.Equal(t, []string{"v2.1.0-rc1", "v2.0.0"}, tags(filterReleases(releases, true, 2)))

	// Limit larger than the list
	assert.Equal(t, []string{"v2.0.0", "v1.9.0", "v1.8.0"}, tags(filterReleases(releases, false, 10)))
}
