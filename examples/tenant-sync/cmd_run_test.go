package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTargetsFromFlags(t *testing.T) {
	t.Run("no flags returns nil", func(t *testing.T) {
		spec, err := targetsFromFlags(nil, false, "", false)
		require.NoError(t, err)
		assert.Nil(t, spec)
	})

	t.Run("targets are split on commas", func(t *testing.T) {
		spec, err := targetsFromFlags([]string{"t1,t2", " t3 "}, false, "", false)
		require.NoError(t, err)
		require.NotNil(t, spec)
		assert.Equal(t, []string{"t1", "t2", "t3"}, spec.Tenants)
		assert.False(t, spec.IncludesCurrent())
	})

	t.Run("selector and include-current", func(t *testing.T) {
		spec, err := targetsFromFlags(nil, true, "domain=*.x.com,company=ACME", true)
		require.NoError(t, err)
		require.NotNil(t, spec)
		assert.True(t, spec.AllChildren)
		require.NotNil(t, spec.Selector)
		assert.Equal(t, "*.x.com", spec.Selector.Domain)
		assert.Equal(t, "ACME", spec.Selector.Company)
		assert.True(t, spec.IncludesCurrent())
	})

	t.Run("invalid selector", func(t *testing.T) {
		_, err := targetsFromFlags(nil, false, "domain", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected key=value")

		_, err = targetsFromFlags(nil, false, "owner=x", false)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown --target-selector key "owner"`)
	})
}

func TestMergeTargetOverrides(t *testing.T) {
	manifestSpec := &TargetsSpec{
		AllChildren: true,
		Credentials: &TargetCredentials{Mode: CredentialsModeSessions, SessionHome: "/sessions"},
	}

	t.Run("no overrides keeps the manifest untouched", func(t *testing.T) {
		assert.Nil(t, mergeTargetOverrides(manifestSpec, nil, ""))
	})

	t.Run("flags replace the selection but keep manifest credentials", func(t *testing.T) {
		spec := mergeTargetOverrides(manifestSpec, &TargetsSpec{Tenants: []string{"t1"}}, "")
		require.NotNil(t, spec)
		assert.False(t, spec.AllChildren)
		assert.Equal(t, []string{"t1"}, spec.Tenants)
		assert.Equal(t, CredentialsModeSessions, spec.CredentialsMode())
	})

	t.Run("credentials mode overrides the manifest without mutating it", func(t *testing.T) {
		spec := mergeTargetOverrides(manifestSpec, nil, CredentialsModeServiceUser)
		require.NotNil(t, spec)
		assert.True(t, spec.AllChildren)
		assert.Equal(t, CredentialsModeServiceUser, spec.CredentialsMode())
		assert.Equal(t, "/sessions", spec.Credentials.SessionHome)
		// The manifest spec keeps its own credentials config
		assert.Equal(t, CredentialsModeSessions, manifestSpec.CredentialsMode())
	})

	t.Run("credentials mode alone with no manifest targets", func(t *testing.T) {
		spec := mergeTargetOverrides(nil, nil, CredentialsModeSessions)
		require.NotNil(t, spec)
		assert.Equal(t, CredentialsModeSessions, spec.CredentialsMode())
		assert.True(t, spec.IncludesCurrent())
	})
}
