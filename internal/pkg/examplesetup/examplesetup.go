// Package examplesetup builds a client wired to an in-process fake Cumulocity
// server, so runnable Example functions run offline and deterministically.
package examplesetup

import (
	"io"
	"log/slog"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
)

// New starts a fake server and returns a client for it. Call the returned
// function to stop the server.
func New() (*api.Client, func()) {
	fs, err := fakeserver.NewServer("")
	if err != nil {
		panic(err)
	}
	client := api.NewClient(api.ClientOptions{
		BaseURL: fs.URL(),
		Logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		Auth: authentication.AuthOptions{
			Tenant: "t12345", Username: "admin", Password: "admin-pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
	return client, fs.Server.Close
}
