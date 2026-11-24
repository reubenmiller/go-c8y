package dataprep_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/dataprep"
)

func samplePackage() *dataprep.RulePackage {
	return dataprep.NewPackage(dataprep.RuleConfig{
		SmartFunctionFile: "transform.js",
		Description:       "Temperature readings to measurements",
		Tags:              []string{"sensor", "temperature"},
		Input: dataprep.Input{
			Transport:       dataprep.TransportMQTT,
			TopicPattern:    "sensors/temperature/*",
			ClientIDPattern: "*",
		},
	}).
		WithSmartFunction("export function onMessage(msg, ctx) { return []; }\n").
		WithTest("reading", dataprep.TestCase{
			Inputs: []dataprep.TestInput{{
				Payload: `{"temperature":25.5}`, PayloadFormat: "json",
				Topic: "sensors/temperature/s01", ClientID: "s01", Time: "2026-01-01T12:00:00.000Z",
			}},
			ExpectedOutput: []map[string]any{{"cumulocityType": "measurement"}},
		}).
		WithTest("inlet/outlet", dataprep.TestCase{
			Inputs: []dataprep.TestInput{{
				Payload: "dGVzdA==", PayloadFormat: "base64",
				Topic: "sensors/temperature/s02", ClientID: "s02", Time: "2026-01-01T12:00:00Z",
			}},
		})
}

func readArchive(t *testing.T, b []byte) map[string]string {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	files := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(tr)
		files[hdr.Name] = buf.String()
	}
	return files
}

func TestArchiveLayoutAndRoundTrip(t *testing.T) {
	pkg := samplePackage()
	pkg.Config.Extra = map[string]any{"futureField": "kept"}

	a1, err := pkg.Archive()
	if err != nil {
		t.Fatal(err)
	}
	a2, _ := pkg.Archive()
	if !bytes.Equal(a1, a2) {
		t.Error("Archive is not reproducible")
	}

	files := readArchive(t, a1)
	var names []string
	for n := range files {
		names = append(names, n)
	}
	slices.Sort(names)
	want := []string{"data-prep.yaml", "tests/inlet%2Foutlet.yaml", "tests/reading.yaml", "transform.js"}
	if !slices.Equal(names, want) {
		t.Errorf("entries = %v, want %v", names, want)
	}
	cfg := files["data-prep.yaml"]
	for _, s := range []string{"smartFunctionFile: transform.js", "topicPattern: sensors/temperature/*", "futureField: kept"} {
		if !strings.Contains(cfg, s) {
			t.Errorf("data-prep.yaml missing %q:\n%s", s, cfg)
		}
	}
	if strings.Contains(cfg, "disabled") {
		t.Errorf("disabled: false should be omitted:\n%s", cfg)
	}

	back, err := dataprep.PackageFromArchive(bytes.NewReader(a1))
	if err != nil {
		t.Fatal(err)
	}
	if back.Checksum() != pkg.Checksum() {
		t.Error("checksum changed after round trip")
	}
	if !slices.Equal(back.TestNames(), []string{"inlet/outlet", "reading"}) {
		t.Errorf("tests = %v", back.TestNames())
	}
	if back.Config.Extra["futureField"] != "kept" {
		t.Errorf("unknown field lost: %v", back.Config.Extra)
	}
}

