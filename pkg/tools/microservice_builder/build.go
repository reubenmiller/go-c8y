// Package microservice_builder builds Cumulocity microservice archives, a zip
// file containing the cumulocity.json manifest and the docker image saved as
// image.tar, which can be uploaded directly to Cumulocity.
//
// Cumulocity requires the image tarball to be in the legacy docker archive
// format. Docker engines which use the containerd image store
// (io.containerd.snapshotter.v1) - the default in recent Docker Desktop,
// colima and other installations - produce OCI formatted tarballs from
// "docker save" which Cumulocity fails to load. When such an engine is
// detected, the image is transparently built inside a docker-in-docker
// container which uses the classic image store and therefore produces a
// compatible tarball.
//
// All docker interactions go through the docker CLI (no shell scripts), so
// builds work the same on Linux, macOS and Windows.
package microservice_builder

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
)

const (
	// DefaultPlatform is the image platform supported by Cumulocity
	DefaultPlatform = "linux/amd64"

	// DefaultDinDImage is the docker-in-docker image used for fallback builds.
	// docker:28 still uses the classic image store which produces image
	// tarballs that Cumulocity can load
	DefaultDinDImage = "docker:28-dind"

	// DefaultDinDContainerName is the name of the docker-in-docker container
	// which is kept running between builds to speed up subsequent builds
	DefaultDinDContainerName = "c8y-microservice-builder"

	// binfmtImage installs QEMU emulators to support cross-platform builds,
	// e.g. building linux/amd64 images on an arm64 host
	binfmtImage = "tonistiigi/binfmt"

	// containerdSnapshotterDriverType is the docker engine driver-type which
	// produces OCI image tarballs that Cumulocity cannot load
	containerdSnapshotterDriverType = "io.containerd.snapshotter.v1"
)

// BuildWith controls which build strategy is used
type BuildWith int

const (
	// BuildWithAuto inspects the docker engine and picks the native build if
	// the engine produces Cumulocity compatible image tarballs, otherwise it
	// falls back to docker-in-docker
	BuildWithAuto BuildWith = iota

	// BuildWithNative builds using the host's docker engine
	BuildWithNative

	// BuildWithDockerInDocker builds inside a docker-in-docker container
	BuildWithDockerInDocker
)

func (b BuildWith) String() string {
	switch b {
	case BuildWithNative:
		return "native"
	case BuildWithDockerInDocker:
		return "docker-in-docker"
	default:
		return "auto"
	}
}

// DockerInDockerOptions control the docker-in-docker container used for
// fallback builds
type DockerInDockerOptions struct {
	// ContainerName of the docker-in-docker container. Defaults to
	// DefaultDinDContainerName
	ContainerName string

	// Image used for the docker-in-docker container. Defaults to
	// DefaultDinDImage
	Image string

	// Remove the docker-in-docker container after the build. By default the
	// container is kept running so subsequent builds can reuse the docker
	// layer cache
	Remove bool
}

func (o *DockerInDockerOptions) applyDefaults() {
	if o.ContainerName == "" {
		o.ContainerName = DefaultDinDContainerName
	}
	if o.Image == "" {
		o.Image = DefaultDinDImage
	}
}

// BuildOptions control how the microservice archive is built
type BuildOptions struct {
	// Manifest of the microservice. Manifest.Name is required
	Manifest microservices.Manifest

	// DockerFile path. Defaults to "Dockerfile"
	DockerFile string

	// BuildContext directory. Defaults to "."
	BuildContext string

	// OutputFile path of the microservice zip archive. Defaults to
	// "<name>_<version>.zip" in the current directory
	OutputFile string

	// Image name to tag the built image with. Defaults to the manifest name
	Image string

	// BuildArgs are additional arguments passed to "docker buildx build"
	BuildArgs []string

	// Platform to build the image for. Defaults to DefaultPlatform
	Platform string

	// BuildWith selects the build strategy. Defaults to BuildWithAuto
	BuildWith BuildWith

	// DockerInDocker options used when building with docker-in-docker
	DockerInDocker DockerInDockerOptions

	// Stdout / Stderr receive the output of the docker commands. Default to
	// os.Stdout / os.Stderr
	Stdout io.Writer
	Stderr io.Writer
}

