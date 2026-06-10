package microservice_builder

import (
	"archive/zip"
	"bytes"
	"io"
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
