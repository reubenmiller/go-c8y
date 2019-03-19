package c8y_test

import (
	"context"
	"net/http"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/c8y_test/testingutils"
)

func TestApplicationService_GetApplications(t *testing.T) {
	client := createTestClient()
	apps, resp, err := client.Application.GetApplications(
		context.Background(),
		&c8y.ApplicationOptions{
			PaginationOptions: c8y.PaginationOptions{
				PageSize: 10,
			},
		},
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(apps.Applications) > 0, "At least one application should be found")
	testingutils.Assert(t, apps.Applications[0].Name != "", "Application should have a name")
}

func TestApplicationService_GetCurrentApplicationSubscriptions(t *testing.T) {
	t.Skip("TODO: Requires a new application and application credentials")
}

func TestApplicationService_GetApplicationsByName(t *testing.T) {
	client := createTestClient()

	exampleAppName := "cockpit"

	data, _, err := client.Application.GetApplicationsByName(context.Background(), exampleAppName, nil)

	if err != nil {
		t.Errorf("Unexpected error. want: nil, got: %s", err)
	}
	minApplications := 1
	if len(data.Items) < minApplications {
		t.Errorf("Unexpected amount of applications found. want >=%d, got: %d", minApplications, len(data.Items))
	}

	actualAppName := data.Items[0].Get("name").String()
	if actualAppName != exampleAppName {
		t.Errorf("Wrong application name. want: %s, got: %s", exampleAppName, actualAppName)
	}
}

func TestApplicationService_GetApplicationsByOwner(t *testing.T) {
	client := createTestClient()

	data, resp, err := client.Application.GetApplicationsByOwner(
		context.Background(),
		client.TenantName,
		nil,
	)
	minApplications := 1
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(data.Items) >= minApplications, "Unexpected amount of applications found. want >=%d, got: %d", minApplications, len(data.Items))
	testingutils.Assert(t, data.Applications[0].Name != "", "Application should have a name")
}

func TestApplicationService_GetApplicationsByTenant(t *testing.T) {
	client := createTestClient()

	data, resp, err := client.Application.GetApplicationsByTenant(
		context.Background(),
		client.TenantName,
		nil,
	)
	minApplications := 1
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Assert(t, len(data.Items) >= minApplications, "Unexpected amount of applications found. want >=%d, got: %d", minApplications, len(data.Items))
	testingutils.Assert(t, data.Applications[0].Name != "", "Application should have a name")
}

func TestApplicationService_GetApplication(t *testing.T) {
	client := createTestClient()

	applicationName := "cockpit"

	apps, resp, err := client.Application.GetApplicationsByName(
		context.Background(),
		applicationName,
		nil,
	)
	testingutils.Ok(t, err)
	testingutils.Assert(t, len(apps.Applications) > 0, "Should return at least 1 application")

	expApp := apps.Applications[0]

	app, resp, err := client.Application.GetApplication(
		context.Background(),
		apps.Applications[0].ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, expApp.ID, app.ID)
}
