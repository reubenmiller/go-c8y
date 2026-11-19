package trustedcertificates

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// ApiTrustedCertificatesBulk uploads several trusted certificates at once.
var ApiTrustedCertificatesBulk = "/tenant/tenants/{tenantId}/trusted-certificates/bulk"

// CreateBulk uploads several trusted certificates in one request. body is
// {"certificates": [...]}, each entry like the body of Create.
func (s *Service) CreateBulk(ctx context.Context, opt CreateOptions, body any) op.Result[jsonmodels.TrustedCertificate] {
	return httpcore.ExecuteCollection(ctx, s.createBulkB(opt, body), "certificates", "", jsonmodels.NewTrustedCertificate)
}

func (s *Service) createBulkB(opt CreateOptions, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetPathParam(core.PathParamTenantID, opt.TenantID).
		SetURL(ApiTrustedCertificatesBulk)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
