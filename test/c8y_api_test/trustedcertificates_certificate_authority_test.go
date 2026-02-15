package api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates/certificateauthority"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TrustedCertificatesCertificateAuthority(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// get
	cert := client.TrustedCertificates.CertificateAuthority.Get(context.Background(), certificateauthority.GetOptions{
		TenantID: client.Auth.Tenant,
	})
	assert.NoError(t, cert.Err)
	assert.NotEqual(t, 0, cert.Data.Length())
	assert.Equal(t, cert.Data.TenantCertificateAuthority(), true)

	// create
	cert2 := client.TrustedCertificates.CertificateAuthority.Create(context.Background(), certificateauthority.CreateOptions{})
	assert.Error(t, cert2.Err)
	assert.True(t, core.ErrHasStatus(cert2.Err, 409))
	assert.Equal(t, 0, cert2.Data.Length())

	// get or create
	cert3 := client.TrustedCertificates.CertificateAuthority.GetOrCreate(context.Background(), certificateauthority.GetOptions{})
	assert.NoError(t, cert3.Err)
	assert.NotEmpty(t, cert3.Data.Self())
	assert.Equal(t, cert3.Data.TenantCertificateAuthority(), true)
}
