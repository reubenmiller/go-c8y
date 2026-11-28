package certificaterevocationlist

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiCertificateRevocationList = "/tenant/trusted-certificates/settings/crl"

const resultProperty = "crls"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc: *s,
	}
}

// Service api to interact with the trusted certificates revocation list
// type Service httpcore.Service
type Service struct {
	svc httpcore.Service
}

// ListOptions trusted certificates filter options
type GetOptions struct{}

// List the certificate revocation list
func (s *Service) List(ctx context.Context, opt GetOptions) op.Result[core.BinaryResponse] {
	return httpcore.ExecuteBinary(ctx, s.getB(opt))
}

func (s *Service) getB(opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypePkixCRL).
		SetURL(apiCertificateRevocationList)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// Add certificates to the revocation list
func (s *Service) Add(ctx context.Context, body any) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.createB(body))
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiCertificateRevocationList)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type AddFileFileOptions = core.UploadFileOptions

// Add CRL list from file
func (s *Service) AddFile(ctx context.Context, opt AddFileFileOptions) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.addFileB(opt))
}

func (s *Service) addFileB(opt AddFileFileOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetMultipartFields(httpcore.NewMultiPartFileFields(opt)...).
		SetURL(apiCertificateRevocationList)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
