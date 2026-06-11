package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeSessionFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600))
}

func TestLoadSessionFiles(t *testing.T) {
	dir := t.TempDir()
	writeSessionFile(t, dir, "tenant-a.json", `{
  "host": "https://tenant-a.iot.example.com",
  "tenant": "t100",
  "username": "admin",
  "password": "secret-a"
}`)
	writeSessionFile(t, dir, "tenant-b.yaml", `
host: tenant-b.iot.example.com
tenant: t200
username: admin
password: secret-b
`)
	// Ignored: no credentials, unknown extension, not parseable
	writeSessionFile(t, dir, "settings.json", `{"settings": {"defaultUsername": "x"}}`)
	writeSessionFile(t, dir, "notes.txt", "not a session")
	writeSessionFile(t, dir, "broken.json", "{not json")

	sessions, err := loadSessionFiles(dir)
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, "t100", sessions[0].Tenant)
	assert.Equal(t, "secret-a", sessions[0].Password)
	assert.Equal(t, "t200", sessions[1].Tenant)
}

func TestLoadSessionFilesMissingDir(t *testing.T) {
	_, err := loadSessionFiles(filepath.Join(t.TempDir(), "does-not-exist"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "session directory")
}

func TestMatchSession(t *testing.T) {
	sessions := []goc8ycliSession{
		{Host: "https://tenant-a.iot.example.com", Tenant: "t100", Username: "admin", Password: "a"},
		{Host: "https://tenant-b.iot.example.com:443/", Username: "admin", Password: "b"},
	}

	t.Run("by tenant id", func(t *testing.T) {
		match := matchSession(sessions, Target{TenantID: "t100"})
		require.NotNil(t, match)
		assert.Equal(t, "a", match.Password)
	})

	t.Run("by domain against the session host", func(t *testing.T) {
		match := matchSession(sessions, Target{TenantID: "t200", Domain: "tenant-b.iot.example.com"})
		require.NotNil(t, match)
		assert.Equal(t, "b", match.Password)
	})

	t.Run("no match", func(t *testing.T) {
		assert.Nil(t, matchSession(sessions, Target{TenantID: "t999", Domain: "other.example.com"}))
	})
}

func TestHostMatchesDomain(t *testing.T) {
	assert.True(t, hostMatchesDomain("https://a.example.com", "a.example.com"))
	assert.True(t, hostMatchesDomain("a.example.com", "a.example.com"))
	assert.True(t, hostMatchesDomain("https://a.example.com:8443/path", "a.example.com"))
	assert.True(t, hostMatchesDomain("HTTPS://A.EXAMPLE.COM", "a.example.com"))
	assert.False(t, hostMatchesDomain("https://b.example.com", "a.example.com"))
}

func TestStringSetEqual(t *testing.T) {
	assert.True(t, stringSetEqual(nil, nil))
	assert.True(t, stringSetEqual([]string{"a", "b"}, []string{"b", "a"}))
	assert.True(t, stringSetEqual([]string{"a", "a", "b"}, []string{"b", "a"}))
	assert.False(t, stringSetEqual([]string{"a"}, []string{"a", "b"}))
	assert.False(t, stringSetEqual([]string{"a", "c"}, []string{"a", "b"}))
}
