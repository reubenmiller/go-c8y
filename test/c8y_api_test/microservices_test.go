package api_test

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/tools/microservice_builder"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_MicroserviceListAll(t *testing.T) {
	client := testcore.CreateTestClient(t)
	items := client.Microservices.ListAll(context.Background(), microservices.ListOptions{})
	assert.NoError(t, items.Err())
	total := 0
	for item := range items.Items() {
		fmt.Printf("id=%s, type=%s, owner=%s\n", item.ID(), item.Type(), item.Owner())
		total += 1
	}
	assert.Greater(t, total, 0)
}

func Test_MicroserviceListAll_WithLimit(t *testing.T) {
	client := testcore.CreateTestClient(t)
	items := client.Microservices.ListAll(context.Background(), microservices.ListOptions{
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 10,
			PageSize: 10,
		},
	})
	assert.NoError(t, items.Err())
	total := 0
	for item := range items.Items() {
		fmt.Printf("id=%s, type=%s, owner=%s\n", item.ID(), item.Type(), item.Owner())
		total += 1
	}
	assert.LessOrEqual(t, 10, total)
}

func Test_MicroserviceGetByName(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Microservices.Get(
		context.Background(),
		client.Microservices.ByName("reporting"),
	)
	assert.NoError(t, result.Err)
	assert.Equal(t, "reporting", result.Data.Name())
}

func Test_MicroserviceGetByContext(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Microservices.Get(
		context.Background(),
		client.Microservices.ByContextPath("reporting"),
	)
	assert.NoError(t, result.Err)
	assert.Equal(t, "reporting", result.Data.ContextPath())
}

func Test_MicroserviceUpload(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.SetDebug(true)

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
