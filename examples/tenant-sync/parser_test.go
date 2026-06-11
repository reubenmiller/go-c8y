package main

import (
	"os"
	"path/filepath"
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
		// noarch/all architecture variants
		{
			name:        "rpm noarch package",
			filepath:    "/path/to/tedge-mapper-thingsboard-0.0.1-1.noarch.rpm",
			wantName:    "tedge-mapper-thingsboard",
			wantVersion: "0.0.1",
			wantArch:    "noarch",
			wantType:    "rpm",
		},
		{
			name:        "deb all architecture package",
			filepath:    "/path/to/tedge-mapper-thingsboard_0.0.1_all.deb",
			wantName:    "tedge-mapper-thingsboard",
			wantVersion: "0.0.1",
			wantArch:    "all",
			wantType:    "apt",
		},
		{
			name:        "apk noarch package",
			filepath:    "/path/to/tedge-mapper-thingsboard_0.0.1_noarch.apk",
			wantName:    "tedge-mapper-thingsboard",
			wantVersion: "0.0.1",
			wantArch:    "noarch",
			wantType:    "apk",
		},
		// IPK packages (OpenWrt/OpenEmbedded)
		{
			name:        "ipk package standard",
			filepath:    "/path/to/tedge_1.6.2~584+gd629c53_aarch64.ipk",
			wantName:    "tedge",
			wantVersion: "1.6.2~584+gd629c53",
			wantArch:    "aarch64",
			wantType:    "ipk",
		},
		{
			name:        "ipk package all arch",
			filepath:    "/path/to/tedge-mapper_0.0.1_all.ipk",
			wantName:    "tedge-mapper",
			wantVersion: "0.0.1",
			wantArch:    "all",
			wantType:    "ipk",
		},
		// Arch Linux packages
		{
			name:        "arch linux pkg.tar.zst",
			filepath:    "/path/to/pacman-6.0.2-6-x86_64.pkg.tar.zst",
			wantName:    "pacman",
			wantVersion: "6.0.2",
			wantArch:    "x86_64",
			wantType:    "pacman",
		},
		{
			name:        "arch linux pkg.tar.xz hyphenated name",
			filepath:    "/path/to/linux-firmware-20241210.b81c7f9-1-any.pkg.tar.xz",
			wantName:    "linux-firmware",
			wantVersion: "20241210.b81c7f9",
			wantArch:    "any",
			wantType:    "pacman",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseSoftwareFromFilename(tt.filepath, "", "")
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

func TestDecodeSoftwareName(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain name unchanged",
			input: "myapp",
			want:  "myapp",
		},
		{
			name:  "standard percent-encoding decoded",
			input: "c8y%2Fexample",
			want:  "c8y/example",
		},
		{
			name:  "github artifact dot-encoding decoded",
			input: "c8y.2Fexample",
			want:  "c8y/example",
		},
		{
			name:  "at-sign encoding decoded",
			input: "scope.40package",
			want:  "scope@package",
		},
		{
			name:  "colon encoding decoded",
			input: "ns.3Aservice",
			want:  "ns:service",
		},
		{
			name:  "version component not corrupted (python3.10 style)",
			input: "python3.10",
			want:  "python3.10",
		},
		{
			name:  "semver dots not corrupted (1.0.0 style)",
			input: "myapp-1.0.0",
			want:  "myapp-1.0.0",
		},
		{
			name:  "multiple encoded chars decoded",
			input: "c8y.2Forg.2Fpackage",
			want:  "c8y/org/package",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeSoftwareName(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeSoftwareNameInFilename(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		wantName string
	}{
		{
			name:     "deb with percent-encoded slash in name",
			filepath: "/path/to/c8y%2Fexample_1.0.0_arm64.deb",
			wantName: "c8y/example",
		},
		{
			name:     "deb with github-artifact dot-encoded slash in name",
			filepath: "/path/to/c8y.2Fexample_1.0.0_arm64.deb",
			wantName: "c8y/example",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseSoftwareFromFilename(tt.filepath, "", "")
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
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
			info, err := ParseSoftwareFromFilename(tt.filepath, tt.defaultType, "")
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

func TestComposeParser_CanParse(t *testing.T) {
	parser := &ComposeParser{}

	accept := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
		"compose.myapp.yaml",
		"compose.myapp.yml",
		"docker-compose.myapp.yaml",
		"docker-compose.myapp.yml",
		"COMPOSE.YAML", // case-insensitive
		"Docker-Compose.yml",
	}
	reject := []string{
		"compose.yaml.bak",
		"mycompose.yaml",
		"docker-compose", // no extension
		"random.yaml",
		"values.yaml",
	}

	for _, f := range accept {
		assert.True(t, parser.CanParse(f), "expected CanParse=true for %q", f)
	}
	for _, f := range reject {
		assert.False(t, parser.CanParse(f), "expected CanParse=false for %q", f)
	}
}

func TestComposeParser_NameFromFilename(t *testing.T) {
	// These cases derive the name entirely from the filename, so the file does not
	// need to exist on disk.
	parser := &ComposeParser{}

	tests := []struct {
		filename    string
		wantName    string
		wantVersion string
	}{
		{"compose.myapp.yaml", "myapp", "latest"},
		{"compose.myapp.yml", "myapp", "latest"},
		{"docker-compose.myapp.yaml", "myapp", "latest"},
		{"docker-compose.myapp.yml", "myapp", "latest"},
		{"compose.my-service.yaml", "my-service", "latest"},
		// "stack-v2" → extractNameAndVersion splits off the v-prefixed major version
		{"docker-compose.stack-v2.yaml", "stack", "2"},
		// version encoded in the name segment
		{"compose.myapp-1.2.3.yaml", "myapp", "1.2.3"},
		{"docker-compose.myapp-1.2.3.yaml", "myapp", "1.2.3"},
		{"compose.my-service_2.0.0.yaml", "my-service", "2.0.0"},
		{"compose.backend-v1.0.0.yml", "backend", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info, err := parser.Parse("/nonexistent/"+tt.filename, tt.filename)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantVersion, info.Version)
			assert.Equal(t, "container-group", info.SoftwareType)
		})
	}
}

func TestComposeParser_NameFromContent(t *testing.T) {
	// Write a real compose file with a "name" and "services" field.
	dir := t.TempDir()
	filename := "compose.yaml"
	content := "name: my-project\nservices:\n  web:\n    image: nginx\n"
	filePath := filepath.Join(dir, filename)
	require_NoError(t, os.WriteFile(filePath, []byte(content), 0600))

	parser := &ComposeParser{}
	info, err := parser.Parse(filePath, filename)
	assert.NoError(t, err)
	assert.Equal(t, "my-project", info.Name)
	assert.Equal(t, "container-group", info.SoftwareType)
	assert.Equal(t, "latest", info.Version)
}

func TestComposeParser_NameFromParentDir(t *testing.T) {
	// compose.yaml with no "name" field → fall back to parent directory name.
	dir := t.TempDir()
	// Create a subdirectory whose name will be used as the software name.
	projectDir := filepath.Join(dir, "awesome-stack")
	require_NoError(t, os.MkdirAll(projectDir, 0700))
	filename := "compose.yaml"
	content := "services:\n  web:\n    image: nginx\n"
	filePath := filepath.Join(projectDir, filename)
	require_NoError(t, os.WriteFile(filePath, []byte(content), 0600))

	parser := &ComposeParser{}
	info, err := parser.Parse(filePath, filename)
	assert.NoError(t, err)
	assert.Equal(t, "awesome-stack", info.Name)
	assert.Equal(t, "container-group", info.SoftwareType)
}

func TestComposeParser_NonExistentFile_UsesParentDir(t *testing.T) {
	// When the file cannot be read and there's no name segment in the filename,
	// the parser should fall back to the parent directory name gracefully.
	parser := &ComposeParser{}
	info, err := parser.Parse("/projects/my-service/compose.yaml", "compose.yaml")
	assert.NoError(t, err)
	assert.Equal(t, "my-service", info.Name)
	assert.Equal(t, "container-group", info.SoftwareType)
}

func TestComposeParser_ViaParseFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filepath string
		wantName string
		wantType string
	}{
		{
			name:     "compose with name segment",
			filepath: "/path/to/compose.myapp.yaml",
			wantName: "myapp",
			wantType: "container-group",
		},
		{
			name:     "compose with name and version segment",
			filepath: "/path/to/compose.myapp-1.2.3.yaml",
			wantName: "myapp",
			wantType: "container-group",
		},
		{
			name:     "docker-compose with name segment",
			filepath: "/path/to/docker-compose.backend.yml",
			wantName: "backend",
			wantType: "container-group",
		},
		{
			name:     "plain compose.yaml falls back to parent dir",
			filepath: "/projects/my-stack/compose.yaml",
			wantName: "my-stack",
			wantType: "container-group",
		},
		{
			name:     "plain docker-compose.yaml falls back to parent dir",
			filepath: "/projects/infra/docker-compose.yml",
			wantName: "infra",
			wantType: "container-group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseSoftwareFromFilename(tt.filepath, "", "")
			assert.NoError(t, err)
			assert.Equal(t, tt.wantName, info.Name)
			assert.Equal(t, tt.wantType, info.SoftwareType)
		})
	}
}

// require_NoError is a small helper to fail fast on setup errors.
func require_NoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
