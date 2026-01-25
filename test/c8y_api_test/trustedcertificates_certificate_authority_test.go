package c8y_api_test

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/trustedcertificates/certificateauthority"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TrustedCertificatesCertificateAuthority(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	// get
	cert, err := client.TrustedCertificates.CertificateAuthority.Get(context.Background(), certificateauthority.GetOptions{
		TenantID: client.Auth.Tenant,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, cert.Self)
	assert.Equal(t, cert.TenantCertificateAuthority, true)

	// create
	cert2, err := client.TrustedCertificates.CertificateAuthority.Create(context.Background(), certificateauthority.CreateOptions{})
	assert.Error(t, err)
	assert.True(t, core.ErrHasStatus(err, 409))
	assert.Nil(t, cert2)

	// get or create
	cert3, err := client.TrustedCertificates.CertificateAuthority.GetOrCreate(context.Background(), certificateauthority.GetOptions{})
	assert.NoError(t, err)
	assert.NotEmpty(t, cert3.Self)
	assert.Equal(t, cert3.TenantCertificateAuthority, true)
}
