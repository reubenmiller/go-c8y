// Package testclient builds API clients for tests.
package testclient

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"

// Must is api.NewClient for tests: it panics when the options are invalid.
func Must(opts api.ClientOptions) *api.Client {
	client, err := api.NewClient(opts)
	if err != nil {
		panic(err)
	}
	return client
}
