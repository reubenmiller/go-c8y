package certificateauthority

import (
	"context"
	"net/http"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiCertificateAuthority = "/certificate-authority"
var ApiCertificateAuthorityRenew = "/certificate-authority/renew"
var ApiTrustedCertificates = "/tenant/tenants/{tenantID}/trusted-certificates"

const ParamTenant = "tenantID"

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
func (s *Service) Get(ctx context.Context, opt GetOptions) (*model.TrustedCertificate, error) {
	cert, err := core.ExecuteResultOnly[model.TrustedCertificateCollection](ctx, s.GetB(opt))
	if err != nil {
		return nil, err
	}
	if len(cert.Certificates) == 0 {
		// simulate a not found error
		return nil, core.Error{Code: 404}
	}
	return &cert.Certificates[0], nil
}

func (s *Service) GetB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamTenant, opt.TenantID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetQueryParam("certificateAuthority", "true").
		SetURL(ApiTrustedCertificates)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Create certificate authority
func (s *Service) Create(ctx context.Context, opt CreateOptions) (*model.TrustedCertificate, error) {
	return core.ExecuteResultOnly[model.TrustedCertificate](ctx, s.CreateB(opt))
}

func (s *Service) CreateB(opt CreateOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiCertificateAuthority)
	return core.NewTryRequest(s.Client, req)
}

// Get or create a certificate authority
// If the certificate does not exist it will be created
func (s *Service) GetOrCreate(ctx context.Context, opt GetOptions) (*model.TrustedCertificate, error) {
	result, err := s.Create(ctx, CreateOptions{})
	if err == nil {
		return result, nil
	}
	if !core.ErrHasStatus(err, http.StatusConflict) {
		return result, err
	}
	return s.Get(ctx, opt)
}

type RenewOptions struct{}

// Renew certificate authority
func (s *Service) Renew(ctx context.Context, opt CreateOptions) (*model.TrustedCertificate, error) {
	return core.ExecuteResultOnly[model.TrustedCertificate](ctx, s.RenewB(opt))
}

func (s *Service) RenewB(opt CreateOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiCertificateAuthorityRenew)
	return core.NewTryRequest(s.Client, req)
}
