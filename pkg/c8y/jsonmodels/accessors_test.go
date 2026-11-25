package jsonmodels

import "testing"

func TestAccessors(t *testing.T) {
	app := NewApplication([]byte(`{"id":"1","contextPath":"cockpit","owner":{"tenant":{"id":"t100"}}}`))
	if got := app.ContextPath(); got != "cockpit" {
		t.Errorf("Application.ContextPath() = %q, want %q", got, "cockpit")
	}
	if got := app.OwnerTenantID(); got != "t100" {
		t.Errorf("Application.OwnerTenantID() = %q, want %q", got, "t100")
	}
	if got := NewApplication([]byte(`{"id":"1"}`)).OwnerTenantID(); got != "" {
		t.Errorf("Application.OwnerTenantID() without owner = %q, want empty", got)
	}

	if got := NewTenant([]byte(`{"id":"t100","company":"Example Ltd"}`)).Company(); got != "Example Ltd" {
		t.Errorf("Tenant.Company() = %q, want %q", got, "Example Ltd")
	}

	if got := NewTrustedCertificate([]byte(`{"name":"ca","issuer":"CN=Root CA"}`)).Issuer(); got != "CN=Root CA" {
		t.Errorf("TrustedCertificate.Issuer() = %q, want %q", got, "CN=Root CA")
	}

	sub := NewNotification2Subscription([]byte(`{"id":"5","source":{"id":"123","name":"device01"}}`))
	if got := sub.SourceName(); got != "device01" {
		t.Errorf("Notification2Subscription.SourceName() = %q, want %q", got, "device01")
	}
	if got := NewNotification2Subscription([]byte(`{"id":"5","context":"tenant"}`)).SourceName(); got != "" {
		t.Errorf("Notification2Subscription.SourceName() without source = %q, want empty", got)
	}
}
