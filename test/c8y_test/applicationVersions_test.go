package c8y_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

var uiExampleExtension = "https://github.com/SoftwareAG/cumulocity-remote-access-cloud-http-proxy/releases/download/v2.5.0/cloud-http-proxy-ui.zip"

func createTestExtension(t *testing.T, client *c8y.Client, version *c8y.ApplicationVersion) *c8y.Application {
	appName := testingutils.RandomString(12)
	appOptions := c8y.NewApplicationExtension(appName)
	app, _, err := client.Application.Create(context.Background(), &appOptions.Application)
	testingutils.Ok(t, err)

	_, _, err = client.ApplicationVersions.CreateVersion(context.Background(), app.ID, uiExampleExtension, *version)
	testingutils.Ok(t, err)

	t.Cleanup(func() {
		client.Application.Delete(context.Background(), app.ID)
	})

	return app
}

func downloadFile(u string, out io.WriteCloser) error {
	defer out.Close()
	resp, err := http.Get(u)
	if err != nil {
		return fmt.Errorf("failed to download extension from url. %w", err)
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// CreateTempFile creates a file which will be cleaned up at the end of the test
func CreateTempFile(t *testing.T, name string) *os.File {
	file, err := os.CreateTemp("", "*_"+name)
	testingutils.Ok(t, err)
	t.Cleanup(func() {
		file.Close()
		os.Remove(file.Name())
	})
	return file
}

func TestApplicationVersionsService_GetVersions(t *testing.T) {
	client := createTestClient()
	app := createTestExtension(t, client, &c8y.ApplicationVersion{
		Version: "1.0.0",
		Tags:    []string{"latest"},
	})
	apps, resp, err := client.ApplicationVersions.GetVersions(
		context.Background(),
		app.ID,
		&c8y.ApplicationVersionsOptions{
			PaginationOptions: c8y.PaginationOptions{
				PageSize: 10,
			},
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(apps.Versions) > 0, "At least one application version should be found")
	testingutils.Assert(t, apps.Versions[0].Version == "1.0.0", "Version should be set")
	testingutils.Assert(t, len(apps.Versions[0].Tags) == 1, "Tags should be present")
	testingutils.Assert(t, apps.Versions[0].Tags[0] == "latest", "Tag should be set")
}

func TestApplicationVersionsService_GetVersionByTag(t *testing.T) {
	client := createTestClient()
	app := createTestExtension(t, client, &c8y.ApplicationVersion{
		Version: "1.0.1",
		Tags:    []string{"latest", "tag1"},
	})

	data, resp, err := client.ApplicationVersions.GetVersionByTag(
		context.Background(),
		app.ID,
		"tag1",
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, data.Version, "1.0.1")
	testingutils.Equals(t, 2, len(data.Tags))
	testingutils.ContainsString(t, "latest", data.Tags)
	testingutils.ContainsString(t, "tag1", data.Tags)
}

func TestApplicationVersionsService_GetVersionByName(t *testing.T) {
	client := createTestClient()
	app := createTestExtension(t, client, &c8y.ApplicationVersion{
		Version: "1.0.2",
		Tags:    []string{"tag1"},
	})

	data, resp, err := client.ApplicationVersions.GetVersionByName(
		context.Background(),
		app.ID,
		"1.0.2",
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, data.Version, "1.0.2")
	testingutils.Equals(t, 2, len(data.Tags))
	testingutils.ContainsString(t, "latest", data.Tags) // Latest is automatically added when it is activated
	testingutils.ContainsString(t, "tag1", data.Tags)
}

func TestApplicationVersionsService_CRUD_Extension(t *testing.T) {
	client := createTestClient()

	file := CreateTempFile(t, "exampleExtension.zip")

	err := downloadFile(uiExampleExtension, file)
	testingutils.Ok(t, err)

	app, err := client.UIExtension.NewUIExtensionFromFile(file.Name())
	testingutils.Ok(t, err)

	// Use a unique name
	app.Name = testingutils.RandomString(12)
	app.Key = app.Name + "-key"

	//
	// Create
	appVersion, resp, err := client.UIExtension.CreateExtension(context.Background(), &app.Application, file.Name(), c8y.UpsertOptions{
		SkipActivation: false,
		Version: &c8y.ApplicationVersion{
			Version: app.ManifestFile.Version,
			Tags:    []string{"latest", "tag1"},
		},
	})

	t.Cleanup(func() {
		// Don't check if it failed or not, as the test will also delete the application
		// but this cleanup is done just in case the test fails earlier
		client.Application.Delete(context.Background(), appVersion.Application.ID)
	})

	testingutils.Ok(t, err)
	testingutils.Assert(t, http.StatusCreated == resp.StatusCode() || http.StatusOK == resp.StatusCode(), "Status code is ok")
	testingutils.Equals(t, appVersion.Version, app.ManifestFile.Version)
	testingutils.Equals(t, 2, len(appVersion.Tags))
	testingutils.ContainsString(t, "latest", appVersion.Tags)
	testingutils.ContainsString(t, "tag1", appVersion.Tags)

	// Create second version
	appVersion2, resp, err := client.UIExtension.CreateExtension(context.Background(), &app.Application, file.Name(), c8y.UpsertOptions{
		SkipActivation: false,
		Version: &c8y.ApplicationVersion{
			Version: "2.5.1",
			Tags:    []string{"latest", "tagA"},
		},
	})
	testingutils.Ok(t, err)
	testingutils.Assert(t, http.StatusCreated == resp.StatusCode() || http.StatusOK == resp.StatusCode(), "Status code is ok")
	testingutils.Equals(t, appVersion2.Version, "2.5.1")
	testingutils.Equals(t, 2, len(appVersion2.Tags))
	testingutils.ContainsString(t, "latest", appVersion2.Tags)
	testingutils.ContainsString(t, "taga", appVersion2.Tags) // Tags are lowercased by the platform

	// Update?
	updatedVersion, resp, err := client.ApplicationVersions.ReplaceTags(context.Background(), appVersion.Application.ID, app.ManifestFile.Version, []string{"tag2", "tag3"})
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, 2, len(updatedVersion.Tags))
	testingutils.ContainsString(t, "tag2", updatedVersion.Tags)
	testingutils.ContainsString(t, "tag3", updatedVersion.Tags)

	_, _, err = client.UIExtension.SetActive(context.Background(), appVersion.Application.ID, "")
	testingutils.Ok(t, err)

	//
	// Delete by version (the non active version)
	resp, err = client.ApplicationVersions.DeleteVersionByName(
		context.Background(),
		appVersion.Application.ID,
		appVersion.Version,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	//
	// Delete by tag
	// TODO
}
