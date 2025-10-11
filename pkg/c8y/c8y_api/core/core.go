package core

import (
	"net/url"

	"github.com/google/go-querystring/query"
	"resty.dev/v3"
)

type Service struct {
	Client *resty.Client
}

type TryRequest struct {
	Request  *resty.Request
	Client   *resty.Client
	Property string
}

func QueryParameters(opt any) url.Values {
	v, _ := query.Values(opt)
	return v
}
