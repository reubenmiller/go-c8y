package core

import (
	"net/url"

	"github.com/google/go-querystring/query"
	"resty.dev/v3"
)

type Service struct {
	Client *resty.Client
}

func QueryParameters(opt any) url.Values {
	v, _ := query.Values(opt)
	return v
}
