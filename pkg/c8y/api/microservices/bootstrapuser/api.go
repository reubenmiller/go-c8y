package bootstrapuser

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	apiApplicationBootstrapUser = "/application/applications/{id}/bootstrapUser"
)

var paramID = "id"

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service struct{ svc httpcore.Service }

func NewService(common *httpcore.Service) *Service {
	return &Service{svc: *common}
}

// Get an microservice bootstrap user
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.BootstrapUser] {
	return httpcore.Execute(ctx, s.getB(ID), jsonmodels.NewBootstrapUser)
}

func (s *Service) getB(ID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, ID).
		SetURL(apiApplicationBootstrapUser)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