// Build the microservice archive and return the path to the created zip file
func Build(ctx context.Context, opts BuildOptions) (string, error) {
	if opts.Manifest.Name == "" {
		return "", errors.New("manifest name is required")
	}
	if opts.Manifest.Version == "" {
		opts.Manifest.Version = "0.0.1-SNAPSHOT"
	}
	if opts.Image == "" {
		opts.Image = opts.Manifest.Name
	}
	if opts.OutputFile == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		opts.OutputFile = filepath.Join(cwd, fmt.Sprintf("%s_%s.zip", opts.Manifest.Name, opts.Manifest.Version))
	}
	outputFile := opts.OutputFile

	manifest, err := json.Marshal(opts.Manifest)
	if err != nil {
		return "", err
	}

	docker, err := newDockerCLI(opts.Stdout, opts.Stderr)
	if err != nil {
		return "", err
	}

	engine, err := docker.engineInfo(ctx)
	if err != nil {
		return "", err
	}
	if engine.OSType != "" && !strings.EqualFold(engine.OSType, "linux") {
		return "", fmt.Errorf("docker engine is running %q containers, but Cumulocity microservices require linux images (on Windows, switch Docker Desktop to Linux containers)", engine.OSType)
	}

	buildWith := opts.BuildWith
	switch {
	case buildWith == BuildWithAuto:
		if engine.UsesContainerdImageStore() {
			slog.Info("Docker engine uses the containerd image store which produces image tarballs that Cumulocity cannot load. Falling back to docker-in-docker", "driver-type", containerdSnapshotterDriverType)
			buildWith = BuildWithDockerInDocker
		} else {
			buildWith = BuildWithNative
		}
	case buildWith == BuildWithNative && engine.UsesContainerdImageStore():
		slog.Warn("Docker engine uses the containerd image store; the exported image will most likely be rejected by Cumulocity. Use BuildWithAuto or BuildWithDockerInDocker instead", "driver-type", containerdSnapshotterDriverType)
	}

	tarball, err := os.CreateTemp("", "c8y-image-*.tar")
	if err != nil {
		return outputFile, err
	}
	tarball.Close()
	defer os.Remove(tarball.Name())

	imageOpts := BuildImageOptions{
		DockerFile:      opts.DockerFile,
		BuildContext:    opts.BuildContext,
		Image:           opts.Image,
		Platform:        opts.Platform,
		ExtraDockerArgs: opts.BuildArgs,
		ImageOutput:     tarball.Name(),
		DockerInDocker:  opts.DockerInDocker,
		Stdout:          opts.Stdout,
		Stderr:          opts.Stderr,
	}

	slog.Info("Building microservice image", "image", opts.Image, "strategy", buildWith.String())
	switch buildWith {
	case BuildWithDockerInDocker:
		err = docker.buildImageWithDinD(ctx, imageOpts)
	default:
		err = docker.buildImageNative(ctx, engine, imageOpts)
	}
	if err != nil {
		return outputFile, err
	}

	// Create archive
	if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
		return outputFile, err
	}
	archive, err := os.Create(outputFile)
	if err != nil {
		return outputFile, err
	}
	tarballIn, err := os.Open(tarball.Name())
	if err != nil {
		archive.Close()
		return outputFile, err
	}
	defer tarballIn.Close()
	if err := Pack(bytes.NewReader(manifest), tarballIn, archive); err != nil {
		archive.Close()
		return outputFile, err
	}
	return outputFile, archive.Close()
}

// BuildImageOptions control how the docker image is built and exported
type BuildImageOptions struct {
	// DockerFile path. Defaults to "Dockerfile"
	DockerFile string

	// BuildContext directory. Defaults to "."
	BuildContext string

	// Image name to tag the built image with
	Image string

	// Platform to build the image for. Defaults to DefaultPlatform
	Platform string

	// ExtraDockerArgs are additional arguments passed to "docker buildx build"
	ExtraDockerArgs []string

	// ImageOutput is the file path the image tarball is written to
	ImageOutput string

	// DockerInDocker options used when building with docker-in-docker
	DockerInDocker DockerInDockerOptions

	// Stdout / Stderr receive the output of the docker commands. Default to
	// os.Stdout / os.Stderr
	Stdout io.Writer
	Stderr io.Writer
}

func (o *BuildImageOptions) applyDefaults() {
	if o.DockerFile == "" {
		o.DockerFile = "Dockerfile"
	}
	if o.BuildContext == "" {
		o.BuildContext = "."
	}
	if o.Platform == "" {
		o.Platform = DefaultPlatform
	}
	o.DockerInDocker.applyDefaults()
}

