package microservice_builder

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "embed"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
)

//go:embed build-dind.sh
var BuildScript string

type BuildWith int

const (
	BuildWithAuto BuildWith = iota
	BuildWithNative
	BuildWithDockerInDocker
)

type BuildOptions struct {
	Manifest     microservices.Manifest
	DockerFile   string
	BuildContext string
	OutputFile   string
	Image        string
	BuildArgs    []string

	// BuildWith type to build using native tooling or docker-in-docker.
	BuildWith BuildWith
}

type DockerEngineInfo struct {
	DriverStatus [][]string `json:"DriverStatus,omitempty"`
}

func CheckDockerEngineDriverType() (bool, error) {
	proc := exec.Command("docker", "info", "-f", "json")
	proc.Env = os.Environ()
	out, err := proc.Output()
	if err != nil {
		return false, err
	}

	engine := new(DockerEngineInfo)
	if err := json.Unmarshal(out, engine); err != nil {
		return false, err
	}
	for _, item := range engine.DriverStatus {
		if len(item) >= 2 {
			if strings.EqualFold(item[0], "driver-type") {
				if strings.EqualFold(item[1], "io.containerd.snapshotter.v1") {
					slog.Warn("Detected incompatible docker engine driver-type", "value", item[1])
					return false, nil
				}
			}
		}
	}
	return true, nil
}

func Build(opts BuildOptions) (string, error) {
	b, err := json.Marshal(opts.Manifest)
	if err != nil {
		return "", err
	}
	manifestReader := bytes.NewReader(b)

	if err := PrepareImage(); err != nil {
		return "", err
	}

	if opts.Manifest.Version == "" {
		opts.Manifest.Version = "0.0.1-SNAPSHOT"
	}

	if opts.OutputFile == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		opts.OutputFile = filepath.Join(cwd, fmt.Sprintf("%s_%s.zip", opts.Manifest.Name, opts.Manifest.Version))
	}
	outputFile := opts.OutputFile

	if opts.Image == "" {
		opts.Image = opts.Manifest.Name
	}

	engineCompatible, engineErr := CheckDockerEngineDriverType()
	if engineErr != nil {
		return "", engineErr
	}

	if opts.BuildWith == BuildWithAuto {
		if engineCompatible {
			opts.BuildWith = BuildWithNative
		} else {
			opts.BuildWith = BuildWithDockerInDocker
		}
	}

	// Use docker save
	tarball, err := os.CreateTemp("", "image.tar")
	if err != nil {
		return outputFile, err
	}
	defer tarball.Close()

	switch opts.BuildWith {
	case BuildWithDockerInDocker:
		slog.Info("Building using docker-in-docker")
		if err := BuildImageWithDockerInDocker(BuildImageOptions{
			DockerFile:      opts.DockerFile,
			BuildContext:    opts.BuildContext,
			Image:           opts.Image,
			ExtraDockerArgs: opts.BuildArgs,
			ImageOutput:     tarball.Name(),
		}); err != nil {
			return outputFile, err
		}
	case BuildWithNative:
		slog.Info("Building using native docker")
		if err := BuildImage(BuildImageOptions{
			DockerFile:      opts.DockerFile,
			BuildContext:    opts.BuildContext,
			Image:           opts.Image,
			ExtraDockerArgs: opts.BuildArgs,
			ImageOutput:     tarball.Name(),
		}); err != nil {
			return outputFile, err
		}
	}

	// Create archive
	if err := os.MkdirAll(filepath.Dir(opts.OutputFile), 0755); err != nil {
		return outputFile, err
	}
	archive, err := os.OpenFile(opts.OutputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return outputFile, err
	}
	defer archive.Close()

	tarballIn, err := os.Open(tarball.Name())
	if err != nil {
		return outputFile, err
	}
	defer tarballIn.Close()
	if err := Pack(manifestReader, tarballIn, archive); err != nil {
		return outputFile, err
	}

	return outputFile, nil
}

type BuildImageOptions struct {
	DockerFile      string
	BuildContext    string
	Image           string
	ExtraDockerArgs []string
	ImageOutput     string
}

