package certificaterevocationlist

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiCertificateRevocationList = "/tenant/trusted-certificates/settings/crl"

const ResultProperty = "crls"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service api to interact with the trusted certificates revocation list
// type Service core.Service
type Service struct {
	core.Service
}

// ListOptions trusted certificates filter options
type GetOptions struct{}

// List the certificate revocation list
func (s *Service) List(ctx context.Context, opt GetOptions) op.Result[core.BinaryResponse] {
	return core.ExecuteBinaryResponse(ctx, s.GetB(opt))
}

func (s *Service) GetB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypePkixCRL).
		SetURL(ApiCertificateRevocationList)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Add certificates to the revocation list
func (s *Service) Add(ctx context.Context, body any) error {
	return core.ExecuteNoResult(ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiCertificateRevocationList)
	return core.NewTryRequest(s.Client, req)
}

type AddFileFileOptions = core.UploadFileOptions

// Add CRL list from file
func (s *Service) AddFile(ctx context.Context, opt AddFileFileOptions) error {
	return core.ExecuteNoResult(ctx, s.AddFileB(opt))
}

func (s *Service) AddFileB(opt AddFileFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetMultipartFields(core.NewMultiPartFileFields(opt)...).
		SetURL(ApiCertificateRevocationList)
	return core.NewTryRequest(s.Client, req)
}