// BuildImage builds the image with the host's docker engine and writes the
// image tarball to opt.ImageOutput. Note: the resulting tarball is only
// compatible with Cumulocity if the engine does not use the containerd image
// store (see InspectEngine)
func BuildImage(ctx context.Context, opt BuildImageOptions) error {
	docker, err := newDockerCLI(opt.Stdout, opt.Stderr)
	if err != nil {
		return err
	}
	engine, err := docker.engineInfo(ctx)
	if err != nil {
		return err
	}
	return docker.buildImageNative(ctx, engine, opt)
}

// BuildImageWithDockerInDocker builds the image inside a docker-in-docker
// container and writes the image tarball to opt.ImageOutput
func BuildImageWithDockerInDocker(ctx context.Context, opt BuildImageOptions) error {
	docker, err := newDockerCLI(opt.Stdout, opt.Stderr)
	if err != nil {
		return err
	}
	return docker.buildImageWithDinD(ctx, opt)
}

// RemoveBuilder removes the docker-in-docker builder container and its
// volumes. Use an empty containerName to remove the default builder
func RemoveBuilder(ctx context.Context, containerName string) error {
	if containerName == "" {
		containerName = DefaultDinDContainerName
	}
	docker, err := newDockerCLI(nil, nil)
	if err != nil {
		return err
	}
	return docker.run(ctx, "rm", "--force", "--volumes", containerName)
}

// EngineInfo is a subset of the information reported by "docker info"
type EngineInfo struct {
	Architecture string     `json:"Architecture,omitempty"`
	OSType       string     `json:"OSType,omitempty"`
	DriverStatus [][]string `json:"DriverStatus,omitempty"`
	ServerErrors []string   `json:"ServerErrors,omitempty"`
}

// UsesContainerdImageStore checks if the docker engine uses the containerd
// image store which produces OCI image tarballs that Cumulocity cannot load
func (e *EngineInfo) UsesContainerdImageStore() bool {
	for _, item := range e.DriverStatus {
		if len(item) >= 2 && strings.EqualFold(item[0], "driver-type") && strings.EqualFold(item[1], containerdSnapshotterDriverType) {
			return true
		}
	}
	return false
}

// InspectEngine queries the docker engine information used to decide which
// build strategy to use
func InspectEngine(ctx context.Context) (*EngineInfo, error) {
	docker, err := newDockerCLI(nil, nil)
	if err != nil {
		return nil, err
	}
	return docker.engineInfo(ctx)
}

// Pack microservice using the manifest and image tarball to create the
// archive which can be used to deploy to Cumulocity
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

	return zipWriter.Close()
}

//
// docker CLI plumbing
//

type dockerCLI struct {
	bin    string
	stdout io.Writer
	stderr io.Writer
}

func newDockerCLI(stdout, stderr io.Writer) (*dockerCLI, error) {
	bin, err := exec.LookPath("docker")
	if err != nil {
		return nil, fmt.Errorf("docker cli was not found in PATH. Install Docker Desktop, Docker Engine, colima or another docker compatible runtime: %w", err)
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	return &dockerCLI{bin: bin, stdout: stdout, stderr: stderr}, nil
}

func (d *dockerCLI) env() []string {
	return append(os.Environ(),
		"BUILDX_NO_DEFAULT_ATTESTATIONS=1",
		"BUILDX_METADATA_PROVENANCE=0",
	)
}

// run a docker command, streaming its output to the configured writers
func (d *dockerCLI) run(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, d.bin, args...)
	cmd.Env = d.env()
	cmd.Stdout = d.stdout
	cmd.Stderr = d.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker %s failed: %w", args[0], err)
	}
	return nil
}

// capture a docker command's stdout without streaming anything to the user
func (d *dockerCLI) capture(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, d.bin, args...)
	cmd.Env = d.env()
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return out, fmt.Errorf("docker %s failed: %w: %s", args[0], err, msg)
		}
		return out, fmt.Errorf("docker %s failed: %w", args[0], err)
	}
	return out, nil
}

// probe runs a docker command silently and only reports whether it succeeded
func (d *dockerCLI) probe(ctx context.Context, args ...string) bool {
	cmd := exec.CommandContext(ctx, d.bin, args...)
	cmd.Env = d.env()
	return cmd.Run() == nil
}

// engineInfo queries the docker engine. cmdPrefix targets the engine, e.g.
// none for the host or "exec", "<name>", "docker" for a docker-in-docker
// container
func (d *dockerCLI) engineInfo(ctx context.Context, cmdPrefix ...string) (*EngineInfo, error) {
	out, err := d.capture(ctx, append(cmdPrefix, "info", "--format", "{{json .}}")...)
	if err != nil {
		return nil, fmt.Errorf("could not query the docker engine. Is docker running?: %w", err)
	}
	info := &EngineInfo{}
	if err := json.Unmarshal(out, info); err != nil {
		return nil, fmt.Errorf("could not parse docker engine info: %w", err)
	}
	if len(info.ServerErrors) > 0 {
		return nil, fmt.Errorf("cannot connect to the docker engine: %s", strings.Join(info.ServerErrors, "; "))
	}
	return info, nil
}

