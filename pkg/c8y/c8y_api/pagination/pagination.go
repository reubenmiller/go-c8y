package pagination

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

type PageSize int

const DefaultPageSize PageSize = 5

func (p PageSize) IsZero() bool {
	return p < PageSize(1) || p == PageSize(DefaultPageSize)
}

// PaginationOptions is the cumulocity pagination options
type PaginationOptions struct {
	// Pagesize of results to return in one request
	PageSize int `url:"pageSize,omitempty,omitzero" json:"pageSize,omitempty,omitzero"`

	// Include total pages included in the pagination at the given page size
	WithTotalPages bool `url:"withTotalPages,omitempty"`

	// Include count of elements in the statistics response. Only supported >= 10.13
	WithTotalElements bool `url:"withTotalElements,omitempty"`

	// Defines the slice of data to be returned, starting with 1. By default, the first page is returned.
	CurrentPage int `url:"currentPage,omitempty,omitzero"`
}

func (o PaginationOptions) IsZero() bool {
	return o.PageSize <= 0 || o.PageSize == 5 // Define zero as any non-positive value
}

// Set the current page to return
func (o PaginationOptions) SetCurrentPage(v int) *PaginationOptions {
	o.CurrentPage = v
	return &o
}
func (o PaginationOptions) SetPageSize(v int) *PaginationOptions {
	o.PageSize = v
	return &o
}

type PagerOptions struct {
	MaxPages    int64 `url:"-"`
	PageSize    int64 `url:"pageSize"`
	CurrentPage int64 `url:"currentPage"`
}

func IncludeAll() PagerOptions {
	return PagerOptions{}
}

func (p *PagerOptions) GetPageSize() int64 {
	if p.PageSize <= 0 {
		return 2000
	}
	return p.PageSize
}

// NewPaginationOptions returns a pagination options object with a specified pagesize and WithTotalPages set to false
func NewPaginationOptions(pageSize int) *PaginationOptions {
	return &PaginationOptions{
		PageSize: pageSize,
	}
}

func ForEach[A any](r *core.TryRequest, pagerOpts PagerOptions, out chan<- A) error {
	var nextReq *resty.Request
	nextReq = r.Request
	nextReq.SetQueryParam("pageSize", fmt.Sprintf("%d", pagerOpts.GetPageSize()))
	if pagerOpts.CurrentPage > 0 {
		nextReq.SetQueryParam("currentPage", fmt.Sprintf("%d", pagerOpts.CurrentPage))
	}
	pageCount := int64(0)

	for {
		resp, err := nextReq.Send()
		if err != nil {
			slog.Error("Request failed", "err", err)
			break
		}
		pageCount++

		body := gjson.Parse(resp.String())
		slog.Debug("Response", "size", resp.Size(), "duration", resp.Duration())

		items := body.Get(r.Property)

		if !items.Exists() || !items.IsArray() {
			// nothing to iterate over
			slog.Error("Stopping as results isn't an array")
			break
		}

		if len(items.Array()) == 0 {
			slog.Info("Stopping pagination as results array is empty")
			break
		}

		items.ForEach(func(key, value gjson.Result) bool {
			data := new(A)
			err := json.Unmarshal([]byte(value.Raw), &data)
			if err != nil {
				slog.Warn("Could not decode message", "err", err)
				return true
			}
			out <- *data
			return true
		})

		next := body.Get("next")
		if !next.Exists() || next.String() == "" {
			slog.Info("next url is empty")
			break
		}

		if pagerOpts.MaxPages > 0 && pageCount >= pagerOpts.MaxPages {
			slog.Info("max pages reached", "total", pageCount)
			break
		}

		// prepare next request

		// TODO: Make the url parsing more robust to use the external url rather the the txxx url
		nextReq = r.Client.R().WithContext(r.Request.Context()).SetMethod(nextReq.Method).SetURL(trimHost(next.String()))
		slog.Info("Next request", "url", nextReq.URL)

	}

	close(out)
	return nil
}

func ForEachJSON(r *core.TryRequest, pagerOpts PagerOptions, out chan<- gjson.Result) error {
	var nextReq *resty.Request
	nextReq = r.Request
	nextReq.SetQueryParam("pageSize", fmt.Sprintf("%d", pagerOpts.GetPageSize()))
	if pagerOpts.CurrentPage > 0 {
		nextReq.SetQueryParam("currentPage", fmt.Sprintf("%d", pagerOpts.CurrentPage))
	}
	pageCount := int64(0)

	for {
		resp, err := nextReq.Send()
		if err != nil {
			slog.Error("Request failed", "err", err)
			break
		}
		pageCount++

		body := gjson.Parse(resp.String())
		slog.Debug("Response", "size", resp.Size(), "duration", resp.Duration())

		items := body.Get(r.Property)

		if !items.Exists() || !items.IsArray() {
			// nothing to iterate over
			slog.Error("Stopping as results isn't an array")
			break
		}

		if len(items.Array()) == 0 {
			slog.Info("Stopping pagination as results array is empty")
			break
		}

		items.ForEach(func(key, value gjson.Result) bool {
			out <- value
			return true
		})

		next := body.Get("next")
		if !next.Exists() || next.String() == "" {
			slog.Info("next url is empty")
			break
		}

		if pagerOpts.MaxPages > 0 && pageCount >= pagerOpts.MaxPages {
			slog.Info("max pages reached", "total", pageCount)
			break
		}

		// prepare next request

		// TODO: Make the url parsing more robust to use the external url rather the the txxx url
		nextReq = r.Client.R().WithContext(r.Request.Context()).SetMethod(nextReq.Method).SetURL(trimHost(next.String()))
		slog.Info("Next request", "url", nextReq.URL)

	}

	close(out)
	return nil
}

func trimHost(v string) string {
	i := strings.Index(v, "://") + 3
	for i < len(v) {
		if v[i] == '/' {
			break
		}
		i++
	}
	return v[i:]
}

type Pager[A any] struct {
	Input  *core.TryRequest
	Output chan A
}

func NewPager[A any](r *core.TryRequest, channelSize ...int) *Pager[A] {
	size := 0
	if len(channelSize) > 0 {
		size = channelSize[0]
	}
	return &Pager[A]{
		Input:  r,
		Output: make(chan A, size),
	}
}

func (s *Pager[A]) IncludeAll() error {
	return ForEach(s.Input, PagerOptions{}, s.Output)
}

func (s *Pager[A]) Pages(paging PagerOptions) error {
	return ForEach(s.Input, paging, s.Output)
}

type PagerJSON struct {
	Input  *core.TryRequest
	Output chan gjson.Result
}

func NewPagerJSON(r *core.TryRequest, channelSize ...int) *PagerJSON {
	size := 0
	if len(channelSize) > 0 {
		size = channelSize[0]
	}
	return &PagerJSON{
		Input:  r,
		Output: make(chan gjson.Result, size),
	}
}

func (s *PagerJSON) IncludeAll(ctx context.Context) error {
	return ForEach(s.Input, PagerOptions{}, s.Output)
}

func (s *PagerJSON) Page(ctx context.Context, paging PagerOptions) error {
	return ForEach(s.Input, paging, s.Output)
}
