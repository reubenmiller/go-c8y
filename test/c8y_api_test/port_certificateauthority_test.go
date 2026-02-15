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

	// Delete (setup)
	// Ignore error as certificate may not exist
	client.TrustedCertificates.Delete(ctx, trustedcertificates.DeleteOptions{
		TenantID:    client.Auth.Tenant,
		Fingerprint: "",
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
	deleteResult := client.TrustedCertificates.Delete(ctx, trustedcertificates.DeleteOptions{
		TenantID:    client.Auth.Tenant,
		Fingerprint: createResult.Data.Fingerprint(),
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