func (d *dockerCLI) buildImageNative(ctx context.Context, engine *EngineInfo, opt BuildImageOptions) error {
	opt.applyDefaults()

	if normalizeArch(engine.Architecture) != platformArch(opt.Platform) {
		if err := d.installBinfmt(ctx, nil); err != nil {
			return err
		}
	}

	args := []string{"buildx", "build", "--load", "-f", opt.DockerFile, "--platform", opt.Platform}
	args = append(args, proxyBuildArgs()...)
	args = append(args, opt.ExtraDockerArgs...)
	args = append(args, "-t", opt.Image, opt.BuildContext)
	if err := d.run(ctx, args...); err != nil {
		return err
	}

	return d.run(ctx, "save", "-o", opt.ImageOutput, opt.Image)
}

func (d *dockerCLI) buildImageWithDinD(ctx context.Context, opt BuildImageOptions) error {
	opt.applyDefaults()
	name := opt.DockerInDocker.ContainerName

	if err := d.ensureDinD(ctx, name, opt.DockerInDocker.Image); err != nil {
		return err
	}

	// Guard against docker-in-docker images which default to the containerd
	// image store (the classic store is requested when creating the
	// container, but an existing container may predate that setting)
	inner, err := d.engineInfo(ctx, "exec", name, "docker")
	if err != nil {
		return fmt.Errorf("could not query the docker-in-docker engine: %w", err)
	}
	if inner.UsesContainerdImageStore() {
		return fmt.Errorf("the docker-in-docker container %q uses the containerd image store which produces image tarballs that Cumulocity cannot load. Remove the container (e.g. docker rm --force --volumes %s) and rebuild so it is recreated with the classic image store", name, name)
	}
	if opt.DockerInDocker.Remove {
		defer func() {
			// removal should also happen when the build was cancelled
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
			defer cancel()
			if err := d.run(cleanupCtx, "rm", "--force", "--volumes", name); err != nil {
				slog.Warn("Could not remove docker-in-docker container", "name", name, "err", err)
			}
		}()
	}

	// Container side paths always use forward slashes, independent of the host OS
	buildDir := path.Join("/build", sanitizeName(opt.Image))
	contextDir := path.Join(buildDir, "context")
	dockerFile := path.Join(buildDir, "Dockerfile")
	imageTar := path.Join(buildDir, "image.tar")

	// Copy the build inputs into the container as the inner docker engine
	// cannot see the host's file system
	if err := d.run(ctx, "exec", name, "rm", "-rf", buildDir); err != nil {
		return err
	}
	if err := d.run(ctx, "exec", name, "mkdir", "-p", buildDir); err != nil {
		return err
	}
	if err := d.run(ctx, "cp", opt.DockerFile, name+":"+dockerFile); err != nil {
		return fmt.Errorf("could not copy Dockerfile into the builder: %w", err)
	}
	if err := d.run(ctx, "cp", opt.BuildContext, name+":"+contextDir); err != nil {
		return fmt.Errorf("could not copy the build context into the builder: %w", err)
	}

	// Install emulators inside the builder when building foreign platforms,
	// e.g. linux/amd64 images on an arm64 host
	if normalizeArch(inner.Architecture) != platformArch(opt.Platform) {
		if err := d.installBinfmt(ctx, []string{"exec", name, "docker"}); err != nil {
			return err
		}
	}

	buildArgs := []string{
		"exec",
		"--env", "BUILDX_NO_DEFAULT_ATTESTATIONS=1",
		"--env", "BUILDX_METADATA_PROVENANCE=0",
		name,
		"docker", "buildx", "build", "--load", "--platform", opt.Platform, "-f", dockerFile,
	}
	buildArgs = append(buildArgs, proxyBuildArgs()...)
	buildArgs = append(buildArgs, opt.ExtraDockerArgs...)
	buildArgs = append(buildArgs, "-t", opt.Image, contextDir)
	if err := d.run(ctx, buildArgs...); err != nil {
		return err
	}

	if err := d.run(ctx, "exec", name, "docker", "save", "-o", imageTar, opt.Image); err != nil {
		return err
	}
	if err := d.run(ctx, "cp", name+":"+imageTar, opt.ImageOutput); err != nil {
		return fmt.Errorf("could not copy the image tarball out of the builder: %w", err)
	}

	// Best effort cleanup of the build inputs, the layer cache stays warm
	if err := d.run(ctx, "exec", name, "rm", "-rf", buildDir); err != nil {
		slog.Warn("Could not clean up the builder's build directory", "dir", buildDir, "err", err)
	}
	return nil
}

