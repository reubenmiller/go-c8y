package certificaterevocationlist

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
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
	return core.ExecuteBinary(ctx, s.getB(opt))
}

func (s *Service) getB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypePkixCRL).
		SetURL(ApiCertificateRevocationList)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Add certificates to the revocation list
func (s *Service) Add(ctx context.Context, body any) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.createB(body))
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiCertificateRevocationList)
	return core.NewTryRequest(s.Client, req)
}

type AddFileFileOptions = core.UploadFileOptions

// Add CRL list from file
func (s *Service) AddFile(ctx context.Context, opt AddFileFileOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.addFileB(opt))
}

func (s *Service) addFileB(opt AddFileFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetMultipartFields(core.NewMultiPartFileFields(opt)...).
		SetURL(ApiCertificateRevocationList)
	return core.NewTryRequest(s.Client, req)
}
