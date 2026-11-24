package api_test

import (
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices/registration"
	"github.com/reubenmiller/go-c8y/v2/pkg/certutil"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newCSR(t *testing.T, externalID string) *x509.CertificateRequest {
	t.Helper()
	keyPEM, err := certutil.MakeEllipticPrivateKeyPEM()
	require.NoError(t, err)
	key, err := certutil.ParsePrivateKeyPEM(keyPEM)
	require.NoError(t, err)
	csr, err := certutil.CreateCertificateSigningRequest(pkix.Name{CommonName: externalID}, key)
	require.NoError(t, err)
	return csr
}

func Test_Devices_Enroll_Register(t *testing.T) {
	testcore.SkipLive(t)
	client, _ := testcore.CreateTestClientWithFakeServer(t)
	ctx := context.Background()

	id := "enroll" + testingutils.RandomString(8)
	called := false
	result := client.Devices.Enroll(ctx, devices.EnrollOptions{
		ExternalID:             id,
		CSR:                    newCSR(t, id),
		RemoveRequest:          true,
		Register:               true,
		OnRegistrationRequired: func(devices.RegistrationRequired) { called = true },
	})
	require.NoError(t, result.Err)
	require.NotNil(t, result.Data.Certificate)
	assert.Equal(t, id, result.Data.Subject.CommonName)
	assert.False(t, called, "a registered device needs no user")
}

func Test_Devices_Enroll_RegisteredByUser(t *testing.T) {
	testcore.SkipLive(t)
	client, _ := testcore.CreateTestClientWithFakeServer(t)
	ctx := context.Background()

	id := "enroll" + testingutils.RandomString(8)
	var required devices.RegistrationRequired
	attempts := 0
	result := client.Devices.Enroll(ctx, devices.EnrollOptions{
		ExternalID: id,
		CSR:        newCSR(t, id),
		Interval:   50 * time.Millisecond,
		OnRegistrationRequired: func(r devices.RegistrationRequired) {
			required = r
		},
		OnAttemptError: func(error) {
			// The user registers the device after the first attempt
			if attempts++; attempts == 1 {
				reg := client.Devices.Registration.Register(ctx, registration.RegisterOptions{
					ExternalID:    required.ExternalID,
					AuthType:      registration.AuthTypeCertificates,
					EnrollmentOTP: required.OneTimePassword,
				})
				require.NoError(t, reg.Err)
			}
		},
	})
	require.NoError(t, result.Err)
	assert.Equal(t, id, result.Data.Subject.CommonName)
	assert.Equal(t, 1, attempts)
	assert.NotEmpty(t, required.OneTimePassword, "a one-time password is generated")
	assert.NotContains(t, required.OneTimePassword, ".")
	assert.Contains(t, required.URL, "externalId="+id)
	assert.Contains(t, required.URL, "one-time-password="+required.OneTimePassword)
	assert.NoError(t, required.Err)
}

func Test_Devices_Enroll_Timeout(t *testing.T) {
	testcore.SkipLive(t)
	client, _ := testcore.CreateTestClientWithFakeServer(t)

	result := client.Devices.Enroll(context.Background(), devices.EnrollOptions{
		ExternalID: "notregistered",
		CSR:        newCSR(t, "notregistered"),
		Interval:   50 * time.Millisecond,
		Timeout:    300 * time.Millisecond,
	})
	require.ErrorIs(t, result.Err, devices.ErrEnrollTimeout)
	require.ErrorIs(t, result.Err, context.DeadlineExceeded)
	assert.Contains(t, result.Err.Error(), "401", "the last error is described")
	var apiErr *core.Error
	assert.False(t, errors.As(result.Err, &apiErr), "the last error is not the error")
}

func Test_Devices_Enroll_DryRun(t *testing.T) {
	testcore.SkipLive(t)
	client, srv := testcore.CreateTestClientWithFakeServer(t)

	var requests []string
	ctx := api.WithDryRunHandler(api.WithDryRun(context.Background(), true), func(req *http.Request) {
		requests = append(requests, req.Method+" "+req.URL.Path)
	})
	before := srv.ManagedObjects.Count()
	result := client.Devices.Enroll(ctx, devices.EnrollOptions{
		ExternalID:    "dryrun01",
		CSR:           newCSR(t, "dryrun01"),
		RemoveRequest: true,
		Register:      true,
	})
	require.NoError(t, result.Err)
	assert.Equal(t, []string{
		"DELETE /devicecontrol/newDeviceRequests/dryrun01",
		"POST /devicecontrol/bulkNewDeviceRequests",
		"POST /.well-known/est/simpleenroll",
	}, requests)
	assert.Equal(t, before, srv.ManagedObjects.Count(), "nothing is created")
	n := 0
	for range result.Documents() {
		n++
	}
	assert.Zero(t, n)
}
