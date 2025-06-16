package c8y_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

func TestSimpleEnrollment_OneTimePasswordGenerator(t *testing.T) {
	client := createTestClient()
	i := 0
	for i < 100 {
		opt, err := client.DeviceEnrollment.GenerateOneTimePassword()
		testingutils.Ok(t, err)
		fmt.Printf("password: %s\n", opt)
		i += 1
	}
}

func TestSimpleEnrollment_PollEnroll(t *testing.T) {
	client := createTestClient()

	deviceID := "TestDevice" + testingutils.RandomString(10)

	// Create private key
	keyPem, err := certutil.MakeEllipticPrivateKeyPEM()
	testingutils.Ok(t, err)

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	testingutils.Ok(t, err)

	// Enroll
	csr, err := client.DeviceEnrollment.CreateCertificateSigningRequest(deviceID, key)
	testingutils.Ok(t, err)
	testingutils.Equals(t, deviceID, csr.Subject.CommonName)

	result := <-client.DeviceEnrollment.PollEnroll(context.Background(), c8y.DeviceEnrollmentOption{
		ExternalID:                deviceID,
		OneTimePassword:           "",
		Interval:                  1 * time.Second,
		Timeout:                   1500 * time.Millisecond,
		CertificateSigningRequest: csr,
		Banner:                    c8y.NewDeviceEnrollmentBannerOptions(true, true),
	})
	testingutils.Assert(t, result.Err != nil, "enrollment should timeout")
	testingutils.Equals(t, false, result.Ok())
}

func TestSimpleEnrollment_Register(t *testing.T) {
	client := createTestClient()

	// Ensure there is a Cumulocity CA Certificate
	_, err := client.CertificateAuthority.Create(context.Background(), c8y.CertificateAuthorityOptions{
		AutoRegistration: true,
		Status:           c8y.CertificateStatusEnabled,
	})
	testingutils.Ok(t, err)

	deviceID := "TestDevice" + testingutils.RandomString(10)
	otp, err := client.DeviceEnrollment.GenerateOneTimePassword()
	testingutils.Ok(t, err)

	// Delete any pre-existing values, but ignore any errors
	client.DeviceCredentials.Delete(context.Background(), deviceID)

	// Cleanup all of the artifacts afterwards
	t.Cleanup(func() {
		if xid, _, err := client.Identity.GetExternalID(context.Background(), "c8y_Serial", deviceID); err == nil {
			deleteOptions := &c8y.ManagedObjectDeleteOptions{}
			deleteOptions.WithCascade(true)
			client.Inventory.DeleteWithOptions(context.Background(), xid.ManagedObject.ID, deleteOptions)
		}
		client.User.Delete(context.Background(), "device_"+deviceID)
	})

	csvContents := bytes.NewBufferString("")
	csvErr := c8y.BulkRegistrationRecordWriter(
		csvContents,
		c8y.BulkRegistrationRecord{
			ID:            deviceID,
			AuthType:      c8y.BulkRegistrationAuthTypeCertificates,
			EnrollmentOTP: otp,
			Name:          deviceID,
			Type:          "test_ci_reg",
			IDType:        "c8y_Serial",
			IsAgent:       true,
		},
	)
	testingutils.Ok(t, csvErr)

	_, _, regErr := client.DeviceCredentials.CreateBulk(context.Background(), csvContents)
	testingutils.Ok(t, regErr)

	// Create private key
	keyPem, err := certutil.MakeEllipticPrivateKeyPEM()
	testingutils.Ok(t, err)

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	testingutils.Ok(t, err)

	// Enroll
	csr, err := client.DeviceEnrollment.CreateCertificateSigningRequest(deviceID, key)
	testingutils.Ok(t, err)
	testingutils.Equals(t, deviceID, csr.Subject.CommonName)

	cert, resp, err := client.DeviceEnrollment.Enroll(context.Background(), deviceID, otp, csr)
	testingutils.Ok(t, err)
	testingutils.Equals(t, true, resp != nil)

	certPEM := certutil.MarshalCertificateToPEM(cert.Raw)

	// Use the certificate to request an access token to use for re-enrollment
	clientCert, err := tls.X509KeyPair(certPEM, keyPem)
	testingutils.Ok(t, err)

	token, tokenResp, err := client.DeviceEnrollment.RequestAccessToken(context.Background(), &clientCert, nil)
	testingutils.Ok(t, err)
	testingutils.Equals(t, 200, tokenResp.StatusCode())
	testingutils.Assert(t, token.AccessToken != "", "Token should not be empty")

	// Re-enroll
	secondCSR, err := client.DeviceEnrollment.CreateCertificateSigningRequest(deviceID, key)
	testingutils.Ok(t, err)
	secondCert, resp, err := client.DeviceEnrollment.ReEnroll(context.Background(), c8y.ReEnrollOptions{
		Token: token.AccessToken,
		CSR:   secondCSR,
	})
	testingutils.Ok(t, err)
	testingutils.Equals(t, 200, resp.StatusCode())
	testingutils.Equals(t, deviceID, secondCert.Subject.CommonName)

	secondCertPEM := certutil.MarshalCertificateToPEM(cert.Raw)
	testingutils.Assert(t, len(secondCertPEM) > 0, "certificate should not be empty")

	// Use the second certificate to request another token
	secondClientCert, err := tls.X509KeyPair(secondCertPEM, keyPem)
	testingutils.Ok(t, err)
	secondToken, secondTokenResp, err := client.DeviceEnrollment.RequestAccessToken(context.Background(), &secondClientCert, nil)
	testingutils.Ok(t, err)
	testingutils.Equals(t, 200, secondTokenResp.StatusCode())
	testingutils.Assert(t, secondToken.AccessToken != "", "Token should not be empty")
}
