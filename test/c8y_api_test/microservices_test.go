package c8y_api_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/microservices"
	"github.com/reubenmiller/go-c8y/pkg/tools/microservice_builder"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_MicroserviceUpload(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	tmpDir := t.TempDir()

	manifest, err := microservices.NewManifest(nil, microservices.FromFile(RelativePath("./data/microservices/python-example/cumulocity.json")))
	manifest.Name = "python-example"
	manifest.Version = "1.0.1-SNAPSHOT"
	manifest.APIVersion = "v2"
	assert.NoError(t, err)
	filename, err := microservice_builder.Build(microservice_builder.BuildOptions{
		DockerFile:   RelativePath("./data/microservices/python-example/docker/Dockerfile"),
		BuildContext: RelativePath("./data/microservices/python-example/docker"),
		Image:        manifest.Name,
		Manifest:     *manifest,
		OutputFile:   filepath.Join(tmpDir, manifest.Name+".zip"),
	})
	if !assert.NoError(t, err) {
		assert.FailNow(t, "aborting")
	}
	assert.NotEmpty(t, filename)

	app, err := client.Microservices.CreateOrUpdate(context.Background(), microservices.CreateOptions{
		File:       filename,
		TenantID:   client.Auth.Tenant,
		SkipUpload: false,
	})
	assert.NoError(t, err)
	assert.Equal(t, manifest.Name, app.Name)
}

func RelativePath(p string) string {
	_, filename, _, _ := runtime.Caller(0)
	wd := filepath.Dir(filename)
	out := filepath.Join(wd, p)
	return out
}
