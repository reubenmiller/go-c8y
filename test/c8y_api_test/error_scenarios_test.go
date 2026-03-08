package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

// --- 404 Not Found scenarios ---

func TestAlarm_Get_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Alarms.Get(context.Background(), "99999999")
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
	assert.True(t, core.IsNotFound(result.Err))
}

func TestEvent_Get_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Events.Get(context.Background(), "99999999")
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
	assert.True(t, core.IsNotFound(result.Err))
}

func TestManagedObject_Get_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.ManagedObjects.Get(context.Background(), "99999999", managedobjects.GetOptions{})
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
	assert.True(t, core.IsNotFound(result.Err))
}

func TestOperation_Get_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Operations.Get(context.Background(), "99999999")
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
	assert.True(t, core.IsNotFound(result.Err))
}

func TestAlarm_Update_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Alarms.Update(context.Background(), "99999999", map[string]any{
		"status": "ACKNOWLEDGED",
	})
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
}

func TestManagedObject_Update_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.ManagedObjects.Update(context.Background(), "99999999", map[string]any{
		"name": "does-not-exist",
	})
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
}

func TestManagedObject_Delete_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.ManagedObjects.Delete(context.Background(), "99999999", managedobjects.DeleteOptions{})
	assert.Error(t, result.Err)
	assert.True(t, core.IsNotFound(result.Err))
}

// --- 401 Unauthorized scenarios ---

func TestAlarm_List_Unauthorized(t *testing.T) {
	client := testcore.CreateTestClientNoAuth(t)
	result := client.Alarms.List(context.Background(), alarms.ListOptions{})
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusUnauthorized, result.HTTPStatus)
}

func TestManagedObject_List_Unauthorized(t *testing.T) {
	client := testcore.CreateTestClientNoAuth(t)
	result := client.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusUnauthorized, result.HTTPStatus)
}

// --- Empty collection scenarios ---

func TestAlarm_List_EmptyWithFilter(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Alarms.List(context.Background(), alarms.ListOptions{
		Type: []string{"nonexistent_type_xyz"},
	})
	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
	assert.Equal(t, 0, result.Data.Length())
}

func TestEvent_List_EmptyWithFilter(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Events.List(context.Background(), events.ListOptions{
		Type: "nonexistent_type_xyz",
	})
	assert.NoError(t, result.Err)
	assert.Equal(t, http.StatusOK, result.HTTPStatus)
	assert.Equal(t, 0, result.Data.Length())
}

// --- Create then delete lifecycle ---

func TestManagedObject_CreateAndDelete(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Create
	createResult := client.ManagedObjects.Create(ctx, map[string]any{
		"name": "ephemeral-object",
		"type": "test_lifecycle",
	})
	assert.NoError(t, createResult.Err)
	assert.Equal(t, http.StatusCreated, createResult.HTTPStatus)
	id := createResult.Data.ID()
	assert.NotEmpty(t, id)

	// Verify it exists
	getResult := client.ManagedObjects.Get(ctx, id, managedobjects.GetOptions{})
	assert.NoError(t, getResult.Err)
	assert.Equal(t, http.StatusOK, getResult.HTTPStatus)

	// Delete
	delResult := client.ManagedObjects.Delete(ctx, id, managedobjects.DeleteOptions{})
	assert.NoError(t, delResult.Err)
	assert.Equal(t, http.StatusNoContent, delResult.HTTPStatus)

	// Verify it's gone
	getResult2 := client.ManagedObjects.Get(ctx, id, managedobjects.GetOptions{})
	assert.Error(t, getResult2.Err)
	assert.Equal(t, http.StatusNotFound, getResult2.HTTPStatus)
}

// --- IsNotFound helper ---

func TestIsNotFound_TrueForMissing(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Alarms.Get(context.Background(), "00000000")
	assert.True(t, core.IsNotFound(result.Err), "IsNotFound should be true for missing resource")
}

func TestIsNotFound_FalseForSuccess(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Alarms.List(context.Background(), alarms.ListOptions{})
	assert.False(t, core.IsNotFound(result.Err), "IsNotFound should be false for successful request")
}
