package dataprep

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// DiagnosticsService downloads Data Preparation diagnostics.
type DiagnosticsService struct {
	svc httpcore.Service
}

// Download streams a zip with the tenant's rules (including smart function
// code), its Data Preparation configuration, and the Streaming Analytics
// diagnostics. Close result.Data when done.
//
// The server answers 429 while an archive is already being prepared; the
// client retries those with the Retry-After delay, like other requests.
func (s *DiagnosticsService) Download(ctx context.Context) op.Result[core.BinaryResponse] {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", MimeTypeZip).
		SetURL(ApiDiagnostics)
	return httpcore.ExecuteBinary(ctx, httpcore.NewTryRequest(s.svc.Client, req))
}
