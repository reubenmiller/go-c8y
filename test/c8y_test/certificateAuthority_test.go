package c8y_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func TestCertificateAuthority_get(t *testing.T) {
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

	// Update
	cert3, resp, err := client.CertificateAuthority.Update(ctx, cert2.Fingerprint, &c8y.Certificate{
		Status:                  c8y.CertificateStatusDisabled,
		AutoRegistrationEnabled: false,
	})
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
