package api_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices/enrollment"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/identity"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates/certificateauthority"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Enrollment_OneTimePasswordGenerator(t *testing.T) {
	client := testcore.CreateTestClient(t)
	i := 0
	for i < 100 {
		opt, err := client.Devices.Enrollment.GenerateOneTimePassword()
		require.NoError(t, err)
		fmt.Printf("password: %s\n", opt)
		i += 1
	}
}

func Test_Enrollment_PollEnroll(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	deviceID := "TestDevice" + testingutils.RandomString(10)

	// Create private key
	keyPem, err := certutil.MakeEllipticPrivateKeyPEM()
	require.NoError(t, err)

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	require.NoError(t, err)

	// Enroll
	csr, err := client.Devices.Enrollment.CreateCertificateSigningRequest(deviceID, key)
	require.NoError(t, err)
	assert.Equal(t, deviceID, csr.Subject.CommonName)

	result := <-client.Devices.Enrollment.PollEnroll(ctx, enrollment.DeviceEnrollmentOption{
		ExternalID:                deviceID,
		OneTimePassword:           "",
		Interval:                  1 * time.Second,
		Timeout:                   1500 * time.Millisecond,
		CertificateSigningRequest: csr,
		Banner:                    enrollment.NewDeviceEnrollmentBannerOptions(true, true),
	})
	assert.NotNil(t, result.Err, "enrollment should timeout")
	assert.False(t, result.Ok())
}

func Test_Enrollment_Register(t *testing.T) {
	testcore.SkipOffline(t, "requires certificate authority infrastructure")
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	// Ensure there is a Cumulocity CA Certificate
	_ = client.TrustedCertificates.CertificateAuthority.Create(ctx, certificateauthority.CreateOptions{})

	deviceID := "TestDevice" + testingutils.RandomString(10)
	otp, err := client.Devices.Enrollment.GenerateOneTimePassword()
	require.NoError(t, err)

	// Delete any pre-existing values, but ignore any errors
	client.Devices.Registration.Delete(ctx, deviceID)

	// Cleanup all of the artifacts afterwards
	t.Cleanup(func() {
		xidResult := client.Identity.Get(ctx, identity.IdentityOptions{
			Type:       "c8y_Serial",
			ExternalID: deviceID,
		})
		if xidResult.Err == nil {
			client.ManagedObjects.Delete(ctx, xidResult.Data.ManagedObjectID(), managedobjects.DeleteOptions{
				Cascade: true,
			})
		}
		client.Users.Delete(ctx, users.DeleteOptions{
			ID:     users.ByDeviceUser(deviceID),
			Tenant: client.Auth.Tenant,
		})
	})

	client.SetDebug(true)

	csvContents := bytes.NewBufferString("")
	csvErr := model.BulkRegistrationCertificateAuthorityWriter(
		csvContents,
		model.BulkRegistrationRecord{
			ID:            deviceID,
			AuthType:      model.BulkRegistrationAuthTypeCertificates,
			EnrollmentOTP: otp,
			Name:          deviceID,
			Type:          "test_ci_reg",
			IDType:        "c8y_Serial",
			IsAgent:       true,
		},
	)
	require.NoError(t, csvErr)
	fmt.Printf("\n\n%s\n\n", csvContents)

	regResult := client.Devices.Registration.CreateBulk(ctx, core.UploadFileOptions{
		Reader: csvContents,
	})
	require.NoError(t, regResult.Err)

	// Create private key
	keyPem, err := certutil.MakeEllipticPrivateKeyPEM()
	require.NoError(t, err)

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	require.NoError(t, err)

	// Enroll
	csr, err := client.Devices.Enrollment.CreateCertificateSigningRequest(deviceID, key)
	require.NoError(t, err)
	assert.Equal(t, deviceID, csr.Subject.CommonName)

	enrollResult := client.Devices.Enrollment.Enroll(ctx, enrollment.EnrollOptions{
		ExternalID:      deviceID,
		OneTimePassword: otp,
		CSR:             csr,
	})
	require.NoError(t, enrollResult.Err)
	require.NotNil(t, enrollResult.Data)

	// Create client which uses the device certificate
	// It should automatically request a token to use for API calls
	deviceClient := api.NewClient(api.ClientOptions{
		BaseURL:       client.HTTPClient.BaseURL(),
		Debug:         true,
		ShowSensitive: true,
		Auth: authentication.AuthOptions{
			CertificateKey: string(keyPem),
			Certificate:    string(certutil.MarshalCertificateToPEM(enrollResult.Data.Raw)),
		},
	})
	assert.NotEmpty(t, deviceClient.Auth.Token)

	// Query data
	deviceResult := deviceClient.ManagedObjects.List(context.Background(), managedobjects.ListOptions{})
	assert.NoError(t, deviceResult.Err)
	assert.Greater(t, deviceResult.Data.Length(), 0)

	// Re-enroll
	secondCSR, err := client.Devices.Enrollment.CreateCertificateSigningRequest(deviceID, key)
	require.NoError(t, err)

	secondEnrollResult := deviceClient.Devices.Enrollment.ReEnroll(ctx, enrollment.ReEnrollOptions{
		CSR: secondCSR,
	})
	require.NoError(t, secondEnrollResult.Err)
	assert.Equal(t, 200, secondEnrollResult.HTTPStatus)
	assert.Equal(t, deviceID, secondEnrollResult.Data.Subject.CommonName)

	secondCertPEM := certutil.MarshalCertificateToPEM(enrollResult.Data.Raw)
	assert.Greater(t, len(secondCertPEM), 0, "certificate should not be empty")

	// Open another client using the newly issued certificate
	deviceClient2 := api.NewClient(api.ClientOptions{
		BaseURL: client.HTTPClient.BaseURL(),
		Auth: authentication.AuthOptions{
			CertificateKey: string(keyPem),
			Certificate:    string(secondCertPEM),
		},
	})
	assert.NotEmpty(t, deviceClient2.Auth.Token)
}
