package c8y

import (
	"context"

	"github.com/tidwall/gjson"
)

// AlarmService provides api to get/set/delete alarms in Cumulocity
type AlarmService service

// AlarmCollectionOptions to use when search for alarms
type AlarmCollectionOptions struct {
	// Source device to filter measurements by
	Source string `url:"source,omitempty"`

	// DateFrom Timestamp `url:"dateFrom,omitempty"`
	DateFrom string `url:"dateFrom,omitempty"`

	DateTo string `url:"dateTo,omitempty"`

	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Status string `url:"status,omitempty"`

	Severity string `url:"severity,omitempty"`

	Resolved bool `url:"resolved,omitempty"`

	WithAssets bool `url:"withAssets,omitempty"`

	WithDevices bool `url:"withDevices,omitempty"`

	PaginationOptions
}

// AlarmObject todo
type AlarmObject struct {
	ID                  string    `json:"id,omitempty"`
	Source              Source    `json:"source,omitempty"`
	Type                string    `json:"type,omitempty"`
	Time                Timestamp `json:"time,omitempty"`
	CreationTime        Timestamp `json:"creationTime,omitempty"`
	FirstOccurrenceTime Timestamp `json:"firstOccurrenceTime,omitempty"`
	Text                string    `json:"text,omitempty"`
	Status              string    `json:"status,omitempty"`
	Severity            string    `json:"severity,omitempty"`
	Count               uint64    `json:"count,omitempty"`
	Self                string    `json:"self,omitempty"`

	// Allow access to custom fields
	Item gjson.Result `json:"-"`
}

// AlarmCollection todo
type AlarmCollection struct {
	*BaseResponse

	Alarms []AlarmObject `json:"alarms"`

	Items []gjson.Result `json:"-"`
}

// GetAlarmCollection returns a list of alarms using the specified search options
func (s *AlarmService) GetAlarmCollection(ctx context.Context, opt *AlarmCollectionOptions) (*AlarmCollection, *Response, error) {
	data := new(AlarmCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "alarm/alarms",
		Query:        opt,
		ResponseData: data,
	})
	data.Items = resp.JSON.Get("alarms").Array()
	return data, resp, err
}

// CreateAlarm creates a new alarm object
func (s *AlarmService) CreateAlarm(ctx context.Context, body interface{}) (*AlarmObject, *Response, error) {
	data := new(AlarmObject)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "alarm/alarms",
		Body:         body,
		ResponseData: data,
	})
	data.Item = gjson.Parse(resp.JSON.Raw)
	return data, resp, err
}