func TestDisabledIsWrittenLowercase(t *testing.T) {
	pkg := samplePackage()
	pkg.Config.Disabled = true
	files := readArchive(t, must(pkg.Archive()))
	if !strings.Contains(files["data-prep.yaml"], "disabled: true") {
		t.Errorf("data-prep.yaml:\n%s", files["data-prep.yaml"])
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		change func(p *dataprep.RulePackage)
		want   string
	}{
		{"valid", func(p *dataprep.RulePackage) {}, ""},
		{"transport", func(p *dataprep.RulePackage) { p.Config.Input.Transport = "http" }, `input.transport must be "mqtt"`},
		{"double star", func(p *dataprep.RulePackage) { p.Config.Input.TopicPattern = "a/**" }, "must not contain **"},
		{"mqtt wildcard", func(p *dataprep.RulePackage) { p.Config.Input.TopicPattern = "a/+/b" }, "MQTT wildcards"},
		{"no code", func(p *dataprep.RulePackage) { p.SmartFunction = nil }, "is empty"},
		{"nested code file", func(p *dataprep.RulePackage) { p.Config.SmartFunctionFile = "dist/x.js" }, "archive root"},
		{"payload format", func(p *dataprep.RulePackage) {
			tc := p.Tests["reading"]
			tc.Inputs[0].PayloadFormat = "xml"
			p.Tests["reading"] = tc
		}, "payloadFormat must be"},
		{"time zone", func(p *dataprep.RulePackage) {
			tc := p.Tests["reading"]
			tc.Inputs[0].Time = "2026-01-01T12:00:00"
			p.Tests["reading"] = tc
		}, "time zone"},
		{"entry limit", func(p *dataprep.RulePackage) { p.Limits = &dataprep.Limits{MaxEntries: 3} }, "4 entries, the limit is 3"},
		{"file size limit", func(p *dataprep.RulePackage) {
			p.Limits = &dataprep.Limits{MaxFileSize: 10}
		}, "transform.js is"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := samplePackage()
			tt.change(p)
			err := p.Validate()
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, dataprep.ErrInvalidPackage) || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.want)
			}
			if _, err := p.Archive(); err == nil {
				t.Error("Archive must refuse an invalid package")
			}
		})
	}
}

func TestTestNameEncoding(t *testing.T) {
	tests := map[string]string{
		"inlet/outlet": "inlet%2Foutlet",
		".hidden":      "%2Ehidden",
		"100%":         "100%25",
		"a:b*c":        "a%3Ab%2Ac",
		"plain-name":   "plain-name",
	}
	for name, file := range tests {
		if got := dataprep.EncodeTestName(name); got != file {
			t.Errorf("Encode(%q) = %q, want %q", name, got, file)
		}
		if got := dataprep.DecodeTestName(file); got != name {
			t.Errorf("Decode(%q) = %q, want %q", file, got, name)
		}
	}
	if got := dataprep.DecodeTestName("a%zz"); got != "a%zz" {
		t.Errorf("invalid escape changed: %q", got)
	}
}

func TestDirRoundTrip(t *testing.T) {
	dir := t.TempDir()
	pkg := samplePackage()
	if err := pkg.WriteDir(dir); err != nil {
		t.Fatal(err)
	}
	// Files outside the package layout are ignored when reading.
	_ = os.MkdirAll(filepath.Join(dir, "src"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "src", "transform.ts"), []byte("ts"), 0o644)

	back, err := dataprep.PackageFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if back.Checksum() != pkg.Checksum() {
		t.Errorf("entries after round trip = %v", back.Entries())
	}
}

func TestPackageFromArchiveRejectsTraversal(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "../evil.js", Mode: 0o644, Size: 1, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("x"))
	_ = tw.Close()
	_ = gz.Close()
	if _, err := dataprep.PackageFromArchive(&buf); !errors.Is(err, dataprep.ErrInvalidPackage) {
		t.Errorf("err = %v", err)
	}
}

func TestInputHelpers(t *testing.T) {
	in := dataprep.BinaryInput("t", "c", []byte("test"))
	if in.PayloadFormat != "base64" || in.Payload != "dGVzdA==" || !strings.HasSuffix(in.Time, "Z") {
		t.Errorf("binary input = %+v", in)
	}
	if dataprep.TextInput("t", "c", "x").PayloadFormat != "text" || dataprep.JSONInput("t", "c", "{}").PayloadFormat != "json" {
		t.Error("wrong payload formats")
	}
}
