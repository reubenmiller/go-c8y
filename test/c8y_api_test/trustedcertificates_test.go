package api_test

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"io"
	"path"
	"testing"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates/certificaterevocationlist"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
)

func Test_TrustedCertificateListWithCACertificateOnly(t *testing.T) {
	client := testcore.CreateTestClient(t)
	client.Client.SetDebug(true)
	certs := client.TrustedCertificates.List(context.Background(), trustedcertificates.ListOptions{
		TenantID:             client.Auth.Tenant,
		CertificateAuthority: true,
	})
	assert.NoError(t, certs.Err)
	if certs.Data.Length() > 0 {
		for item := range jsondoc.DecodeIter[model.TrustedCertificate](certs.Data.Iter()) {
			assert.Equal(t, item.TenantCertificateAuthority, true)
			break
		}
	}
	assert.NotEmpty(t, certs.Meta["self"])
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
	cert := client.TrustedCertificates.Create(context.Background(), trustedcertificates.CreateOptions{
		TenantID: client.Auth.Tenant,
	}, localCert)
	assert.NoError(t, err)
	assert.Equal(t, cert.Data.Fingerprint(), localCert.Fingerprint)

	t.Cleanup(func() {
		// always run a cleanup in case the test fails before the cert is deleted
		client.TrustedCertificates.Delete(context.Background(), trustedcertificates.DeleteOptions{
			Fingerprint: cert.Data.Fingerprint(),
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
	verifiedCertificate := client.TrustedCertificates.ProofEndToEnd(context.Background(), trustedcertificates.ProofOptions{
		Fingerprint: cert.Data.Fingerprint(),
		TenantID:    client.Auth.Tenant,
		PrivateKey:  keyFile,
	})
	assert.NoError(t, verifiedCertificate.Err)
	assert.True(t, verifiedCertificate.Data.ProofOfPossessionValid())

	// confirm proof of possession
	// confirmedCertificate, err := client.TrustedCertificates.Confirm(context.Background(), trustedcertificates.ConfirmOptions{
	// 	Fingerprint: cert.Fingerprint,
	// })
	// assert.NoError(t, err)
	// assert.True(t, confirmedCertificate.ProofOfPossessionValid)

	// update
	updatedCert := client.TrustedCertificates.Update(context.Background(), trustedcertificates.UpdateOptions{
		Fingerprint: cert.Data.Fingerprint(),
		TenantID:    client.Auth.Tenant,
	}, model.TrustedCertificate{
		Status: model.TrustedCertificateStatusDisabled,
	})
	assert.NoError(t, updatedCert.Err)
	assert.Equal(t, updatedCert.Data.Status(), model.TrustedCertificateStatusDisabled)

	// get
	getCert := client.TrustedCertificates.Get(context.Background(), trustedcertificates.GetOptions{
		Fingerprint: cert.Data.Fingerprint(),
		TenantID:    client.Auth.Tenant,
	})
	assert.NoError(t, getCert.Err)
	assert.Equal(t, getCert.Data.Fingerprint(), cert.Data.Fingerprint())

	// list
	listCerts := client.TrustedCertificates.List(context.Background(), trustedcertificates.ListOptions{
		TenantID: client.Auth.Tenant,
	})
	assert.NoError(t, listCerts.Err)
	assert.GreaterOrEqual(t, listCerts.Data.Length(), 1)

	// delete
	result := client.TrustedCertificates.Delete(context.Background(), trustedcertificates.DeleteOptions{
		Fingerprint: updatedCert.Data.Fingerprint(),
		TenantID:    client.Auth.Tenant,
	})
	assert.NoError(t, result.Err)

	//
	// Certificate Revocation List

	// Add cert to CRL
	cert2 := new(model.TrustedCertificate)
	unmarshalErr := json.Unmarshal(updatedCert.Data.Bytes(), &cert2)
	assert.NoError(t, unmarshalErr)

	toAddCRL := model.NewTrustedCertificateRevocationCollectionFromCertificates(*cert2)
	result = client.TrustedCertificates.RevocationList.Add(context.Background(), toAddCRL)
	assert.NoError(t, result.Err)

	// Revocation list
	crl := client.TrustedCertificates.RevocationList.List(context.Background(), certificaterevocationlist.GetOptions{})
	assert.NoError(t, crl.Err)
	crlBinary, err := io.ReadAll(crl.Data.Reader())
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

	result = client.TrustedCertificates.RevocationList.AddFile(context.Background(), certificaterevocationlist.AddFileFileOptions{
		Reader: bytes.NewReader(buf.Bytes()),
	})
	assert.NoError(t, result.Err, "CRL file list should be able to be uploaded")
}
