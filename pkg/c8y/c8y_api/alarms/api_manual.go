package alarms

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// Count the total number of active alarms on your tenant
func (s *Service) Count(ctx context.Context, opt ListOptions) (int64, error) {
	count, err := core.ExecuteResultText(ctx, s.CountB(opt))
	if count == "" {
		return 0, err
	}
	return gjson.Parse(count).Int(), err
}

func (s *Service) CountB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeTextPlain).
		SetURL(ApiAlarmsCount)
	return core.NewTryRequest(s.Client, req)
}
