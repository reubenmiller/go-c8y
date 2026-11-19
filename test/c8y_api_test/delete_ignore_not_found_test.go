package api_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

// Calling IgnoreNotFound again on an already skipped delete keeps it skipped.
func TestManagedObject_Delete_IgnoreNotFound_Twice(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.ManagedObjects.Delete(context.Background(), "99999999", managedobjects.DeleteOptions{}).IgnoreNotFound()

	assert.NoError(t, result.Err)
	assert.Equal(t, op.StatusSkipped, result.Status)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
}

// Only 404 is ignored: other errors from a delete are still returned.
func TestManagedObject_Delete_KeepsOtherErrors(t *testing.T) {
	client := testcore.CreateTestClientNoAuth(t)
	result := client.ManagedObjects.Delete(context.Background(), "99999999", managedobjects.DeleteOptions{})

	assert.Error(t, result.Err)
	assert.Equal(t, http.StatusUnauthorized, result.HTTPStatus)
	assert.Equal(t, op.StatusFailed, result.Status)
}

// The not-found behavior applies to the other services' deletes too.
func TestEvent_Delete_NotFound(t *testing.T) {
	client := testcore.CreateTestClient(t)
	result := client.Events.Delete(context.Background(), "99999999")

	assert.NoError(t, result.Err)
	assert.Equal(t, op.StatusSkipped, result.Status)
	assert.Equal(t, http.StatusNotFound, result.HTTPStatus)
}
