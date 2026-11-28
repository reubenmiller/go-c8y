// Package probe is the smallest package that uses the SDK client. The
// compilecost test counts the functions compiled into it.
package probe

import (
	"net/url"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
)

// BaseURL only reads a field, so everything compiled into this package is
// what the SDK costs any package that refers to api.Client.
func BaseURL(c *api.Client) *url.URL { return c.BaseURL() }
