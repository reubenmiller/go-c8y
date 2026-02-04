package alarms

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// Count the total number of active alarms on your tenant
func (s *Service) Count(ctx context.Context, opt ListOptions) op.Result[int64] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctx
		if ctxhelpers.IsDeferredExecution(ctx) {
			resolutionCtx = context.Background()
		}

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[int64](err, true)
		}
		opt.Source = resolvedID
	}

	return core.Execute(ctx, s.CountB(opt), func(b []byte) int64 {
		return gjson.ParseBytes(b).Int()
	})
}

func (s *Service) CountB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeTextPlain).
		SetURL(ApiAlarmsCount)
	return core.NewTryRequest(s.Client, req)
}
