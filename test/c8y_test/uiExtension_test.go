package c8y_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

var uiExtensionURLApp1Version1 = "https://github.com/SoftwareAG/cumulocity-remote-access-cloud-http-proxy/releases/download/v2.4.3/cloud-http-proxy-ui.zip"
var uiExtensionURLApp1Version2 = "https://github.com/SoftwareAG/cumulocity-remote-access-cloud-http-proxy/releases/download/v2.5.0/cloud-http-proxy-ui.zip"

func TestUIExtensionService_CreateExtension(t *testing.T) {
	client := createTestClient()

	var err error

	appName := testingutils.RandomString(12)

	//
	// Version 1
	file1 := CreateTempFile(t, "exampleExtension1.zip")
	err = downloadFile(uiExtensionURLApp1Version1, file1)
	testingutils.Ok(t, err)

	app1, err := client.UIExtension.NewUIExtensionFromFile(file1.Name())
	testingutils.Assert(t, app1.Name != "", "Name should not be empty")
	testingutils.Assert(t, app1.Key != "", "Key should not be empty")

	// Use unique name to avoid name clashes
	app1.Name = appName
	app1.Key = appName + "-key"
	testingutils.Ok(t, err)

	appVersion1, _, err := client.UIExtension.CreateExtension(context.Background(), &app1.Application, file1.Name(), c8y.UpsertOptions{
		SkipActivation: false,
		Version: &c8y.ApplicationVersion{
			Version: app1.ManifestFile.Version,
			Tags:    []string{"tag1"},
		},
	})
	t.Cleanup(func() {
		client.Application.Delete(context.Background(), appVersion1.Application.ID)
	})
	testingutils.Ok(t, err)
	testingutils.Equals(t, "2.4.3", appVersion1.Version)
	testingutils.Assert(t, len(appVersion1.Tags) == 2, "Tags should be present")
	testingutils.ContainsString(t, "tag1", appVersion1.Tags)
	testingutils.ContainsString(t, "latest", appVersion1.Tags)

	//
	// Version 2
	file2 := CreateTempFile(t, "exampleExtension2.zip")
	err = downloadFile(uiExtensionURLApp1Version2, file2)
	testingutils.Ok(t, err)

	appVersion2, _, err := client.UIExtension.CreateExtension(context.Background(), appVersion1.Application, file2.Name(), c8y.UpsertOptions{
		SkipActivation: false,
		Version: &c8y.ApplicationVersion{
			Version: "2.5.0",
			Tags:    []string{"latest", "tag2"},
		},
	})
	testingutils.Ok(t, err)
	testingutils.Equals(t, "2.5.0", appVersion2.Version)
	testingutils.Assert(t, len(appVersion2.Tags) == 2, "Tags should be present")
	testingutils.ContainsString(t, "tag2", appVersion2.Tags)
	testingutils.ContainsString(t, "latest", appVersion2.Tags)
}
