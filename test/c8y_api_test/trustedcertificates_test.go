package c8y_api_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"io"
	"path"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/trustedcertificates/certificaterevocationlist"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TrustedCertificateListWithCACertificateOnly(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	certs, err := client.TrustedCertificates.List(context.Background(), trustedcertificates.ListOptions{
		TenantID:             client.Auth.Tenant,
		CertificateAuthority: true,
	})
	assert.NoError(t, err)
	if len(certs.Certificates) > 0 {
		assert.Equal(t, certs.Certificates[0].TenantCertificateAuthority, true)
	}
	assert.NotEmpty(t, certs.Self)
}

func Test_TrustedCertifcates(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)

	keyFile := path.Join(t.TempDir(), "key.pem")
	_, _, err := certutil.LoadOrGenerateKeyFile(keyFile)
	assert.NoError(t, err)
	key, err := certutil.PrivateKeyFromFile(keyFile)
	assert.NoError(t, err)

	commonName := "test_ci" + testingutils.RandomString(16)

	rootCert, err := certutil.NewSelfSignedCertificate(key, commonName)
	assert.NoError(t, err)

	x509Cert, err := x509.ParseCertificate(rootCert.Raw)
	assert.NoError(t, err)
	assert.Equal(t, x509Cert.Subject.CommonName, commonName)

	localCert, err := model.NewTrustedCertificate(rootCert.Raw)
	assert.NoError(t, err)
	assert.NotEmpty(t, localCert.Fingerprint)

	// create
	cert, err := client.TrustedCertificates.Create(context.Background(), trustedcertificates.CreateOptions{
		TenantID: client.Auth.Tenant,
	}, localCert)
	assert.NoError(t, err)
	assert.Equal(t, cert.Fingerprint, localCert.Fingerprint)

	t.Cleanup(func() {
		// always run a cleanup in case the test fails before the cert is deleted
		client.TrustedCertificates.Delete(context.Background(), trustedcertificates.DeleteOptions{
			Fingerprint: cert.Fingerprint,
			TenantID:    client.Auth.Tenant,
		})
	})

	// confirm proof of possession (should fail)
	// confirmedCertificateFail, err := client.TrustedCertificates.Confirm(context.Background(), trustedcertificates.ConfirmOptions{
	// 	Fingerprint: cert.Fingerprint,
	// })
	// assert.Error(t, err)
	// assert.Nil(t, confirmedCertificateFail)
	// assert.False(t, confirmedCertificateFail.ProofOfPossessionValid)

	// proof of possession
	verifiedCertificate, err := client.TrustedCertificates.ProofEndToEnd(context.Background(), trustedcertificates.ProofOptions{
		Fingerprint: cert.Fingerprint,
		TenantID:    client.Auth.Tenant,
		PrivateKey:  keyFile,
	})
	assert.NoError(t, err)
	assert.True(t, verifiedCertificate.ProofOfPossessionValid)

	// confirm proof of possession
	// confirmedCertificate, err := client.TrustedCertificates.Confirm(context.Background(), trustedcertificates.ConfirmOptions{
	// 	Fingerprint: cert.Fingerprint,
	// })
	// assert.NoError(t, err)
	// assert.True(t, confirmedCertificate.ProofOfPossessionValid)

	// update
	updatedCert, err := client.TrustedCertificates.Update(context.Background(), trustedcertificates.UpdateOptions{
		Fingerprint: cert.Fingerprint,
		TenantID:    client.Auth.Tenant,
	}, model.TrustedCertificate{
		Status: model.TrustedCertificateStatusDisabled,
	})
	assert.NoError(t, err)
	assert.Equal(t, updatedCert.Status, model.TrustedCertificateStatusDisabled)

	// get
	getCert, err := client.TrustedCertificates.Get(context.Background(), trustedcertificates.GetOptions{
		Fingerprint: cert.Fingerprint,
		TenantID:    client.Auth.Tenant,
	})
	assert.NoError(t, err)
	assert.Equal(t, getCert.Fingerprint, cert.Fingerprint)

	// list
	listCerts, err := client.TrustedCertificates.List(context.Background(), trustedcertificates.ListOptions{
		TenantID: client.Auth.Tenant,
	})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(listCerts.Certificates), 1)

	// delete
	err = client.TrustedCertificates.Delete(context.Background(), trustedcertificates.DeleteOptions{
		Fingerprint: updatedCert.Fingerprint,
		TenantID:    client.Auth.Tenant,
	})
	assert.NoError(t, err)

	//
	// Certificate Revocation List

	// Add cert to CRL
	toAddCRL := model.NewTrustedCertificateRevocationCollectionFromCertificates(*updatedCert)
	err = client.TrustedCertificates.RevocationList.Add(context.Background(), toAddCRL)
	assert.NoError(t, err)

	// Revocation list
	crl, err := client.TrustedCertificates.RevocationList.List(context.Background(), certificaterevocationlist.GetOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, crl)
	crlBinary, err := io.ReadAll(crl.Reader())
	assert.NoError(t, err)
	assert.NotEmpty(t, crlBinary)

	// parse CRL
	crlParsed, err := x509.ParseRevocationList(crlBinary)
	assert.NoError(t, err, "CRL should be parsable by std lib")
	assert.NotNil(t, crlParsed)

	// create a csv with the certificate to be revoked
	buf := &bytes.Buffer{}
	_ = toAddCRL.WriteCSV(buf)
	assert.NotEmpty(t, buf.Bytes())

	err = client.TrustedCertificates.RevocationList.AddFile(context.Background(), certificaterevocationlist.AddFileFileOptions{
		Reader: bytes.NewReader(buf.Bytes()),
	})
	assert.NoError(t, err, "CRL file list should be able to be uploaded")
}
