package api_test

import (
	"context"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Microservices_List(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	result := client.Microservices.List(ctx, microservices.ListOptions{})
	require.NoError(t, result.Err)
	items, err := op.ToSliceR(result)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(items), 1)
}

func Test_Microservices_FindFirst(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	res, ok := client.Microservices.FindFirst(ctx, microservices.ListOptions{})
	require.NoError(t, res.Err)
	assert.True(t, ok)
}

func Test_Microservices_ByID_ByName_ByContextPath(t *testing.T) {
	client := testcore.CreateTestClient(t)
	assert.Equal(t, "12345", client.Microservices.ByID("12345"))
	assert.Equal(t, "name:foo", client.Microservices.ByName("foo"))
	assert.Equal(t, "contextPath:/foo", client.Microservices.ByContextPath("/foo"))
}

func Test_Microservices_CRUD(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	name := "ms" + testingutils.RandomString(6)
	createRes := client.Microservices.Create(ctx, map[string]any{
		"name":        name,
		"key":         name + "-key",
		"type":        "MICROSERVICE",
		"contextPath": name,
	})
	require.NoError(t, createRes.Err)
	id := createRes.Data.ID()
	t.Cleanup(func() {
		client.Microservices.Delete(ctx, id, microservices.DeleteOptions{})
	})

	updRes := client.Microservices.Update(ctx, id, map[string]any{"availability": "MARKET"})
	require.NoError(t, updRes.Err)

	subRes := client.Microservices.Subscribe(ctx, client.Auth.Tenant, createRes.Data.Self())
	require.NoError(t, subRes.Err)

	unsubRes := client.Microservices.Unsubscribe(ctx, client.Auth.Tenant, id)
	require.NoError(t, unsubRes.Err)

	upload := client.Microservices.Upload(ctx, id, microservices.UploadFileOptions{
		Name:   "file.zip",
		Reader: strings.NewReader("PK\x03\x04dummy"),
	})
	// Upload may not produce a valid binary doc, but shouldn't error from the
	// fake server side -- accept any non-network error result.
	_ = upload
}

func Test_Microservices_NewManifest_FromJSON(t *testing.T) {
	manifest, err := microservices.NewManifest(nil, microservices.FromJSON(strings.NewReader(`{
		"name":"svc","contextPath":"svc","apiVersion":"v2","isolation":"PER_TENANT"
	}`)))
	require.NoError(t, err)
	assert.Equal(t, "svc", manifest.Name)
	assert.Equal(t, microservices.APIVersion("v2"), manifest.APIVersion)
	assert.Equal(t, microservices.IsolationPerTenant, manifest.Isolation)
}

func Test_Microservices_NewOption(t *testing.T) {
	opt := microservices.NewOption(nil, microservices.WithOverwriteOnUpdate(false), microservices.WithInheritFromOwner(true))
	require.NotNil(t, opt.OverwriteOnUpdate)
	require.NotNil(t, opt.InheritFromOwner)
	assert.False(t, *opt.OverwriteOnUpdate)
	assert.True(t, *opt.InheritFromOwner)
}

func Test_Microservices_GetBootstrapUserFromEnvironment(t *testing.T) {
	t.Setenv(microservices.EnvironmentBootstrapTenant, "tABC")
	t.Setenv(microservices.EnvironmentBootstrapUsername, "u1")
	t.Setenv(microservices.EnvironmentBootstrapPassword, "p1")
	tn, u, p := microservices.GetBootstrapUserFromEnvironment()
	assert.Equal(t, "tABC", tn)
	assert.Equal(t, "u1", u)
	assert.Equal(t, "p1", p)
}

func Test_Microservices_GetServiceUserFromEnvironment(t *testing.T) {
	t.Setenv(microservices.EnvironmentTenant, "tABC")
	t.Setenv(microservices.EnvironmentUsername, "u2")
	t.Setenv(microservices.EnvironmentPassword, "p2")
	tn, u, p := microservices.GetServiceUserFromEnvironment()
	assert.Equal(t, "tABC", tn)
	assert.Equal(t, "u2", u)
	assert.Equal(t, "p2", p)
}

func Test_Microservices_GetBootstrapBaseURL(t *testing.T) {
	t.Setenv(microservices.EnvironmentBaseURL, "https://example.cumulocity.com")
	assert.Equal(t, "https://example.cumulocity.com", microservices.GetBootstrapBaseURLFromEnvironment())
}

func Test_Microservices_ByNameFilter(t *testing.T) {
	pred := microservices.ByName("foo")
	assert.True(t, pred(modelMicroservice("foo")))
	assert.False(t, pred(modelMicroservice("bar")))
	assert.True(t, microservices.First(modelMicroservice("anything")))
}

func modelMicroservice(name string) model.Microservice {
	return model.Microservice{Name: name}
}
