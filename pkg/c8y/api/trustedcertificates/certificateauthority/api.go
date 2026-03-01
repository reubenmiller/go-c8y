package certificateauthority

import (
	"context"
	"net/http"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiCertificateAuthority = "/certificate-authority"
var ApiCertificateAuthorityRenew = "/certificate-authority/renew"
var ApiTrustedCertificates = "/tenant/tenants/{tenantID}/trusted-certificates"
var ApiTrustedCertificate = "/tenant/tenants/{tenantID}/trusted-certificates/{fingerprint}"

const ParamTenant = "tenantID"
const ParamFingerprint = "fingerprint"

const ResultProperty = "certificates"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service inventory api to interact with the current tenant
// type Service core.Service
type Service struct {
	core.Service
}

type CreateOptions struct{}

// ListOptions trusted certificates filter options
type GetOptions struct {
	TenantID string
}

// Get the certificate authority
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.TrustedCertificate] {
	result := core.ExecuteCollection(ctx, s.getB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewTrustedCertificate)
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

func (s *Service) getB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamTenant, opt.TenantID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetQueryParam("certificateAuthority", "true").
		SetURL(ApiTrustedCertificates)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Create certificate authority
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.TrustedCertificate] {
	return core.Execute(ctx, s.createB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) createB(opt CreateOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiCertificateAuthority)
	return core.NewTryRequest(s.Client, req)
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
	TenantID string
}

// Delete removes the tenant's CA certificate from the trusted certificates repository.
// It first fetches the CA certificate to obtain its fingerprint, then deletes it via
// the trusted-certificates API.
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	// Step 1: get the CA certificate so we can extract the fingerprint
	cert := s.Get(ctx, GetOptions{TenantID: opt.TenantID})
	if cert.IsError() {
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
	return core.ExecuteNoContent(ctx, s.deleteB(opt, cert.Data.Fingerprint()))
}

func (s *Service) deleteB(opt DeleteOptions, fingerprint string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTenant, opt.TenantID).
		SetPathParam(ParamFingerprint, fingerprint).
		SetURL(ApiTrustedCertificate)
	return core.NewTryRequest(s.Client, req)
}

type RenewOptions struct{}

// Renew certificate authority
func (s *Service) Renew(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.TrustedCertificate] {
	return core.Execute(ctx, s.renewB(opt), jsonmodels.NewTrustedCertificate)
}

func (s *Service) renewB(opt CreateOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiCertificateAuthorityRenew)
	return core.NewTryRequest(s.Client, req)
}