// BuildImage microservice
func BuildImage(opt BuildImageOptions) error {
	if opt.DockerFile == "" {
		opt.DockerFile = "Dockerfile"
	}
	if opt.BuildContext == "" {
		opt.BuildContext = "."
	}

	args := make([]string, 0)
	args = append(args, "buildx", "build", "--load", "-f", opt.DockerFile, "--platform=linux/amd64")
	args = append(args, detectProxySettings()...)
	args = append(args, opt.ExtraDockerArgs...)
	args = append(args, "-t", opt.Image, opt.BuildContext)

	proc := exec.Command("docker", args...)
	proc.Env = os.Environ()
	proc.Env = append(proc.Env, "BUILDX_NO_DEFAULT_ATTESTATIONS=1")
	proc.Env = append(proc.Env, "BUILDX_METADATA_PROVENANCE=0")
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Run(); err != nil {
		return err
	}
	file, err := os.Open(opt.ImageOutput)
	if err != nil {
		return err
	}
	return ExportImage(opt.Image, file)
}

// NOTE: This currently does not work!
func BuildImageWithDockerBuildXExporter(opt BuildImageOptions) error {
	if opt.DockerFile == "" {
		opt.DockerFile = "Dockerfile"
	}
	if opt.BuildContext == "" {
		opt.BuildContext = "."
	}

	args := make([]string, 0)
	args = append(args, "buildx", "build", "--builder", "container", "--build-arg", "BUILDKIT_MULTI_PLATFORM=0", "--output", fmt.Sprintf("type=docker,dest=%s,oci-mediatypes=false,compression=uncompressed,name=%s", opt.ImageOutput, opt.Image), "-f", opt.DockerFile, "--platform=linux/amd64")
	args = append(args, detectProxySettings()...)
	args = append(args, opt.ExtraDockerArgs...)
	args = append(args, "-t", opt.Image, opt.BuildContext)

	proc := exec.Command("docker", args...)
	proc.Env = os.Environ()
	// proc.Env = append(proc.Env, "DOCKER_DEFAULT_PLATFORM=linux/amd64")
	proc.Env = append(proc.Env, "BUILDX_NO_DEFAULT_ATTESTATIONS=1")
	proc.Env = append(proc.Env, "BUILDX_METADATA_PROVENANCE=0")
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Run(); err != nil {
		return err
	}
	return nil
}

func BuildImageWithDockerInDocker(opt BuildImageOptions) error {
	if opt.DockerFile == "" {
		opt.DockerFile = "Dockerfile"
	}
	if opt.BuildContext == "" {
		opt.BuildContext = "."
	}

	args := make([]string, 0)
	args = append(args, "-s", "--")
	args = append(args, "--context", opt.BuildContext, "-f", opt.DockerFile, "-t", opt.Image, "--output", opt.ImageOutput)

	proc := exec.Command("sh", args...)
	proc.Stdin = strings.NewReader(BuildScript)
	proc.Env = os.Environ()
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Run(); err != nil {
		return err
	}
	return nil
}

func PrepareImage() error {
	proc := exec.Command("docker", "run", "--privileged", "--rm", "tonistiigi/binfmt", "--install", "all")
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	err := proc.Run()
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

// ExportImage the docker image
func ExportImage(image string, outputFile io.Writer) error {
	proc := exec.Command("docker", "save", image)
	proc.Stdout = outputFile
	proc.Stderr = os.Stderr
	proc.Env = os.Environ()
	proc.Env = append(proc.Env, "DOCKER_DEFAULT_PLATFORM=linux/amd64")
	proc.Env = append(proc.Env, "BUILDX_NO_DEFAULT_ATTESTATIONS=1")
	proc.Env = append(proc.Env, "BUILDX_METADATA_PROVENANCE=0")
	return proc.Run()
}

// Pack microservice using the manifest and image.tar file to create the archive which can
// be used to deploy to Cumulocity
func Pack(manifest io.Reader, imageTarball io.Reader, archive io.Writer) error {
	zipWriter := zip.NewWriter(archive)

	// manifest
	manifestWriter, err := zipWriter.Create(microservices.ManifestFile)
	if err != nil {
		return err
	}
	if _, err := io.Copy(manifestWriter, manifest); err != nil {
		return err
	}

	// contents
	imageWriter, err := zipWriter.Create("image.tar")
	if err != nil {
		return err
	}
	if _, err := io.Copy(imageWriter, imageTarball); err != nil {
		return err
	}

	if err := zipWriter.Close(); err != nil {
		return err
	}

	return nil
}

func detectProxySettings() []string {
	output := make([]string, 0)
	envVars := []string{
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"http_proxy",
		"https_proxy",
	}
	for _, name := range envVars {
		if value := os.Getenv(name); value != "" {
			envVars = append(envVars, "--build-arg", fmt.Sprintf("%s=%s", name, value))
		}
	}
	return output
}
