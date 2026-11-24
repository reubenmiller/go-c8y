package certificateauthority

import (
	"context"
	"net/http"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiCertificateAuthority = "/certificate-authority"
var ApiCertificateAuthorityRenew = "/certificate-authority/renew"
var ApiTrustedCertificates = "/tenant/tenants/{tenantId}/trusted-certificates"
var ApiTrustedCertificate = "/tenant/tenants/{tenantId}/trusted-certificates/{fingerprint}"

const ParamFingerprint = "fingerprint"

const ResultProperty = "certificates"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc: *s,
	}
}

// Service inventory api to interact with the current tenant
// type Service httpcore.Service
type Service struct {
	svc httpcore.Service
}

// CreateOptions are the settings of a new certificate authority. Settings
// that are not set keep the platform's defaults.
type CreateOptions struct {
	TenantID string `url:"-"`

	// Status (ENABLED or DISABLED) is set after the certificate authority is
	// created, when it differs. Empty keeps the status.
	Status string `url:"-"`

	// AutoRegistrationEnabled is set after the certificate authority is
	// created, when it differs. nil keeps the setting.
	AutoRegistrationEnabled *bool `url:"-"`
}

// ListOptions trusted certificates filter options
type GetOptions struct {
	TenantID string `url:"-"`
}

// Get the certificate authority
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.TrustedCertificate] {
	result := httpcore.ExecuteCollection(ctx, s.getB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTrustedCertificate)
	if result.Err != nil {
		return result
	}
	// Get first item from iterator
	for doc := range result.Data.Iter() {
		cert := jsonmodels.NewTrustedCertificate(doc.Bytes())
		return op.Result[jsonmodels.TrustedCertificate]{
			Data:       cert,
			Err:        nil,
			Status:     result.Status,
			HTTPStatus: result.HTTPStatus,
			Attempts:   result.Attempts,
			Duration:   result.Duration,
			RequestID:  result.RequestID,
			Meta:       result.Meta,
		}
	}
	// No certificate found - simulate a not found error
	return op.Result[jsonmodels.TrustedCertificate]{
		Err:        core.Error{Code: 404},
		Status:     result.Status,
		HTTPStatus: 404,
		Attempts:   result.Attempts,
		Duration:   result.Duration,
		RequestID:  result.RequestID,
		Meta:       result.Meta,
	}
}

func (s *Service) getB(opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetQueryParam("certificateAuthority", "true").
		SetURL(ApiTrustedCertificates)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
}

// Create certificate authority.
// A 409 response (already exists) is treated as a duplicate (StatusDuplicate,
// Idempotent: true) and returns the existing certificate authority. Status
// and AutoRegistrationEnabled are then applied with Update when they differ
// (not in a dry run, where nothing was created).
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.TrustedCertificate] {
	result := httpcore.Execute(ctx, s.createB(opt), jsonmodels.NewTrustedCertificate).IgnoreConflict()
	if result.Err != nil || ctxhelpers.IsDryRun(ctx) || ctxhelpers.IsDeferredExecution(ctx) {
		return result
	}
	if result.Status == op.StatusDuplicate {
		existing := s.Get(ctx, GetOptions{TenantID: opt.TenantID})
		if existing.Err != nil {
			return existing
		}
		existing.Status = op.StatusDuplicate
		existing.Idempotent = true
		result = existing
	}
	changes := map[string]any{}
	if opt.Status != "" && !strings.EqualFold(opt.Status, result.Data.Status()) {
		changes["status"] = opt.Status
	}
	if opt.AutoRegistrationEnabled != nil && *opt.AutoRegistrationEnabled != result.Data.Get("autoRegistrationEnabled").Bool() {
		changes["autoRegistrationEnabled"] = *opt.AutoRegistrationEnabled
	}
	if len(changes) == 0 {
		return result
	}
	return httpcore.Execute(ctx, s.updateB(opt.TenantID, result.Data.Fingerprint(), changes), jsonmodels.NewTrustedCertificate)
}

// UpdateOptions options to update the tenant CA certificate
type UpdateOptions struct {
	TenantID string `url:"-"`
}

// Update changes the tenant's CA certificate, e.g.
// {"status": "DISABLED", "autoRegistrationEnabled": false}. The certificate
// authority is looked up first to get its fingerprint; that lookup is always
// sent, also in a dry run, so the update request is complete.
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	cert := s.Get(ctxhelpers.ResolutionContext(ctx), GetOptions(opt))
	if cert.Err != nil {
		return cert
	}
	return httpcore.Execute(ctx, s.updateB(opt.TenantID, cert.Data.Fingerprint(), body), jsonmodels.NewTrustedCertificate)
}

func (s *Service) updateB(tenantID, fingerprint string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetPathParam(ParamFingerprint, fingerprint).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiTrustedCertificate)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) createB(opt CreateOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiCertificateAuthority)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Get or create a certificate authority
// If the certificate does not exist it will be created
func (s *Service) GetOrCreate(ctx context.Context, opt GetOptions) op.Result[jsonmodels.TrustedCertificate] {
	result := s.Create(ctx, CreateOptions{})
	if result.IsError() && !core.ErrHasStatus(result.Err, http.StatusConflict) {
		return result
	}
	return s.Get(ctx, opt)
}

// DeleteOptions options to delete the tenant CA certificate
type DeleteOptions struct {
	TenantID string `url:"-"`
}

// Delete removes the tenant's CA certificate from the trusted certificates repository.
// It first fetches the CA certificate to obtain its fingerprint, then deletes it via
// the trusted-certificates API.
// If no CA certificate exists the call is a no-op and returns successfully.
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	// Step 1: get the CA certificate so we can extract the fingerprint. The
	// lookup is always sent, also in a dry run, so the delete is complete.
	cert := s.Get(ctxhelpers.ResolutionContext(ctx), GetOptions(opt))
	if cert.IsError() {
		// No CA present — intent is already fulfilled, but surface the fact as metadata
		if cert.HTTPStatus == 404 {
			return op.Result[core.NoContent]{
				Status: op.StatusSkipped,
				Meta:   map[string]any{"message": "no certificate authority found; nothing to delete"},
			}
		}
		return op.Result[core.NoContent]{
			Err:        cert.Err,
			Status:     cert.Status,
			HTTPStatus: cert.HTTPStatus,
			Attempts:   cert.Attempts,
			Duration:   cert.Duration,
			RequestID:  cert.RequestID,
			Meta:       cert.Meta,
		}
	}
	// Step 2: delete by fingerprint
	return httpcore.ExecuteNoContent(ctx, s.deleteB(opt, cert.Data.Fingerprint())).IgnoreNotFound()
}

func (s *Service) deleteB(opt DeleteOptions, fingerprint string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetPathParam(ParamFingerprint, fingerprint).
		SetURL(ApiTrustedCertificate)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type RenewOptions struct{}

// Renew certificate authority
func (s *Service) Renew(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.Execute(ctx, s.renewB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) renewB(opt CreateOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiCertificateAuthorityRenew)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
