package microservice_builder

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Test_UsesContainerdImageStore(t *testing.T) {
	testcases := []struct {
		name   string
		engine EngineInfo
		want   bool
	}{
		{
			name: "containerd image store",
			engine: EngineInfo{
				DriverStatus: [][]string{{"driver-type", "io.containerd.snapshotter.v1"}},
			},
			want: true,
		},
		{
			name: "classic overlay2 store",
			engine: EngineInfo{
				DriverStatus: [][]string{{"Backing Filesystem", "extfs"}, {"Supports d_type", "true"}},
			},
			want: false,
		},
		{
			name:   "no driver status",
			engine: EngineInfo{},
			want:   false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.engine.UsesContainerdImageStore(); got != tc.want {
				t.Errorf("UsesContainerdImageStore() = %v, want %v", got, tc.want)
			}
		})
	}
}

func Test_ProducesCompatibleImageTarballs(t *testing.T) {
	containerd := [][]string{{"driver-type", "io.containerd.snapshotter.v1"}}
	testcases := []struct {
		name   string
		engine EngineInfo
		want   bool
	}{
		{
			name:   "classic image store",
			engine: EngineInfo{ServerVersion: "28.5.1"},
			want:   true,
		},
		{
			name:   "containerd image store before 29.9",
			engine: EngineInfo{ServerVersion: "29.8.3", DriverStatus: containerd},
			want:   false,
		},
		{
			name:   "containerd image store 29.9",
			engine: EngineInfo{ServerVersion: "29.9.0", DriverStatus: containerd},
			want:   true,
		},
		{
			name:   "containerd image store newer major",
			engine: EngineInfo{ServerVersion: "30.0.0", DriverStatus: containerd},
			want:   true,
		},
		{
			name:   "containerd image store unknown version",
			engine: EngineInfo{DriverStatus: containerd},
			want:   false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.engine.ProducesCompatibleImageTarballs(); got != tc.want {
				t.Errorf("ProducesCompatibleImageTarballs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func Test_VersionAtLeast(t *testing.T) {
	testcases := map[string]bool{
		"29.9.0":      true,
		"29.9.0-rc.1": true,
		"v29.10.1":    true,
		"29.10":       true,
		"30.0.0":      true,
		"29.8.9":      false,
		"28.12.0":     false,
		"dev":         false,
		"":            false,
		" 29.9.1-ce ": true,
	}
	for input, want := range testcases {
		if got := versionAtLeast(input, 29, 9); got != want {
			t.Errorf("versionAtLeast(%q, 29, 9) = %v, want %v", input, got, want)
		}
	}
}

func writeTestTarball(t *testing.T, files map[string]string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "image.tar")
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	w := tar.NewWriter(f)
	for name, content := range files {
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return file
}

func Test_VerifyImageTarball(t *testing.T) {
	testcases := []struct {
		name    string
		files   map[string]string
		wantErr bool
	}{
		{
			name: "single image manifest",
			files: map[string]string{
				"index.json":       `{"manifests":[{"mediaType":"application/vnd.oci.image.manifest.v1+json","annotations":{"io.containerd.image.name":"docker.io/library/app:latest"}}]}`,
				"manifest.json":    `[{"Config":"blobs/sha256/abc"}]`,
				"blobs/sha256/abc": `{"architecture":"amd64"}`,
				"blobs/sha256/def": "binary layer content",
			},
		},
		{
			name: "nested index with attestation manifest",
			files: map[string]string{
				"index.json":       `{"manifests":[{"mediaType":"application/vnd.oci.image.index.v1+json"}]}`,
				"blobs/sha256/abc": `{"manifests":[{"platform":{"os":"linux"}},{"annotations":{"vnd.docker.reference.type":"attestation-manifest"},"platform":{"os":"unknown"}}]}`,
			},
			wantErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			err := verifyImageTarball(writeTestTarball(t, tc.files))
			if (err != nil) != tc.wantErr {
				t.Errorf("verifyImageTarball() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func Test_NormalizeArch(t *testing.T) {
	testcases := map[string]string{
		"x86_64":  "amd64",
		"amd64":   "amd64",
		"aarch64": "arm64",
		"arm64":   "arm64",
		"armv7l":  "arm",
		" X86_64": "amd64",
		"riscv64": "riscv64",
	}
	for input, want := range testcases {
		if got := normalizeArch(input); got != want {
			t.Errorf("normalizeArch(%q) = %q, want %q", input, got, want)
		}
	}
}

func Test_PlatformArch(t *testing.T) {
	testcases := map[string]string{
		"linux/amd64":  "amd64",
		"linux/arm64":  "arm64",
		"linux/arm/v7": "arm",
		"amd64":        "amd64",
	}
	for input, want := range testcases {
		if got := platformArch(input); got != want {
			t.Errorf("platformArch(%q) = %q, want %q", input, got, want)
		}
	}
}

func Test_SanitizeName(t *testing.T) {
	testcases := map[string]string{
		"my-app":                      "my-app",
		"myorg/app:1.0":               "myorg-app-1.0",
		"registry.local:5000/app:2.0": "registry.local-5000-app-2.0",
		"../../etc":                   "etc",
		"":                            "image",
		"app name with spaces":        "app-name-with-spaces",
	}
	for input, want := range testcases {
		if got := sanitizeName(input); got != want {
			t.Errorf("sanitizeName(%q) = %q, want %q", input, got, want)
		}
	}
}

func Test_Pack(t *testing.T) {
	manifest := strings.NewReader(`{"name":"my-app"}`)
	image := strings.NewReader("dummy image content")

	buf := &bytes.Buffer{}
	if err := Pack(manifest, image, buf); err != nil {
		t.Fatalf("Pack() error = %v", err)
	}

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("could not read created zip: %v", err)
	}

	want := map[string]string{
		"cumulocity.json": `{"name":"my-app"}`,
		"image.tar":       "dummy image content",
	}
	if len(reader.File) != len(want) {
		t.Fatalf("zip contains %d files, want %d", len(reader.File), len(want))
	}
	for _, file := range reader.File {
		expected, ok := want[file.Name]
		if !ok {
			t.Errorf("unexpected file in zip: %s", file.Name)
			continue
		}
		f, err := file.Open()
		if err != nil {
			t.Fatalf("could not open %s: %v", file.Name, err)
		}
		content, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			t.Fatalf("could not read %s: %v", file.Name, err)
		}
		if string(content) != expected {
			t.Errorf("%s content = %q, want %q", file.Name, content, expected)
		}
	}
}

func Test_ProxyEnv(t *testing.T) {
	t.Setenv("HTTP_PROXY", "http://proxy:3128")
	t.Setenv("HTTPS_PROXY", "")
	t.Setenv("NO_PROXY", "localhost")
	t.Setenv("http_proxy", "")
	t.Setenv("https_proxy", "")
	t.Setenv("no_proxy", "")

	env := proxyEnv()
	want := []string{"HTTP_PROXY=http://proxy:3128", "NO_PROXY=localhost"}
	if len(env) != len(want) {
		t.Fatalf("proxyEnv() = %v, want %v", env, want)
	}
	for i := range want {
		if env[i] != want[i] {
			t.Errorf("proxyEnv()[%d] = %q, want %q", i, env[i], want[i])
		}
	}

	args := proxyBuildArgs()
	if len(args) != 4 || args[0] != "--build-arg" || args[1] != want[0] {
		t.Errorf("proxyBuildArgs() = %v", args)
	}
}
