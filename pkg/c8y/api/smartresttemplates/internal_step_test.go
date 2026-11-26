package smartresttemplates

import (
	"context"
	"testing"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateIdentityStepIgnoresConflictOption verifies the identity step of a
// create is not affected by WithConflictAsError: the result is the same as
// without the option.
func TestCreateIdentityStepIgnoresConflictOption(t *testing.T) {
	for name, ctx := range map[string]context.Context{
		"default": context.Background(),
		"strict":  ctxhelpers.WithConflictAsError(ctxhelpers.WithNotFoundAsError(context.Background())),
	} {
		t.Run(name, func(t *testing.T) {
			ts := newTestServer(nil)
			ts.identityConflict = true
			defer ts.server.Close()

			result := ts.service().Create(ctx, CreateOptions{
				Name:              "custom_devmgmt",
				ResponseTemplates: templates(responseTemplate("dm101", "name")),
			})
			require.NoError(t, result.Err)
			assert.Equal(t, op.StatusCreated, result.Status)
			assert.False(t, ts.deletedMO)
		})
	}
}
