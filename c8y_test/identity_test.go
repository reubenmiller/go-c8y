package c8y_test

import (
	"context"
	"net/http"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/c8y_test/testingutils"
)

func TestIdentityService_NewExternalIdentity(t *testing.T) {
	client := createTestClient()

	testDevice, _, err := client.Inventory.CreateDevice(context.Background(), "testDevice")
	testingutils.Ok(t, err)
	defer client.Inventory.Delete(context.Background(), testDevice.ID)

	identity, resp, err := client.Identity.Create(context.Background(), testDevice.ID, &c8y.IdentityOptions{
		Type:       "test_Type",
		ExternalID: "testDevice1",
	})

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Equals(t, testDevice.ID, identity.ManagedObject.ID)
	testingutils.Equals(t, "test_Type", identity.Type)
	testingutils.Equals(t, "testDevice1", identity.ExternalID)

	// Get identity object
	identity, resp, err = client.Identity.GetExternalID(context.Background(), "test_Type", "testDevice1")
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusOK, resp.StatusCode)
	testingutils.Equals(t, testDevice.ID, identity.ManagedObject.ID)

	moID := identity.Item.Get("managedObject.id").String()
	testingutils.Equals(t, testDevice.ID, moID)
}
