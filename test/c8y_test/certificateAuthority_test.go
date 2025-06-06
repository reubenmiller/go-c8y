package c8y_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestCertificateAuthority_get(t *testing.T) {
	t.Skip("Skip due to affects other tests due to performing destructive operations on the CA certificate")
	client := createTestClient()

	ctx := context.Background()

	// Delete (setup)
	_, setupErr := client.CertificateAuthority.Delete(ctx, "")
	testingutils.Ok(t, setupErr)

	// Create
	cert, err := client.CertificateAuthority.Create(ctx, c8y.CertificateAuthorityOptions{
		AutoRegistration: true,
	})
	testingutils.Ok(t, err)
	testingutils.Assert(t, cert.Fingerprint != "", "fingerprint should not be empty")

	// Create again (should be idempotent)
	certDuplicate, err := client.CertificateAuthority.Create(ctx, c8y.CertificateAuthorityOptions{
		AutoRegistration: true,
	})
	testingutils.Ok(t, err)
	testingutils.Equals(t, cert.Fingerprint, certDuplicate.Fingerprint)

	// Get
	cert2, err := client.CertificateAuthority.Get(ctx)
	testingutils.Ok(t, err)
	testingutils.Equals(t, cert.Fingerprint, cert2.Fingerprint)

	// Update dry-run (get should work, but the update should not)
	dryRunContext := c8y.WithCommonOptionsContext(ctx, c8y.CommonOptions{
		DryRun: true,
	})
	_, _, dryRunErr := client.CertificateAuthority.Update(
		dryRunContext,
		"",
		c8y.NewCertificate().
			WithAutoRegistration(true),
	)
	testingutils.Ok(t, dryRunErr)

	// Update (skip due to race conditions with other tests)
	cert3, resp, err := client.CertificateAuthority.Update(
		ctx,
		cert2.Fingerprint,
		c8y.NewCertificate().
			WithAutoRegistration(false).
			WithStatus(c8y.CertificateStatusDisabled),
	)
	testingutils.Ok(t, err)
	testingutils.Equals(t, resp.StatusCode(), http.StatusOK)
	testingutils.Equals(t, cert.Fingerprint, cert3.Fingerprint)

	// Delete
	resp, deleteErr := client.CertificateAuthority.Delete(ctx, cert.Fingerprint)
	testingutils.Ok(t, deleteErr)
	testingutils.Equals(t, resp.StatusCode(), http.StatusNoContent)
}

func TestCertificateAuthority_DryRun(t *testing.T) {
	client := createTestClient()
	client.SetRequestOptions(c8y.DefaultRequestOptions{
		DryRun: true,
	})

	ctx := context.Background()

	cert, setupErr := client.CertificateAuthority.Create(ctx, c8y.CertificateAuthorityOptions{
		AutoRegistration: true,
	})
	testingutils.Ok(t, setupErr)
	testingutils.Assert(t, cert == nil, "cert should be nil")
}
