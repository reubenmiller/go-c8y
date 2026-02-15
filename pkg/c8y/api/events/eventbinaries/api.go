package eventbinaries

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"resty.dev/v3"
)

var ApiEventBinary = "/event/events/{id}/binaries"

var ParamId = "id"

// Service provides api to get/set/delete audit entries in Cumulocity
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
}

// Get an event binary
func (s *Service) Get(ctx context.Context, eventID string) op.Result[core.BinaryResponse] {
	return core.ExecuteBinary(ctx, s.getB(eventID))
}

func (s *Service) getB(eventID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, eventID).
		SetURL(ApiEventBinary)
	return core.NewTryRequest(s.Client, req)
}

type UploadFileOptions = core.UploadFileOptions

// Upload a binary file to an event
func (s *Service) Create(ctx context.Context, eventID string, opt UploadFileOptions) op.Result[jsonmodels.EventBinary] {
	return core.Execute(ctx, s.createB(eventID, opt), jsonmodels.NewEventBinary)
}

func (s *Service) createB(eventID string, opt UploadFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetMultipartFields(core.NewMultiPartFileFields(opt)...).
		SetPathParam(ParamId, eventID).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiEventBinary)
	return core.NewTryRequest(s.Client, req)
}

// Create or replace an event binary
//
// It will first try to add a new binary to an event, but if a binary already exists (as indicated by a http 409 response)
// then it is replaced by sending a PUT.
// If the existing binary is only replaced, then some meta information might not be updated
func (s *Service) Upsert(ctx context.Context, eventID string, opt UploadFileOptions) op.Result[jsonmodels.EventBinary] {
	// slog.Debug("Replacing existing event binary", "eventID", eventID)
	result := core.Execute(ctx, s.createB(eventID, opt), jsonmodels.NewEventBinary)
	if result.Err == nil {
		return result
	}

	if !core.ErrHasStatus(result.Err, 409) {
		return result
	}
	return core.Execute(ctx, s.updateB(eventID, opt), jsonmodels.NewEventBinary)
}

// Update an event binary
func (s *Service) Update(ctx context.Context, eventID string, opt UploadFileOptions) op.Result[jsonmodels.EventBinary] {
	return core.Execute(ctx, s.updateB(eventID, opt), jsonmodels.NewEventBinary)
}

func (s *Service) updateB(eventID string, opt UploadFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, eventID).
		SetBody(opt.GetReader()).
		SetContentType(types.MimeTypeApplicationOctetStream).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiEventBinary)
	return core.NewTryRequest(s.Client, req)
}

// Delete an event binary
func (s *Service) Delete(ctx context.Context, eventID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(eventID))
}

func (s *Service) deleteB(eventID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, eventID).
		SetURL(ApiEventBinary)
	return core.NewTryRequest(s.Client, req)
}
