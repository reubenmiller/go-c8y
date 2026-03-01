package api_test

import (
	"context"
	"testing"

	c8yapi "github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates/certificateauthority"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CertificateAuthority_Get(t *testing.T) {
	t.Skip("Skip due to affects other tests due to performing destructive operations on the CA certificate")
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Delete (setup) - use the new composed Delete method
	// Ignore error as certificate may not exist
	client.TrustedCertificates.CertificateAuthority.Delete(ctx, certificateauthority.DeleteOptions{
		TenantID: client.Auth.Tenant,
	})

	// Create
	createResult := client.TrustedCertificates.CertificateAuthority.Create(ctx, certificateauthority.CreateOptions{})
	require.NoError(t, createResult.Err)
	assert.NotEmpty(t, createResult.Data.Fingerprint(), "fingerprint should not be empty")

	// Create again (should be idempotent - returns 409 Conflict)
	createDuplicateResult := client.TrustedCertificates.CertificateAuthority.Create(ctx, certificateauthority.CreateOptions{})
	// Create returns conflict but GetOrCreate handles it
	if createDuplicateResult.HTTPStatus == 409 {
		getResult := client.TrustedCertificates.CertificateAuthority.Get(ctx, certificateauthority.GetOptions{
			TenantID: client.Auth.Tenant,
		})
		require.NoError(t, getResult.Err)
		assert.Equal(t, createResult.Data.Fingerprint(), getResult.Data.Fingerprint())
	} else {
		require.NoError(t, createDuplicateResult.Err)
		assert.Equal(t, createResult.Data.Fingerprint(), createDuplicateResult.Data.Fingerprint())
	}

	// Get
	getResult := client.TrustedCertificates.CertificateAuthority.Get(ctx, certificateauthority.GetOptions{
		TenantID: client.Auth.Tenant,
	})
	require.NoError(t, getResult.Err)
	assert.Equal(t, createResult.Data.Fingerprint(), getResult.Data.Fingerprint())

	// Update
	updateBody := map[string]any{
		"autoRegistrationEnabled": false,
		"status":                  "DISABLED",
	}
	updateResult := client.TrustedCertificates.Update(ctx, trustedcertificates.UpdateOptions{
		TenantID:    client.Auth.Tenant,
		Fingerprint: getResult.Data.Fingerprint(),
	}, updateBody)
	require.NoError(t, updateResult.Err)
	assert.Equal(t, 200, updateResult.HTTPStatus)
	assert.Equal(t, createResult.Data.Fingerprint(), updateResult.Data.Fingerprint())

	// Delete
	deleteResult := client.TrustedCertificates.CertificateAuthority.Delete(ctx, certificateauthority.DeleteOptions{
		TenantID: client.Auth.Tenant,
	})
	require.NoError(t, deleteResult.Err)
	assert.Equal(t, 204, deleteResult.HTTPStatus)
}

func Test_CertificateAuthority_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := c8yapi.WithDryRun(context.Background(), true)

	createResult := client.TrustedCertificates.CertificateAuthority.Create(ctx, certificateauthority.CreateOptions{})
	require.NoError(t, createResult.Err)
	assert.Empty(t, createResult.Data.Fingerprint(), "fingerprint should be empty in dry run mode")
}

func Test_CertificateAuthority_Delete_DryRun(t *testing.T) {
	client := testcore.CreateTestClient(t)

	// The Get step executes for real (GET requests are not skipped by dry-run).
	// Without a live CA the Get returns 404, so Delete propagates that error.
	// This test documents the expected behaviour: Delete fails fast when no CA exists.
	ctx := c8yapi.WithDryRun(context.Background(), true)

	result := client.TrustedCertificates.CertificateAuthority.Delete(ctx, certificateauthority.DeleteOptions{
		TenantID: client.Auth.Tenant,
	})
	// Expected: Get returns 404 (no CA in the test environment)
	require.Error(t, result.Err, "Delete should propagate Get failure when no CA is present")
}
