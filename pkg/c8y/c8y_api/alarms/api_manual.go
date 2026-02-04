package alarms

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// Count the total number of active alarms on your tenant
func (s *Service) Count(ctx context.Context, opt ListOptions) op.Result[int64] {
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
