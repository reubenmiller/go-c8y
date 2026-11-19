package registration

import (
	"context"
	"errors"
	"log/slog"
	"time"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devicegroups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/password"
	"resty.dev/v3"
)

var ApiDeviceRequests = "/devicecontrol/newDeviceRequests"
var ApiDeviceRequest = "/devicecontrol/newDeviceRequests/{id}"
var ApiBulkDeviceRequests = "/devicecontrol/bulkNewDeviceRequests"

var ApiDeviceCredentials = "/devicecontrol/deviceCredentials"

var ParamID = "id"

const ResultProperty = "newDeviceRequests"

// Service provides api to get/set/delete device requests in Cumulocity
type Service struct {
	svc    httpcore.Service
	groups *devicegroups.Service
}

func NewService(common *httpcore.Service) *Service {
	return &Service{svc: *common, groups: devicegroups.NewService(common)}
}

// ListOptions to list the device requests
type ListOptions struct {
	pagination.PaginationOptions
}

// DeviceRequestIterator provides iteration over device registration requests
type DeviceRequestIterator = pagination.Iterator[jsonmodels.DeviceRequest]

// List returns all device requests
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.DeviceRequest] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewDeviceRequest)
}

// ListAll returns an iterator over all device requests, following pagination.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *DeviceRequestIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.DeviceRequest] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewDeviceRequest,
	)
}

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiDeviceRequests)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
}

// Get retrieves a device request
func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.DeviceRequest] {
	return httpcore.Execute(ctx, s.getB(id), jsonmodels.NewDeviceRequest)
}

func (s *Service) getB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, id).
		SetURL(ApiDeviceRequest)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type CreateOptions struct {
	// External ID of the device
	ID string `json:"id,omitempty"`

	// ID of the group to which the device will be assigned
	GroupID string `json:"groupId,omitempty"`

	// Type of the device
	Type string `json:"type,omitempty"`

	// When creating a new device enrollment request this field is treated as device's one time password (OTP)
	EnrollmentToken string `json:"enrollmentToken,omitempty"`
}

// Create creates a new device request
func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.DeviceRequest] {
	return httpcore.Execute(ctx, s.createB(opt), jsonmodels.NewDeviceRequest)
}

// CreateRaw creates a new device request from a raw body (e.g. the CLI's
// --data/--template document), bypassing the typed CreateOptions. A
// device-group reference at "groupId" that is not an id (e.g. "My Group") is
// resolved to the group id first.
func (s *Service) CreateRaw(ctx context.Context, body any) op.Result[jsonmodels.DeviceRequest] {
	body, err := source.ResolveBodyRefs(ctx, body, func(ctx context.Context, ref string) (string, error) {
		return s.groups.ResolveID(ctxhelpers.ResolutionContext(ctx), ref, nil)
	}, "groupId")
	if err != nil {
		return op.Failed[jsonmodels.DeviceRequest](err, false)
	}
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewDeviceRequest)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeNewDeviceRequest).
		SetBody(body).
		SetURL(ApiDeviceRequests)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type UpdateOptions struct {
	// Status of this new device request
	Status string `json:"status,omitempty"`

	// When accepting a device request, the security token is verified against the token submitted
	// by the device when requesting credentials. See Security token policy for details on configuration.
	// See Create device credentials for details on creating token for device registration.
	// securityToken parameter can be added only when submitting ACCEPTED status.
	SecurityToken string `json:"securityToken,omitempty"`

	// When creating a new device enrollment request this field is treated as device's one time password (OTP)
	EnrollmentToken string `json:"enrollmentToken,omitempty"`
}

// Update a specific new device request (by a given ID). You can only update its status
func (s *Service) Update(ctx context.Context, id string, opt UpdateOptions) op.Result[jsonmodels.DeviceRequest] {
	return httpcore.Execute(ctx, s.updateB(id, opt), jsonmodels.NewDeviceRequest)
}

// UpdateRaw updates a device request from a raw body (e.g. the CLI's
// --data/--template document), bypassing the typed UpdateOptions.
func (s *Service) UpdateRaw(ctx context.Context, id string, body any) op.Result[jsonmodels.DeviceRequest] {
	return httpcore.Execute(ctx, s.updateB(id, body), jsonmodels.NewDeviceRequest)
}

