package c8y_test

import (
	"context"
	"net/http"
	"testing"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/reubenmiller/go-c8y/c8y_test/testingutils"
)

func TestIdentityService_Create(t *testing.T) {
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

func TestIdentityService_GetExternalID(t *testing.T) {
	client := createTestClient()

	identity, resp, err := client.Identity.GetExternalID(
		context.Background(),
		"NoExistantType",
		"Value123",
	)

	testingutils.Assert(t, err != nil, "Error should not be nil")
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode)
	testingutils.Equals(t, "", identity.Type)
}

func TestIdentityService_Delete(t *testing.T) {
	client := createTestClient()
	testDevice, err := createRandomTestDevice()
	testingutils.Ok(t, err)

	identityType := "testType"
	externalID := "MyUniqueValue1"

	//
	// Create identity
	//
	identity, resp, err := client.Identity.Create(
		context.Background(),
		testDevice.ID,
		&c8y.IdentityOptions{
			Type:       identityType,
			ExternalID: externalID,
		},
	)

	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusCreated, resp.StatusCode)
	testingutils.Equals(t, identityType, identity.Type)
	testingutils.Equals(t, externalID, identity.ExternalID)

	//
	// Remove identity
	//
	resp, err = client.Identity.Delete(
		context.Background(),
		identityType,
		externalID,
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, http.StatusNoContent, resp.StatusCode)

	//
	// Check that is is really was deleted
	//
	identity2, resp, err := client.Identity.GetExternalID(
		context.Background(),
		identityType,
		externalID,
	)

	testingutils.Assert(t, err != nil, "Error should not be nil")
	testingutils.Equals(t, http.StatusNotFound, resp.StatusCode)
	testingutils.Equals(t, "", identity2.Type)
}
