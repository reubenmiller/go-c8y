package eventbinaries

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"

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
func (s *Service) Get(ctx context.Context, eventID string) (*model.EventBinary, error) {
	return core.ExecuteResultOnly[model.EventBinary](ctx, s.GetB(eventID))
}

func (s *Service) GetB(eventID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, eventID).
		SetURL(ApiEventBinary)
	return core.NewTryRequest(s.Client, req)
}

func selectFirstNonEmptyValue(contentType ...string) string {
	for _, v := range contentType {
		if v != "" {
			return v
		}
	}
	return ""
}

func NewMultiPartFileFields(opt UploadFileOptions) []*resty.MultipartField {
	obj := make(map[string]any)
	obj["name"] = selectFirstNonEmptyValue(opt.Name, filepath.Base(opt.Filename))
	obj["type"] = selectFirstNonEmptyValue(opt.ContentType, mime.TypeByExtension(filepath.Ext(opt.Filename)), "application/octet-stream")
	objB, _ := json.Marshal(obj)

	fields := make([]*resty.MultipartField, 0, 2)
	fields = append(fields, &resty.MultipartField{
		Name:        "object",
		Reader:      bytes.NewReader(objB),
		ContentType: "application/json",
	})
	fields = append(fields, &resty.MultipartField{
		Name:        "file",
		FileName:    obj["name"].(string),
		FilePath:    opt.Filename,
		ContentType: obj["type"].(string),
	})

	return fields
}

type UploadFileOptions struct {
	Filename    string
	Name        string
	ContentType string
}

// Upload a binary file to an event
func (s *Service) Create(ctx context.Context, eventID string, opt UploadFileOptions) (*model.EventBinary, error) {
	return core.ExecuteResultOnly[model.EventBinary](ctx, s.CreateB(eventID, opt))
}

func (s *Service) CreateB(eventID string, opt UploadFileOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetMultipartFields(NewMultiPartFileFields(opt)...).
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
	eventBinary, resp, err := core.Execute[model.EventBinary](ctx, s.CreateB(eventID, opt))
	if err == nil && resp.IsSuccess() {
		return eventBinary, err
	}
	if resp.StatusCode() != http.StatusConflict {
		return eventBinary, err
	}

	slog.Debug("Replacing existing event binary", "eventID", eventID)
	file, err := os.Open(opt.Filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return s.Update(ctx, eventID, file)
}

// Update an event binary
func (s *Service) Update(ctx context.Context, eventID string, contents io.Reader) (*model.EventBinary, error) {
	return core.ExecuteResultOnly[model.EventBinary](ctx, s.UpdateB(eventID, contents))
}
func (s *Service) UpdateB(eventID string, contents io.Reader) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, eventID).
		SetBody(contents).
		SetContentType("application/octet-stream").
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
