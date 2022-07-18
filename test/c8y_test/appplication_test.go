package c8y_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
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
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
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
	minApplications := 0
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(data.Items) >= minApplications, "Unexpected amount of applications found. want >=%d, got: %d", minApplications, len(data.Items))
	if minApplications > 0 {
		testingutils.Assert(t, data.Applications[0].Name != "", "Application should have a name")
	}
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
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Assert(t, len(data.Items) >= minApplications, "Unexpected amount of applications found. want >=%d, got: %d", minApplications, len(data.Items))
	testingutils.Assert(t, data.Applications[0].Name != "", "Application should have a name")
}

func TestApplicationService_GetApplication(t *testing.T) {
	client := createTestClient()

	applicationName := "cockpit"

	apps, _, err := client.Application.GetApplicationsByName(
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
	testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	testingutils.Equals(t, expApp.ID, app.ID)
}

func TestApplicationService_CRUD_Application(t *testing.T) {
	client := createTestClient()

	appName := "testApplication"

	appInfo := &c8y.Application{
		Key:         "testApplicationKey",
		Name:        "testApplication",
		Type:        "HOSTED",
		ContextPath: "/testApplication",
	}

	// Delete application if it already exists
	appCol, _, err := client.Application.GetApplicationsByName(
		context.Background(),
		appName,
		&c8y.ApplicationOptions{
			PaginationOptions: *c8y.NewPaginationOptions(10),
		},
	)
	testingutils.Ok(t, err)
	if len(appCol.Applications) > 0 {
		for _, app := range appCol.Applications {
			_, err := client.Application.Delete(context.Background(), app.ID)
			testingutils.Ok(t, err)
		}
	}

	// Delete the cloned app
	app2Col, _, err := client.Application.GetApplicationsByName(
		context.Background(),
		"clone"+appName,
		&c8y.ApplicationOptions{
			PaginationOptions: *c8y.NewPaginationOptions(10),
		},
	)
	testingutils.Ok(t, err)
	if len(app2Col.Applications) > 0 {
		for _, app := range app2Col.Applications {
			_, err := client.Application.Delete(context.Background(), app.ID)
			testingutils.Ok(t, err)
		}
	}

	//
	// Create
	app1, resp, err := client.Application.Create(
		context.Background(),
		appInfo,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, appInfo.Key, app1.Key)

	//
	// Update
	// app2, resp, err := client.Application.Update(
	// 	context.Background(),
	// 	app1.ID,
	// 	&c8y.Application{
	// 		Name: "UpdatedTestApplicationName",

	// 	},
	// )
	// testingutils.Ok(t, err)
	// testingutils.Equals(t, http.StatusOK, resp.StatusCode())
	// testingutils.Equals(t, "UpdatedTestApplicationName", app2.Name)

	// Copy existing application
	app2Copy, resp, err := client.Application.Copy(
		context.Background(),
		app1.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode())
	testingutils.Equals(t, "clone"+app1.Name, app2Copy.Name)

	//
	// Delete
	resp, err = client.Application.Delete(
		context.Background(),
		app1.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())

	//
	// Delete copied app
	resp, err = client.Application.Delete(
		context.Background(),
		app2Copy.ID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode())
}
