package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeManifest(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

func TestLoadManifest(t *testing.T) {
	path := writeManifest(t, `
tenantOptions:
  - category: configuration
    key: my.setting
    value: "true"

features:
  - key: feature-a
  - key: feature-b
    enabled: false

applications:
  - name: advanced-software-mgmt

software:
  - source:
      path: ./packages
      patterns: ["*.deb", "*.rpm"]
    typeMap:
      - "*.bin=firmware"

firmware:
  - name: rpi4-image
    deviceType: raspberrypi4
    source:
      github:
        repo: example/repo
        release: latest
        assets: ["*.wic.xz"]

configuration:
  - name: mosquitto
    configurationType: mosquitto.conf
    source:
      path: ./config/mosquitto.conf

deviceProfiles:
  - name: base-profile
    deviceType: raspberrypi4
    firmware:
      name: rpi4-image
      version: 1.0.0
    software:
      - name: tedge
        version: 1.6.0
`)

	manifest, err := LoadManifest(path)
	require.NoError(t, err)

	assert.Len(t, manifest.TenantOptions, 1)
	assert.Equal(t, "true", manifest.TenantOptions[0].Value)

	require.Len(t, manifest.Features, 2)
	assert.True(t, manifest.Features[0].IsEnabled())
	assert.False(t, manifest.Features[1].IsEnabled())

	require.Len(t, manifest.Software, 1)
	mappings := manifest.Software[0].TypeMappings()
	require.Len(t, mappings, 1)
	assert.Equal(t, "*.bin", mappings[0].Pattern)
	assert.Equal(t, "firmware", mappings[0].SoftwareType)

	require.Len(t, manifest.Firmware, 1)
	assert.Equal(t, "example/repo", manifest.Firmware[0].Source.GitHub.Repo)

	require.Len(t, manifest.DeviceProfiles, 1)
	assert.Equal(t, "1.0.0", manifest.DeviceProfiles[0].Firmware.Version)
}

func TestLoadManifestExpandsEnvVars(t *testing.T) {
	t.Setenv("TEST_SYNC_TOKEN", "secret-token")

	path := writeManifest(t, `
firmware:
  - name: image
    source:
      github:
        repo: example/repo
        token: ${TEST_SYNC_TOKEN}
`)

	manifest, err := LoadManifest(path)
	require.NoError(t, err)
	assert.Equal(t, "secret-token", manifest.Firmware[0].Source.GitHub.Token)
}

func TestLoadManifestValidation(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		errorMsg string
	}{
		{
			name: "missing source",
			content: `
software:
  - namePrefix: x
`,
			errorMsg: "source requires one of",
		},
		{
			name: "conflicting source",
			content: `
software:
  - source:
      path: ./a
      url: https://example.com/b
`,
			errorMsg: "mutually exclusive",
		},
		{
			name: "bad github repo",
			content: `
firmware:
  - name: image
    source:
      github:
        repo: invalid
`,
			errorMsg: "owner/repo",
		},
		{
			name: "configuration requires type",
			content: `
configuration:
  - name: settings
    source:
      path: ./settings.json
`,
			errorMsg: "configurationType is required",
		},
		{
			name: "firmware url source requires name",
			content: `
firmware:
  - source:
      url: https://example.com/image.wic
`,
			errorMsg: "name is required",
		},
		{
			name: "profile firmware requires version",
			content: `
deviceProfiles:
  - name: profile
    firmware:
      name: image
`,
			errorMsg: "name and version are required",
		},
		{
			name: "tenant option value and valueFrom are exclusive",
			content: `
tenantOptions:
  - category: c8y
    key: a.b
    value: "5"
    valueFrom:
      application: devicemanagement
`,
			errorMsg: "mutually exclusive",
		},
		{
			name: "valueFrom requires exactly one reference",
			content: `
tenantOptions:
  - category: c8y
    key: a.b
    valueFrom:
      application: devicemanagement
      device: mydevice
`,
			errorMsg: "exactly one of",
		},
		{
			name: "unknown deviceType placeholder",
			content: `
firmware:
  - deviceType: "{nme}"
    source:
      path: ./images
`,
			errorMsg: "unknown placeholder {nme}",
		},
		{
			name: "hook requires run",
			content: `
hooks:
  post:
    - name: incomplete
`,
			errorMsg: "run is required",
		},
		{
			name: "application source is validated",
			content: `
applications:
  - name: myapp
    source:
      path: ./app.zip
      url: https://example.com/app.zip
`,
			errorMsg: "mutually exclusive",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeManifest(t, tc.content)
			_, err := LoadManifest(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.errorMsg)
		})
	}
}
