package api

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/restrictedroles"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/trustedcertificates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RestrictedRoles(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		if r.Method == http.MethodGet {
			return 200, `["ROLE_TENANT_ADMIN"]`
		}
		return 200, `{"id":"t100"}`
	})
	client := newScriptClient(ss)
	rr := client.Tenants.RestrictedRoles

	list := rr.List(context.Background(), restrictedroles.Options{})
	require.NoError(t, list.Err)
	assert.Equal(t, []string{"ROLE_TENANT_ADMIN"}, list.Data)
	require.Len(t, ss.matching(http.MethodGet, "/tenant/tenants/t100/applications/restricted-roles"), 1)

	require.NoError(t, rr.Replace(context.Background(), restrictedroles.Options{}, []string{"ROLE_TENANT_ADMIN"}).Err)
	put := ss.matching(http.MethodPut, "/tenant/tenants/t100/applications/restricted-roles")
	require.Len(t, put, 1)
	assert.JSONEq(t, `["ROLE_TENANT_ADMIN"]`, put[0].Body)

	require.NoError(t, rr.Add(context.Background(), restrictedroles.Options{Tenant: "t200"}, "ROLE_TENANT_ADMIN").Err)
	require.NoError(t, rr.Remove(context.Background(), restrictedroles.Options{Tenant: "t200"}, "ROLE_TENANT_ADMIN").Err)
	assert.Len(t, ss.matching(http.MethodPost, "/tenant/tenants/t200/applications/restricted-roles/ROLE_TENANT_ADMIN"), 1)
	assert.Len(t, ss.matching(http.MethodDelete, "/tenant/tenants/t200/applications/restricted-roles/ROLE_TENANT_ADMIN"), 1)
}

func Test_SmartRestBindExternalID(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 201, `{"externalId":"myTemplateXid","type":"c8y_SmartRestDeviceIdentifier"}`
	})
	res := newScriptClient(ss).SmartRestTemplates.BindExternalID(context.Background(), "123", "myTemplateXid")
	require.NoError(t, res.Err)
	posts := ss.matching(http.MethodPost, "/inventory/templates/globalId/123/externalId")
	require.Len(t, posts, 1)
	assert.JSONEq(t, `{"externalId":"myTemplateXid","type":"c8y_SmartRestDeviceIdentifier"}`, posts[0].Body)
}

func Test_TrustedCertificatesBulkAndVerifyChain(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		if r.Path == "/tenant/trusted-certificates/verify-cert-chain" {
			return 200, `{"successfullyValidated":true,"tenantId":"t100"}`
		}
		return 201, `{"certificates":[{"fingerprint":"abc"}]}`
	})
	client := newScriptClient(ss)

	bulk := client.TrustedCertificates.CreateBulk(context.Background(), trustedcertificates.CreateOptions{}, map[string]any{"certificates": []any{}})
	require.NoError(t, bulk.Err)
	assert.Len(t, ss.matching(http.MethodPost, "/tenant/tenants/t100/trusted-certificates/bulk"), 1)

	verify := client.TrustedCertificates.VerifyChain(context.Background(), trustedcertificates.VerifyChainOptions{Chain: rsaCert(t).certPEM})
	require.NoError(t, verify.Err)
	assert.True(t, verify.Data.SuccessfullyValidated())
	assert.Len(t, ss.matching(http.MethodPost, "/tenant/trusted-certificates/verify-cert-chain"), 1)
}

func Test_ChildUnassignMany(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) { return 204, `` })
	client := newScriptClient(ss)

	require.NoError(t, client.ManagedObjects.ChildDevices.UnassignMany(context.Background(), "1", "2", "3").Err)
	require.NoError(t, client.ManagedObjects.ChildAssets.UnassignMany(context.Background(), "1", "4").Err)
	require.NoError(t, client.ManagedObjects.ChildAdditions.UnassignMany(context.Background(), "1", "5").Err)

	dev := ss.matching(http.MethodDelete, "/inventory/managedObjects/1/childDevices")
	require.Len(t, dev, 1)
	assert.JSONEq(t, `{"references":[{"managedObject":{"id":"2"}},{"managedObject":{"id":"3"}}]}`, dev[0].Body)
	assert.Len(t, ss.matching(http.MethodDelete, "/inventory/managedObjects/1/childAssets"), 1)
	assert.Len(t, ss.matching(http.MethodDelete, "/inventory/managedObjects/1/childAdditions"), 1)
}

func Test_TOTPGetActivity(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) { return 200, `{"isActive":true}` })
	res := newScriptClient(ss).Users.Current.TOTP.GetActivity(context.Background())
	require.NoError(t, res.Err)
	assert.True(t, res.Data.IsActive())
	assert.Len(t, ss.matching(http.MethodGet, "/user/currentUser/totpSecret/activity"), 1)
}

func Test_ApplicationDownloadBinaryAndListByOwner(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		if r.Path == "/application/applications/7/binaries/9" {
			return 200, "binary-content"
		}
		return 200, `{"applications":[]}`
	})
	client := newScriptClient(ss)

	bin := client.Applications.DownloadBinary(context.Background(), "7", "9")
	require.NoError(t, bin.Err)
	content, err := io.ReadAll(bin.Data.Reader())
	require.NoError(t, err)
	_ = bin.Data.Close()
	assert.Equal(t, "binary-content", string(content))

	require.NoError(t, client.Applications.ListByOwner(context.Background(), applications.ListByOwnerOptions{TenantID: "t100"}).Err)
	assert.Len(t, ss.matching(http.MethodGet, "/application/applicationsByOwner/t100"), 1, "ListByOwner must call applicationsByOwner")
}
