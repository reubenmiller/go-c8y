package applications

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// DownloadBinary downloads one binary (for example a microservice image or a
// web application archive) of an application. Read the content from the
// result's Reader and close it when done.
func (s *Service) DownloadBinary(ctx context.Context, id string, binaryID string) op.Result[core.BinaryResponse] {
	return httpcore.ExecuteBinary(ctx, s.downloadBinaryB(id, binaryID))
}

func (s *Service) downloadBinaryB(id string, binaryID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", "application/octet-stream").
		SetPathParam(paramID, id).
		SetPathParam("binaryId", binaryID).
		SetURL(apiApplicationBinary)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
