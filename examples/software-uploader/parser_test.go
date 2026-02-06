package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSoftwareFromFilename(t *testing.T) {
	tests := []struct {
		name        string
		filepath    string
		wantName    string
		wantVersion string
		wantArch    string
		wantType    string
	}{
		// Generic semver patterns
		{
			name:        "semver with dash separator",
			filepath:    "/path/to/myapp-1.2.3.tar.gz",
			wantName:    "myapp",
			wantVersion: "1.2.3",
			wantType:    "archive",
		},
		{
			name:        "semver with underscore and v prefix",
			filepath:    "/path/to/device-firmware_v2.0.1.bin",
			wantName:    "device-firmware",
			wantVersion: "2.0.1",
			wantType:    "binary",
		},
		// Debian packages
		{
			name:        "debian package with tilde and plus in version",
			filepath:    "/path/to/tedge-flows_1.6.2~584+gd629c53_arm64.deb",
			wantName:    "tedge-flows",
			wantVersion: "1.6.2~584+gd629c53",
			wantArch:    "arm64",
			wantType:    "apt",
		},
		{
			name:        "debian package simple",
			filepath:    "/path/to/myapp_1.0.0_amd64.deb",
			wantName:    "myapp",
			wantVersion: "1.0.0",
			wantArch:    "amd64",
			wantType:    "apt",
		},
		// RPM packages
		{
			name:        "rpm package with complex version",
			filepath:    "/path/to/tedge-flows-1.6.2~584+gd629c53-1.aarch64.rpm",
			wantName:    "tedge-flows",
			wantVersion: "1.6.2~584+gd629c53",
			wantArch:    "aarch64",
			wantType:    "rpm",
		},
		{
			name:        "rpm package simple",
			filepath:    "/path/to/myapp-1.0.0-1.x86_64.rpm",
			wantName:    "myapp",
			wantVersion: "1.0.0",
			wantArch:    "x86_64",
			wantType:    "rpm",
		},
		// APK packages
		{
			name:        "apk package with rc version",
			filepath:    "/path/to/tedge-flows_1.6.2_rc584+gd629c53-r0_aarch64.apk",
			wantName:    "tedge-flows",
			wantVersion: "1.6.2_rc584+gd629c53",
			wantArch:    "aarch64",
			wantType:    "apk",
		},
		{
			name:        "apk package simple",
			filepath:    "/path/to/myapp_1.0.0-r1_x86_64.apk",
			wantName:    "myapp",
			wantVersion: "1.0.0",
			wantArch:    "x86_64",
			wantType:    "apk",
		},
		// Tar.gz with architecture
		{
			name:        "tar.gz with musl target",
			filepath:    "/path/to/tedge_1.6.2-rc584+gd629c53_aarch64-unknown-linux-musl.tar.gz",
			wantName:    "tedge",
			wantVersion: "1.6.2-rc584+gd629c53",
			wantArch:    "aarch64",
			wantType:    "archive",
		},
		{
			name:        "tar.gz simple",
			filepath:    "/path/to/myapp-1.2.3.tar.gz",
			wantName:    "myapp",
			wantVersion: "1.2.3",
			wantType:    "archive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseSoftwareFromFilename(tt.filepath, "")
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name, "name mismatch")
			assert.Equal(t, tt.wantVersion, info.Version, "version mismatch")
			if tt.wantArch != "" {
				assert.Equal(t, tt.wantArch, info.Architecture, "architecture mismatch")
			}
			if tt.wantType != "" {
				assert.Equal(t, tt.wantType, info.SoftwareType, "type mismatch")
			}
		})
	}
}

