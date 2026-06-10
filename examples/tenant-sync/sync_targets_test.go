package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargetsSpecSelection(t *testing.T) {
	boolPtr := func(v bool) *bool { return &v }

	testCases := []struct {
		name            string
		spec            *TargetsSpec
		includesCurrent bool
		hasRemote       bool
	}{
		{name: "nil spec", spec: nil, includesCurrent: true, hasRemote: false},
		{name: "empty spec", spec: &TargetsSpec{}, includesCurrent: true, hasRemote: false},
		{name: "all children excludes current by default", spec: &TargetsSpec{AllChildren: true}, includesCurrent: false, hasRemote: true},
		{name: "explicit tenants exclude current by default", spec: &TargetsSpec{Tenants: []string{"t1"}}, includesCurrent: false, hasRemote: true},
		{name: "selector excludes current by default", spec: &TargetsSpec{Selector: &TenantSelector{Domain: "*.x.com"}}, includesCurrent: false, hasRemote: true},
		{name: "current can be forced on", spec: &TargetsSpec{AllChildren: true, Current: boolPtr(true)}, includesCurrent: true, hasRemote: true},
		{name: "current only", spec: &TargetsSpec{Current: boolPtr(true)}, includesCurrent: true, hasRemote: false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.includesCurrent, tc.spec.IncludesCurrent())
			assert.Equal(t, tc.hasRemote, tc.spec.HasRemoteSelection())
		})
	}
}

func TestTargetsSpecCredentialsMode(t *testing.T) {
	assert.Equal(t, CredentialsModeServiceUser, (*TargetsSpec)(nil).CredentialsMode())
	assert.Equal(t, CredentialsModeServiceUser, (&TargetsSpec{}).CredentialsMode())
	assert.Equal(t, CredentialsModeSessions,
		(&TargetsSpec{Credentials: &TargetCredentials{Mode: CredentialsModeSessions}}).CredentialsMode())
}

func TestIsDomainReference(t *testing.T) {
	assert.False(t, isDomainReference("t12345"))
	assert.False(t, isDomainReference("management"))
	assert.True(t, isDomainReference("child.example.com"))
}

func TestMatchesDomainGlob(t *testing.T) {
	assert.True(t, matchesDomainGlob("", "anything.example.com"))
	assert.True(t, matchesDomainGlob("*.iot.example.com", "tenant-a.iot.example.com"))
	assert.False(t, matchesDomainGlob("*.iot.example.com", "tenant-a.example.com"))
	assert.True(t, matchesDomainGlob("tenant-a.example.com", "tenant-a.example.com"))
	// Invalid patterns fall back to an exact comparison
	assert.False(t, matchesDomainGlob("[invalid", "x"))
	assert.True(t, matchesDomainGlob("[invalid", "[invalid"))
}

func TestDryRunTargets(t *testing.T) {
	t.Run("default is the current tenant", func(t *testing.T) {
		targets := dryRunTargets(nil)
		assert.Len(t, targets, 1)
		assert.True(t, targets[0].Current)
		assert.Equal(t, "current", targets[0].Label())
	})

	t.Run("selection modes are described", func(t *testing.T) {
		targets := dryRunTargets(&TargetsSpec{
			AllChildren: true,
			Tenants:     []string{"t1", "child.example.com"},
			Selector:    &TenantSelector{Domain: "*.x.com", Company: "ACME"},
		})
		labels := make([]string, 0, len(targets))
		for _, target := range targets {
			labels = append(labels, target.Label())
		}
		assert.Equal(t, []string{
			"all child tenants",
			"t1",
			"child.example.com",
			"tenants matching domain=*.x.com,company=ACME",
		}, labels)
	})
}
