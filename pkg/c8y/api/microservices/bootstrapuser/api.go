package bootstrapuser

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	ApiApplicationBootstrapUser = "/application/applications/{id}/bootstrapUser"
)

var ParamId = "id"

// Service to manage binaries
// Managed objects can perform operations to store, retrieve and delete binaries. One binary can store only one file. Together with the binary, a managed object is created which acts as a metadata information for the binary.
type Service struct{ core.Service }

func NewService(common *core.Service) *Service {
	return &Service{Service: *common}
}

// Get an microservice bootstrap user
func (s *Service) Get(ctx context.Context, ID string) op.Result[jsonmodels.BootstrapUser] {
	return core.Execute(ctx, s.getB(ID), jsonmodels.NewBootstrapUser)
}

func (s *Service) getB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, ID).
		SetURL(ApiApplicationBootstrapUser)
	return core.NewTryRequest(s.Client, req)
}
