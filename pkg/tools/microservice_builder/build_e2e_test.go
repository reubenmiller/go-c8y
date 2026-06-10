package microservice_builder

import (
	"archive/tar"
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
)

// Test_Build_EndToEnd builds a minimal image and verifies the produced
// archive. It requires a running docker engine, so it must be enabled
// explicitly:
//
//	C8Y_BUILDER_E2E=1 go test -v -run Test_Build_EndToEnd ./pkg/tools/microservice_builder/
func Test_Build_EndToEnd(t *testing.T) {
	if os.Getenv("C8Y_BUILDER_E2E") == "" {
		t.Skip("set C8Y_BUILDER_E2E=1 to run the docker end-to-end test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if _, err := InspectEngine(ctx); err != nil {
		t.Skipf("docker engine is not available: %v", err)
	}

	tmpDir := t.TempDir()
	contextDir := filepath.Join(tmpDir, "context")
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contextDir, "hello.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dockerFile := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerFile, []byte("FROM scratch\nCOPY hello.txt /hello.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	outputFile, err := Build(ctx, BuildOptions{
		Manifest: microservices.Manifest{
			Name:    "builder-smoke-test",
			Version: "1.0.0",
		},
		DockerFile:   dockerFile,
		BuildContext: contextDir,
		OutputFile:   filepath.Join(tmpDir, "builder-smoke-test.zip"),
		DockerInDocker: DockerInDockerOptions{
			ContainerName: "c8y-microservice-builder-e2e",
			Remove:        true,
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Verify the archive contents
	reader, err := zip.OpenReader(outputFile)
	if err != nil {
		t.Fatalf("could not open archive: %v", err)
	}
	defer reader.Close()

	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}
	if _, ok := files[microservices.ManifestFile]; !ok {
		t.Errorf("archive is missing %s", microservices.ManifestFile)
	}
	imageFile, ok := files["image.tar"]
	if !ok {
		t.Fatal("archive is missing image.tar")
	}

	// Cumulocity only accepts the legacy docker archive format produced by
	// engines using the classic image store: it contains manifest.json and
	// repositories, and index.json (if present) references an image manifest
	// directly. Engines using the containerd image store omit repositories
	// and nest a multi-platform index which Cumulocity cannot load
	in, err := imageFile.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	tarReader := tar.NewReader(in)
	hasManifest := false
	hasRepositories := false
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("could not read image.tar: %v", err)
		}
		switch hdr.Name {
		case "manifest.json":
			hasManifest = true
		case "repositories":
			hasRepositories = true
		case "index.json":
			content, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatalf("could not read index.json: %v", err)
			}
			index := struct {
				Manifests []struct {
					MediaType string `json:"mediaType"`
				} `json:"manifests"`
			}{}
			if err := json.Unmarshal(content, &index); err != nil {
				t.Fatalf("could not parse index.json: %v", err)
			}
			for _, m := range index.Manifests {
				if m.MediaType == "application/vnd.oci.image.index.v1+json" {
					t.Error("image.tar contains a nested OCI image index (containerd image store output) which Cumulocity cannot load")
				}
			}
		}
	}
	if !hasManifest {
		t.Error("image.tar is missing manifest.json, expected a docker archive")
	}
	if !hasRepositories {
		t.Error("image.tar is missing the repositories file, expected classic image store output")
	}
}