func TestStripExtensions(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"myapp-1.0.0.tar.gz", "myapp-1.0.0"},
		{"software.zip", "software"},
		{"firmware.bin", "firmware"},
		{"app.tar.bz2", "app"},
		{"package.tar.xz", "package"},
		{"noext", "noext"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripExtensions(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractNameAndVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
	}{
		{"myapp-1.2.3", "myapp", "1.2.3"},
		{"myapp_v1.2.3", "myapp", "1.2.3"},
		{"myapp.v1.2.3", "myapp", "1.2.3"},
		{"myapp-v1.2.3", "myapp", "1.2.3"},
		{"my-app-1.2.3", "my-app", "1.2.3"},
		{"com.example.app-1.0.0", "com.example.app", "1.0.0"},
		{"app-1.0", "app", "1.0"},
		{"app-v5", "app", "5"},
		{"noversion", "noversion", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotName, gotVersion := extractNameAndVersion(tt.input)
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}

func TestGroupBySoftwareName(t *testing.T) {
	infos := []*SoftwareInfo{
		{Name: "app1", Version: "1.0.0"},
		{Name: "app1", Version: "1.1.0"},
		{Name: "app2", Version: "2.0.0"},
		{Name: "app1", Version: "1.2.0"},
		{Name: "app3", Version: "3.0.0"},
	}

	groups := GroupBySoftwareName(infos)

	assert.Len(t, groups, 3)
	assert.Len(t, groups["app1"], 3)
	assert.Len(t, groups["app2"], 1)
	assert.Len(t, groups["app3"], 1)
}

func TestCreateSummary(t *testing.T) {
	infos := []*SoftwareInfo{
		{Name: "app1", Version: "1.0.0"},
		{Name: "app1", Version: "1.1.0"},
		{Name: "app2", Version: "2.0.0"},
	}

	summary := CreateSummary(infos)

	assert.Equal(t, 3, summary.TotalFiles)
	assert.Equal(t, 2, summary.TotalSoftware)
	assert.Equal(t, 3, summary.TotalVersions)
}

func TestValidateSoftwareInfo(t *testing.T) {
	tests := []struct {
		name    string
		info    *SoftwareInfo
		wantErr bool
	}{
		{
			name:    "valid info",
			info:    &SoftwareInfo{Name: "app", Version: "1.0.0"},
			wantErr: false,
		},
		{
			name:    "empty name",
			info:    &SoftwareInfo{Name: "", Version: "1.0.0"},
			wantErr: true,
		},
		{
			name:    "empty version",
			info:    &SoftwareInfo{Name: "app", Version: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSoftwareInfo(tt.info)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDetectSoftwareType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"myapp-1.0.0.deb", "apt"},
		{"package-2.1.0.rpm", "rpm"},
		{"app-3.0.0.apk", "apk"},
		{"library-1.0.0.jar", "java"},
		{"webapp-2.0.0.war", "java"},
		{"installer-1.0.0.msi", "windows"},
		{"setup-2.0.0.exe", "windows"},
		{"application-1.0.0.dmg", "macos"},
		{"installer-1.0.0.pkg", "macos"},
		{"myapp-1.0.0.snap", "snap"},
		{"app-1.0.0.flatpak", "flatpak"},
		{"tool-1.0.0.AppImage", "appimage"},
		{"software-1.0.0.tar.gz", "archive"},
		{"package-1.0.0.zip", "archive"},
		{"data-1.0.0.tgz", "archive"},
		{"firmware-1.0.0.bin", "binary"},
		{"unknown-1.0.0.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := detectSoftwareType(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseSoftwareFromFilename_WithAutoDetect(t *testing.T) {
	tests := []struct {
		name        string
		filepath    string
		defaultType string
		wantType    string
	}{
		{
			name:        "auto-detect deb package",
			filepath:    "/path/to/myapp_1.0.0_amd64.deb",
			defaultType: "",
			wantType:    "apt",
		},
		{
			name:        "auto-detect rpm package",
			filepath:    "/path/to/myapp-1.0.0-1.x86_64.rpm",
			defaultType: "",
			wantType:    "rpm",
		},
		{
			name:        "user-specified type overrides auto-detect",
			filepath:    "/path/to/myapp-1.0.0.deb",
			defaultType: "custom-type",
			wantType:    "custom-type",
		},
		{
			name:        "auto-detect archive",
			filepath:    "/path/to/myapp-1.0.0.tar.gz",
			defaultType: "",
			wantType:    "archive",
		},
		{
			name:        "auto-detect binary",
			filepath:    "/path/to/firmware-2.0.1.bin",
			defaultType: "",
			wantType:    "binary",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseSoftwareFromFilename(tt.filepath, tt.defaultType)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantType, info.SoftwareType)
		})
	}
}

func TestDebianParser(t *testing.T) {
	parser := &DebianParser{}

	tests := []struct {
		name        string
		filename    string
		wantName    string
		wantVersion string
		wantArch    string
	}{
		{
			name:        "standard format",
			filename:    "tedge-flows_1.6.2~584+gd629c53_arm64.deb",
			wantName:    "tedge-flows",
			wantVersion: "1.6.2~584+gd629c53",
			wantArch:    "arm64",
		},
		{
			name:        "simple format",
			filename:    "myapp_1.0.0_amd64.deb",
			wantName:    "myapp",
			wantVersion: "1.0.0",
			wantArch:    "amd64",
		},
		{
			name:        "missing architecture",
			filename:    "myapp_1.0.0.deb",
			wantName:    "myapp",
			wantVersion: "1.0.0",
			wantArch:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parser.Parse("/path/to/"+tt.filename, tt.filename)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantVersion, info.Version)
			assert.Equal(t, tt.wantArch, info.Architecture)
			assert.Equal(t, "apt", info.SoftwareType)
		})
	}
}

func TestRPMParser(t *testing.T) {
	parser := &RPMParser{}

	tests := []struct {
		name        string
		filename    string
		wantName    string
		wantVersion string
		wantArch    string
		wantRelease string
	}{
		{
			name:        "standard format",
			filename:    "tedge-flows-1.6.2~584+gd629c53-1.aarch64.rpm",
			wantName:    "tedge-flows",
			wantVersion: "1.6.2~584+gd629c53",
			wantArch:    "aarch64",
			wantRelease: "1",
		},
		{
			name:        "name with dashes",
			filename:    "my-package-name-1.0.0-1.x86_64.rpm",
			wantName:    "my-package-name",
			wantVersion: "1.0.0",
			wantArch:    "x86_64",
			wantRelease: "1",
		},
		{
			name:        "noarch package",
			filename:    "myapp-2.0.0-1.noarch.rpm",
			wantName:    "myapp",
			wantVersion: "2.0.0",
			wantArch:    "noarch",
			wantRelease: "1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parser.Parse("/path/to/"+tt.filename, tt.filename)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantVersion, info.Version)
			assert.Equal(t, tt.wantArch, info.Architecture)
			if tt.wantRelease != "" {
				assert.Equal(t, tt.wantRelease, info.Metadata["release"])
			}
			assert.Equal(t, "rpm", info.SoftwareType)
		})
	}
}

