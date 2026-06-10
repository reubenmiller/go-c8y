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
        type: apt
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
	require.Len(t, manifest.DeviceProfiles[0].Software, 1)
	assert.Equal(t, "apt", manifest.DeviceProfiles[0].Software[0].Type)
}

func TestLoadManifestApplicationSubscribed(t *testing.T) {
	path := writeManifest(t, `
applications:
  - name: app-default
  - name: app-on
    subscribed: true
  - name: app-off
    subscribed: false
`)

	manifest, err := LoadManifest(path)
	require.NoError(t, err)
	require.Len(t, manifest.Applications, 3)
	assert.True(t, manifest.Applications[0].IsSubscribed())
	assert.True(t, manifest.Applications[1].IsSubscribed())
	assert.False(t, manifest.Applications[2].IsSubscribed())
}

func TestLoadManifestTargets(t *testing.T) {
	path := writeManifest(t, `
targets:
  current: true
  allChildren: true
  tenants: [t12345, child.example.com]
  selector:
    domain: "*.iot.example.com"
    company: ACME
  credentials:
    mode: sessions
    sessionHome: /tmp/sessions
`)

	manifest, err := LoadManifest(path)
	require.NoError(t, err)
	targets := manifest.Targets
	require.NotNil(t, targets)
	assert.True(t, targets.IncludesCurrent())
	assert.True(t, targets.HasRemoteSelection())
	assert.True(t, targets.AllChildren)
	assert.Equal(t, []string{"t12345", "child.example.com"}, targets.Tenants)
	require.NotNil(t, targets.Selector)
	assert.Equal(t, "*.iot.example.com", targets.Selector.Domain)
	assert.Equal(t, "ACME", targets.Selector.Company)
	assert.Equal(t, CredentialsModeSessions, targets.CredentialsMode())
	assert.Equal(t, "/tmp/sessions", targets.Credentials.SessionHome)
}

func TestLoadManifestCommandsAndSectionHooks(t *testing.T) {
	path := writeManifest(t, `
commands:
  - name: mycustom1
    actions:
      - c8y devices create --name "foo"
      - c8y devices create --name "bar"
  - name: mycustom2
    actions:
      - c8y devices create --name "foo2"

hooks:
  pre:
    - run: echo global-pre
  sections:
    software:
      pre:
        - run: echo before software
      post:
        - run: echo after software
`)

	manifest, err := LoadManifest(path)
	require.NoError(t, err)

	require.Len(t, manifest.Commands, 2)
	assert.Equal(t, "mycustom1", manifest.Commands[0].Name)
	assert.Len(t, manifest.Commands[0].Actions, 2)
	assert.Equal(t, []string{`c8y devices create --name "foo2"`}, manifest.Commands[1].Actions)

	require.Contains(t, manifest.Hooks.Sections, "software")
	assert.Len(t, manifest.Hooks.Sections["software"].Pre, 1)
	assert.Len(t, manifest.Hooks.Sections["software"].Post, 1)
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
			name: "trusted certificate url source",
			content: `
trustedCertificates:
  - source:
      url: https://example.com/device-ca.pem
`,
			errorMsg: "url sources are not supported",
		},
		{
			name: "trusted certificate unknown status",
			content: `
trustedCertificates:
  - status: PAUSED
    source:
      path: ./certificates
`,
			errorMsg: "unknown status",
		},
		{
			name: "certificate revocation list url source",
			content: `
certificateRevocationLists:
  - source:
      url: https://example.com/revoked.csv
`,
			errorMsg: "url sources are not supported",
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
			name: "targets selector requires criteria",
			content: `
targets:
  selector: {}
`,
			errorMsg: "selector requires at least one of",
		},
		{
			name: "targets must select something",
			content: `
targets:
  current: false
`,
			errorMsg: "no tenants selected",
		},
		{
			name: "targets credentials mode is checked",
			content: `
targets:
  allChildren: true
  credentials:
    mode: magic
`,
			errorMsg: `unknown credentials mode "magic"`,
		},
		{
			name: "command group requires name",
			content: `
commands:
  - actions: ["echo hello"]
`,
			errorMsg: "name is required",
		},
		{
			name: "command group requires actions",
			content: `
commands:
  - name: empty
`,
			errorMsg: "at least one action is required",
		},
		{
			name: "command group names must be unique",
			content: `
commands:
  - name: dup
    actions: ["echo a"]
  - name: dup
    actions: ["echo b"]
`,
			errorMsg: `duplicate group name "dup"`,
		},
		{
			name: "command actions must not be blank",
			content: `
commands:
  - name: blank
    actions: ["echo a", "  "]
`,
			errorMsg: "action must not be empty",
		},
		{
			name: "section hooks only for known sections",
			content: `
hooks:
  sections:
    softwarez:
      pre:
        - run: echo x
`,
			errorMsg: "unknown section",
		},
		{
			name: "section hook requires run",
			content: `
hooks:
  sections:
    software:
      pre:
        - name: incomplete
`,
			errorMsg: "run is required",
		},
		{
			name: "retention rule requires dataType",
			content: `
retentionRules:
  - maximumAge: 365
`,
			errorMsg: "dataType is required",
		},
		{
			name: "retention rule dataType is checked",
			content: `
retentionRules:
  - dataType: MEASUREMENTS
    maximumAge: 365
`,
			errorMsg: `unknown dataType "MEASUREMENTS"`,
		},
		{
			name: "retention rule requires maximumAge",
			content: `
retentionRules:
  - dataType: MEASUREMENT
`,
			errorMsg: "maximumAge must be at least 1",
		},
		{
			name: "retention rule selectors must be unique",
			content: `
retentionRules:
  - dataType: MEASUREMENT
    maximumAge: 365
  - dataType: MEASUREMENT
    maximumAge: 30
`,
			errorMsg: "duplicate rule selector MEASUREMENT/*/*/*",
		},
		{
			name: "smartrest source is required",
			content: `
smartrestTemplates:
  - name: custom_devmgmt
`,
			errorMsg: "source requires one of",
		},
		{
			name: "smartrest url sources are rejected",
			content: `
smartrestTemplates:
  - source:
      url: https://example.com/collection.json
`,
			errorMsg: "url sources are not supported",
		},
		{
			name: "user group requires name",
			content: `
userGroups:
  - description: missing name
`,
			errorMsg: "name is required",
		},
		{
			name: "user group names must be unique",
			content: `
userGroups:
  - name: dup
  - name: dup
`,
			errorMsg: `duplicate group name "dup"`,
		},
		{
			name: "user group roles must not be blank",
			content: `
userGroups:
  - name: operators
    roles: ["ROLE_INVENTORY_READ", " "]
`,
			errorMsg: "role must not be empty",
		},
		{
			name: "user requires userName",
			content: `
users:
  - email: jdoe@example.com
`,
			errorMsg: "userName is required",
		},
		{
			name: "usernames must be unique",
			content: `
users:
  - userName: dup@example.com
  - userName: dup@example.com
`,
			errorMsg: `duplicate userName "dup@example.com"`,
		},
		{
			name: "password reset email requires email",
			content: `
users:
  - userName: jdoe
    sendPasswordResetEmail: true
`,
			errorMsg: "sendPasswordResetEmail requires email",
		},
		{
			name: "user groups must not be blank",
			content: `
users:
  - userName: jdoe
    groups: ["operators", ""]
`,
			errorMsg: "group must not be empty",
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
