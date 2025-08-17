package microservice

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type BuildOptions struct {
	Manifest     Manifest
	DockerFile   string
	BuildContext string
	OutputFile   string
	Image        string
	BuildArgs    []string
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

	if err := BuildImage(opts.DockerFile, opts.BuildContext, opts.Image, opts.BuildArgs...); err != nil {
		return outputFile, err
	}

	tarball, err := os.CreateTemp("", "image.tar")
	if err != nil {
		return outputFile, err
	}
	writer := bufio.NewWriter(tarball)
	if err := ExportImage(opts.Image, writer); err != nil {
		return "", err
	}
	defer tarball.Close()

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

// BuildImage microservice
func BuildImage(dockerFile string, buildContext string, image string, extraDockerArgs ...string) error {
	if dockerFile == "" {
		dockerFile = "Dockerfile"
	}
	if buildContext == "" {
		buildContext = "."
	}

	args := make([]string, 0)
	args = append(args, "buildx", "build", "--load", "-f", dockerFile, "--platform=linux/amd64")
	args = append(args, detectProxySettings()...)
	args = append(args, extraDockerArgs...)
	args = append(args, "-t", image, buildContext)

	proc := exec.Command("docker", args...)
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	return proc.Run()
}

func PrepareImage() error {
	proc := exec.Command("docker", "run", "--privileged", "--rm", "tonistiigi/binfmt", "--install", "all")
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	return proc.Run()

}

// ExportImage the docker image
func ExportImage(image string, outputFile io.Writer) error {
	proc := exec.Command("docker", "save", image)
	proc.Stdout = outputFile
	proc.Stderr = os.Stderr
	return proc.Run()
}

// Pack microservice using the manifest and image.tar file to create the archive which can
// be used to deploy to Cumulocity
func Pack(manifest io.Reader, imageTarball io.Reader, archive io.Writer) error {
	zipWriter := zip.NewWriter(archive)

	// manifest
	manifestWriter, err := zipWriter.Create(ManifestFile)
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
