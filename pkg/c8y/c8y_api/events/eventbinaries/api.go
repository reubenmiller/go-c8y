package eventbinaries

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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
func (s *Service) Get(ctx context.Context, eventID string) (*core.BinaryResponse, error) {
	return core.ExecuteBinaryResponse(ctx, s.GetB(eventID))
}

func (s *Service) GetB(eventID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, eventID).
		SetURL(ApiEventBinary)
	return core.NewTryRequest(s.Client, req)
}

type UploadFileOptions = core.UploadFileOptions

// Upload a binary file to an event
func (s *Service) Create(ctx context.Context, eventID string, opt UploadFileOptions) (*model.EventBinary, error) {
	return core.ExecuteResultOnly[model.EventBinary](ctx, s.CreateB(eventID, opt))
}

func (s *Service) CreateB(eventID string, opt UploadFileOptions) *core.TryRequest {
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
func (s *Service) Upsert(ctx context.Context, eventID string, opt UploadFileOptions) (*model.EventBinary, error) {
	// slog.Debug("Replacing existing event binary", "eventID", eventID)
	return core.ExecuteUpsertResultOnly[model.EventBinary](ctx, s.CreateB(eventID, opt), s.UpdateB(eventID, opt))
}

// Update an event binary
func (s *Service) Update(ctx context.Context, eventID string, opt UploadFileOptions) (*model.EventBinary, error) {
	return core.ExecuteResultOnly[model.EventBinary](ctx, s.UpdateB(eventID, opt))
}

func (s *Service) UpdateB(eventID string, opt UploadFileOptions) *core.TryRequest {
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
func (s *Service) Delete(ctx context.Context, eventID string) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(eventID))
}

func (s *Service) DeleteB(eventID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, eventID).
		SetURL(ApiEventBinary)
	return core.NewTryRequest(s.Client, req)
}
