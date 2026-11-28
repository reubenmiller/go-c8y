package alarms

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// Count the total number of active alarms on your tenant
func (s *Service) Count(ctx context.Context, opt CountOptions) op.Result[int64] {
	// Resolve Source if it contains a resolver scheme
	if opt.Source != "" && s.DeviceResolver != nil {
		resolutionCtx := ctxhelpers.ResolutionContext(ctx)

		resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
		if err != nil {
			return op.Failed[int64](err, true)
		}
		opt.Source = managedobjects.DeviceRef(resolvedID)
	}

	return httpcore.Execute(ctx, s.countB(opt), func(b []byte) int64 {
		return gjson.ParseBytes(b).Int()
	})
}

func (s *Service) countB(opt CountOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeTextPlain).
		SetURL(apiAlarmsCount)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