func (s *Service) updateB(id string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamID, id).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeNewDeviceRequest).
		SetBody(body).
		SetURL(ApiDeviceRequest)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Delete removes a device request
func (s *Service) Delete(ctx context.Context, id string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(id)).IgnoreNotFound()
}

func (s *Service) deleteB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, id).
		SetURL(ApiDeviceRequest)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

/*
* Device Credentials
 */
type CreateCredentialsOptions struct {
	// External ID of the device
	ID string `json:"id,omitempty"`

	// Security token which is required and verified against during device request acceptance.
	// See Security token policy for more details on configuration. See Update specific new device
	// request status for details on submitting token upon device acceptance.
	SecurityToken string `json:"securityToken,omitempty"`
}

// CreateCredentials requests device credentials for a device that does not yet
// have credentials. It posts to the device-credentials endpoint, which is
// distinct from the new-device-requests endpoint used by Create.
func (s *Service) CreateCredentials(ctx context.Context, opt CreateCredentialsOptions) op.Result[jsonmodels.DeviceCredentials] {
	return httpcore.Execute(ctx, s.createCredentialsB(opt), jsonmodels.NewDeviceCredentials)
}

// CreateCredentialsRaw requests device credentials from a raw body (e.g. the
// CLI's --data/--template document), bypassing the typed CreateCredentialsOptions.
func (s *Service) CreateCredentialsRaw(ctx context.Context, body any) op.Result[jsonmodels.DeviceCredentials] {
	return httpcore.Execute(ctx, s.createCredentialsB(body), jsonmodels.NewDeviceCredentials)
}

func (s *Service) createCredentialsB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeDeviceCredentials).
		SetBody(body).
		SetURL(ApiDeviceCredentials)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// PollNewDeviceRequest continuously polls a device request for a specified id at defined intervals. The func will wait until the device request has been set to ACCEPTED.
// If the device request does not reach the ACCEPTED state in the defined timeout period, then an error will be returned.
func (s *Service) PollNewDeviceRequest(ctx context.Context, id string, interval time.Duration, timeout time.Duration) (<-chan struct{}, <-chan error) {
	ticker := time.NewTicker(interval)
	timeoutTimer := time.NewTimer(timeout)

	done := make(chan struct{})
	err := make(chan error)

	go func() {
		defer func() {
			ticker.Stop()
			timeoutTimer.Stop()
		}()
		for {
			select {
			case <-ctx.Done():
				err <- ctx.Err()
				return

			case <-ticker.C:
				slog.Info("Polling for device request")
				result := s.Get(ctx, id)
				if result.IsError() {
					continue
				}
				if result.Data.Status() == string(types.DeviceRequestStatusPendingAccepted) {
					done <- struct{}{}
				}

			case <-timeoutTimer.C:
				err <- errors.New("timeout waiting for device request to reach ACCEPTED state")
				return
			}
		}
	}()

	return done, err
}

type UploadFileOptions = core.UploadFileOptions

/*
* Bulk Registration
 */
// CreateBulk allows multiple devices to be registered in one request
func (s *Service) CreateBulk(ctx context.Context, opt UploadFileOptions) op.Result[jsonmodels.BulkNewDeviceRequests] {
	return httpcore.Execute(ctx, s.createBulkB(opt), jsonmodels.NewBulkNewDeviceRequests)
}

func (s *Service) createBulkB(opt UploadFileOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetMultipartFields(httpcore.NewMultiPartFileFields(opt)...).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiBulkDeviceRequests)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// GeneratePassword generates a device password with the recommended password length by default
// and uses symbols which are compatible with the Bulk Registration API.
func (s *Service) GeneratePassword(opts ...password.PasswordOption) (string, error) {
	defaults := []password.PasswordOption{
		// enforce min/max that the api supports
		password.WithLengthConstraints(8, 32),

		// use max length
		password.WithLength(32),

		// use all available symbols to increase complexity
		password.WithSymbols(2),
	}
	defaults = append(defaults, opts...)
	return password.NewRandomPassword(defaults...)
}