// ensureDinD makes sure the docker-in-docker container exists, is running and
// its inner docker engine accepts connections
func (d *dockerCLI) ensureDinD(ctx context.Context, name, image string) error {
	state, err := d.capture(ctx, "container", "inspect", "--format", "{{.State.Running}}", name)
	switch {
	case err != nil:
		// container does not exist yet
		slog.Info("Starting docker-in-docker builder", "name", name, "image", image)
		args := []string{"run", "--detach", "--privileged", "--name", name}
		for _, e := range proxyEnv() {
			args = append(args, "--env", e)
		}
		// Arguments after the image are passed to the inner dockerd. Pin the
		// classic image store so the engine produces Cumulocity compatible
		// image tarballs even when the dind image defaults to containerd
		args = append(args, image, "--feature", "containerd-snapshotter=false")
		if err := d.run(ctx, args...); err != nil {
			return fmt.Errorf("could not start the docker-in-docker container: %w", err)
		}
	case strings.TrimSpace(string(state)) != "true":
		slog.Info("Starting existing docker-in-docker builder", "name", name)
		if err := d.run(ctx, "start", name); err != nil {
			return fmt.Errorf("could not start the docker-in-docker container: %w", err)
		}
	}

	// Wait for the inner docker engine to accept connections
	deadline := time.Now().Add(60 * time.Second)
	for {
		if d.probe(ctx, "exec", name, "docker", "info") {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("docker-in-docker container %q did not become ready in time. Check its logs with: docker logs %s", name, name)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

// installBinfmt installs QEMU emulators for cross-platform builds. cmdPrefix
// targets the engine, e.g. nil for the host or ["exec", name, "docker"] for
// docker-in-docker
func (d *dockerCLI) installBinfmt(ctx context.Context, cmdPrefix []string) error {
	slog.Info("Installing QEMU emulators for cross-platform builds", "image", binfmtImage)
	args := append(append([]string{}, cmdPrefix...), "run", "--privileged", "--rm", binfmtImage, "--install", "all")
	if err := d.run(ctx, args...); err != nil {
		return fmt.Errorf("could not install QEMU emulators (required to build foreign platform images): %w", err)
	}
	return nil
}

//
// helpers
//

var proxyEnvNames = []string{
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"NO_PROXY",
	"http_proxy",
	"https_proxy",
	"no_proxy",
}

// proxyEnv returns the proxy related environment variables which are set,
// e.g. ["HTTP_PROXY=http://proxy:3128"]
func proxyEnv() []string {
	out := make([]string, 0, len(proxyEnvNames))
	for _, name := range proxyEnvNames {
		if value := os.Getenv(name); value != "" {
			out = append(out, name+"="+value)
		}
	}
	return out
}

// proxyBuildArgs converts the proxy environment variables to docker build arguments
func proxyBuildArgs() []string {
	out := make([]string, 0)
	for _, e := range proxyEnv() {
		out = append(out, "--build-arg", e)
	}
	return out
}

// normalizeArch maps the different spellings of an architecture (GOARCH,
// uname -m, docker info) to a single value
func normalizeArch(arch string) string {
	switch strings.ToLower(strings.TrimSpace(arch)) {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	case "armv7l", "armhf", "arm":
		return "arm"
	default:
		return strings.ToLower(strings.TrimSpace(arch))
	}
}

// platformArch extracts the architecture from a platform string, e.g. "linux/amd64" => "amd64"
func platformArch(platform string) string {
	if _, arch, found := strings.Cut(platform, "/"); found {
		// strip any variant, e.g. linux/arm/v7
		arch, _, _ = strings.Cut(arch, "/")
		return normalizeArch(arch)
	}
	return normalizeArch(platform)
}

var unsafePathChars = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// sanitizeName converts an image name to a value which is safe to use as a
// directory name, e.g. "myorg/app:1.0" => "myorg-app-1.0"
func sanitizeName(name string) string {
	out := unsafePathChars.ReplaceAllString(name, "-")
	out = strings.Trim(out, "-.")
	if out == "" {
		out = "image"
	}
	return out
}