func TestAPKParser(t *testing.T) {
	parser := &APKParser{}

	tests := []struct {
		name        string
		filename    string
		wantName    string
		wantVersion string
		wantArch    string
		wantRelease string
	}{
		{
			name:        "standard format",
			filename:    "tedge-flows_1.6.2_rc584+gd629c53-r0_aarch64.apk",
			wantName:    "tedge-flows",
			wantVersion: "1.6.2_rc584+gd629c53",
			wantArch:    "aarch64",
			wantRelease: "r0",
		},
		{
			name:        "simple format with x86_64",
			filename:    "myapp_1.0.0-r1_x86_64.apk",
			wantName:    "myapp",
			wantVersion: "1.0.0",
			wantArch:    "x86_64",
			wantRelease: "r1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parser.Parse("/path/to/"+tt.filename, tt.filename)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantVersion, info.Version)
			assert.Equal(t, tt.wantArch, info.Architecture)
			if tt.wantRelease != "" {
				assert.Equal(t, tt.wantRelease, info.Metadata["release"])
			}
			assert.Equal(t, "apk", info.SoftwareType)
		})
	}
}

func TestTarGzParser(t *testing.T) {
	parser := &TarGzParser{}

	tests := []struct {
		name        string
		filename    string
		wantName    string
		wantVersion string
		wantArch    string
	}{
		{
			name:        "with musl target",
			filename:    "tedge_1.6.2-rc584+gd629c53_aarch64-unknown-linux-musl.tar.gz",
			wantName:    "tedge",
			wantVersion: "1.6.2-rc584+gd629c53",
			wantArch:    "aarch64",
		},
		{
			name:        "simple tar.gz",
			filename:    "myapp-1.2.3.tar.gz",
			wantName:    "myapp",
			wantVersion: "1.2.3",
			wantArch:    "",
		},
		{
			name:        "with gnu target",
			filename:    "myapp-1.0.0-x86_64-unknown-linux-gnu.tar.gz",
			wantName:    "myapp",
			wantVersion: "1.0.0",
			wantArch:    "x86_64",
		},
		{
			name:        "simple with architecture",
			filename:    "myapp-2.0.0_aarch64.tar.gz",
			wantName:    "myapp",
			wantVersion: "2.0.0",
			wantArch:    "aarch64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parser.Parse("/path/to/"+tt.filename, tt.filename)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantVersion, info.Version)
			if tt.wantArch != "" {
				assert.Equal(t, tt.wantArch, info.Architecture)
			}
			assert.Equal(t, "archive", info.SoftwareType)
		})
	}
}
