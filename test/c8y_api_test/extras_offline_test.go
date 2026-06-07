package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/auditrecords"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/bulkoperations"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Authentication_DecodeBasicAuth(t *testing.T) {
	creds, err := authentication.DecodeBasicAuth("dGVuYW50L3VzZXI6cGFzcw==", "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "tenant", creds.Tenant)
	assert.Equal(t, "user", creds.Username)
	assert.Equal(t, "pass", creds.Password)
}

func Test_Authentication_ParseLoginType(t *testing.T) {
	got, err := authentication.ParseLoginType("BASIC")
	require.NoError(t, err)
	assert.NotEmpty(t, got)

	_, err = authentication.ParseLoginType("invalid")
	assert.Error(t, err)
}

func Test_Authentication_GetAuthTypes(t *testing.T) {
	auth := authentication.AuthOptions{
		Token:    "tok",
		Username: "u",
		Password: "p",
	}
	types := auth.GetAuthTypes()
	assert.NotEmpty(t, types)
	for _, t1 := range types {
		assert.NotEmpty(t, t1.String())
	}
}

func Test_Alarms_UpsertRaw(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	mo := testcore.CreateManagedObject(t, client)

	res := client.Alarms.UpsertRaw(ctx, map[string]any{
		"source":   map[string]any{"id": mo.Data.ID()},
		"type":     "ci_TestType",
		"severity": "MINOR",
		"text":     "raw upsert",
		"status":   "ACTIVE",
		"time":     "2024-01-01T00:00:00Z",
	})
	require.NoError(t, res.Err)
}

func Test_Alarms_ListAll(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	it := client.Alarms.ListAll(ctx, alarms.ListOptions{})
	require.NoError(t, it.Err())
}

func Test_Alarms_SubscribeOffline(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, _ := testcore.CreateTestClientWithFakeServer(t)

	ch := make(chan *realtime.Message, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	subRes := client.Alarms.Subscribe(ctx, "12345", ch)
	require.NoError(t, subRes.Err)
}

func Test_AuditRecords_ListAll(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	it := client.AuditRecords.ListAll(ctx, auditrecords.ListOptions{})
	require.NoError(t, it.Err())
}

func Test_Binaries_List(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	listRes := client.Binaries.List(ctx, binaries.ListOptions{})
	_ = listRes
	itAll := client.Binaries.ListAll(ctx, binaries.ListOptions{})
	_ = itAll
}

func Test_BulkOperations_ListAll(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	it := client.BulkOperations.ListAll(ctx, bulkoperations.ListOptions{})
	_ = it
}

func Test_TrustedCertificates_ListAll(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	it := client.TrustedCertificates.ListAll(ctx, trustedcertificates.ListOptions{})
	_ = it
}

func Test_Devices_BulkRegistration_GeneratePassword(t *testing.T) {
	client := testcore.CreateTestClient(t)
	for i := 0; i < 3; i++ {
		pw, err := client.Devices.Registration.GeneratePassword()
		require.NoError(t, err)
		assert.NotEmpty(t, pw)
		assert.GreaterOrEqual(t, len(pw), 8)
	}
}

func Test_TOTP_GenerateSecret(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	res := client.Users.CurrentUser.TOTP.GenerateSecret(ctx)
	require.NoError(t, res.Err)
}

func Test_TOTP_VerifyCode(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	res := client.Users.CurrentUser.TOTP.VerifyCode(ctx, "123456")
	require.NoError(t, res.Err)
}

func Test_TOTP_SetActivity(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	res := client.Users.CurrentUser.TOTP.SetActivity(ctx, true)
	require.NoError(t, res.Err)
}

func Test_ManagedObjects_CreateWithExternalID(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	extID := "extid-" + testingutils.RandomString(6)
	res := client.Devices.GetOrCreateByExternalID(ctx, managedobjects.GetOrCreateByExternalIDOptions{
		ExternalID:     extID,
		ExternalIDType: "c8y_Serial",
		Body: map[string]any{
			"name": "ExternalDev",
		},
	})
	require.NoError(t, res.Err)
}

func Test_Alarms_Create_WithResolver(t *testing.T) {
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	mo := testcore.CreateManagedObject(t, client)
	res := client.Alarms.Create(ctx, alarms.CreateOptions{
		Source:   managedobjects.DeviceRef(mo.Data.ID()),
		Type:     "ci_resolver",
		Severity: "MINOR",
		Text:     "resolver test",
		Time:     time.Now(),
		Fragments: []model.Fragment{
			model.Frag("c8y_Custom", map[string]any{"key": "value"}),
		},
	})
	require.NoError(t, res.Err)
}
