package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestPathFromArgs(t *testing.T) {
	path, err := manifestPathFromArgs("", nil)
	require.NoError(t, err)
	assert.Equal(t, "tenant.yaml", path)

	path, err = manifestPathFromArgs("custom.yaml", nil)
	require.NoError(t, err)
	assert.Equal(t, "custom.yaml", path)

	path, err = manifestPathFromArgs("", []string{"positional.yaml"})
	require.NoError(t, err)
	assert.Equal(t, "positional.yaml", path)

	_, err = manifestPathFromArgs("a.yaml", []string{"b.yaml"})
	assert.Error(t, err)

	_, err = manifestPathFromArgs("", []string{"a.yaml", "b.yaml"})
	assert.Error(t, err)
}

func TestLoadManifestStrictParsing(t *testing.T) {
	path := writeManifest(t, "softwares: []\n")
	_, err := LoadManifest(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "softwares")
}

func TestLoadManifestEmpty(t *testing.T) {
	path := writeManifest(t, "")
	manifest, err := LoadManifest(path)
	require.NoError(t, err)
	assert.Empty(t, manifest.Software)
}

func TestInitTemplatesAreValid(t *testing.T) {
	// The minimal init template must produce a valid manifest
	minimal := writeManifest(t, minimalTemplate)
	_, err := LoadManifest(minimal)
	require.NoError(t, err, "minimal init template must validate")

	// The embedded full example must be valid too
	full := writeManifest(t, manifestTemplate)
	_, err = LoadManifest(full)
	require.NoError(t, err, "full example manifest must validate")
}
