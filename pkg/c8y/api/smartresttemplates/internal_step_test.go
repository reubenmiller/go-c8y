package smartresttemplates

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateNameTakenReturnsExistingCollection verifies that when the name is
// already registered the new managed object is removed and the existing
// collection is returned as a duplicate, not the new orphan.
func TestCreateNameTakenReturnsExistingCollection(t *testing.T) {
	existing := existingCollection("custom_devmgmt", nil, templates(responseTemplate("dm101", "name")))
	ts := newTestServer(existing)
	ts.identityConflict = true
	defer ts.server.Close()

	result := ts.service().Create(context.Background(), CreateOptions{
		Name:              "custom_devmgmt",
		ResponseTemplates: templates(responseTemplate("dm101", "name")),
	})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusDuplicate, result.Status)
	assert.Equal(t, "123", result.Data.ID(), "the existing collection is returned")
	assert.True(t, ts.deletedMO, "the new managed object is removed")
}

// TestCreateNameTakenWithConflictAsError verifies the caller keeps the 409
// error with WithConflictAsError, and the new managed object is still removed.
func TestCreateNameTakenWithConflictAsError(t *testing.T) {
	existing := existingCollection("custom_devmgmt", nil, templates(responseTemplate("dm101", "name")))
	ts := newTestServer(existing)
	ts.identityConflict = true
	defer ts.server.Close()

	ctx := ctxhelpers.WithConflictAsError(context.Background())
	result := ts.service().Create(ctx, CreateOptions{
		Name:              "custom_devmgmt",
		ResponseTemplates: templates(responseTemplate("dm101", "name")),
	})
	require.Error(t, result.Err)
	assert.True(t, core.ErrHasStatus(result.Err, 409))
	assert.True(t, ts.deletedMO)
}
