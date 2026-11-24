package microservices

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/applications/currentapplication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// ListServiceUsers lists the service users of a microservice: one per tenant
// subscribed to it, with the credentials the microservice uses there.
//
// id is a microservice reference (an id, or "name:..." and the other forms
// ResolveID accepts). The microservice is resolved and its bootstrap user
// fetched first; both are reads and are sent also in a dry run, so the dry
// run shows the subscriptions request with the bootstrap user's credentials.
func (s *Service) ListServiceUsers(ctx context.Context, id string) op.Result[jsonmodels.ApplicationUser] {
	lookupCtx := ctxhelpers.ResolutionContext(ctx)
	appID, err := s.ResolveID(lookupCtx, id, nil)
	if err != nil {
		return op.Failed[jsonmodels.ApplicationUser](err, false)
	}
	user := s.BootstrapUser.Get(lookupCtx, appID)
	if user.Err != nil {
		return op.Result[jsonmodels.ApplicationUser]{
			Err:        user.Err,
			Status:     op.StatusFailed,
			HTTPStatus: user.HTTPStatus,
			Retryable:  user.Retryable,
			Request:    user.Request,
		}
	}
	ctx = authentication.WithAuth(ctx, authentication.AuthOptions{
		Tenant:   user.Data.Tenant(),
		Username: user.Data.Username(),
		Password: user.Data.Password(),
	})
	return httpcore.ExecuteCollection(ctx, s.listServiceUsersB(), currentapplication.SubscriptionsResultProperty, "", jsonmodels.NewApplicationUser)
}

func (s *Service) listServiceUsersB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(currentapplication.ApiApplicationSubscriptions)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
