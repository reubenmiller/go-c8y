// Package wsdial holds the small contract shared by the websocket clients of
// the SDK (realtime, notification2 and remote access) and [api.Client].
//
// The websocket clients do not build their own TLS, proxy, authentication or
// dry-run handling. They take a [Connector], which api.Client implements, so a
// connection behaves like an HTTP call made with the same client and context.
package wsdial

import (
	"context"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
)

// Connector opens websocket connections the way an SDK client sends HTTP
// requests: with the client's TLS and proxy settings, its current
// authorization and its request middleware.
//
// *api.Client implements Connector.
type Connector interface {
	// DialWebsocket opens the websocket at rawURL (ws, wss, http or https).
	// header holds additional request headers. On success the returned
	// response's Request is the upgrade request that was sent.
	//
	// When ctx has dry run enabled nothing is dialed: it returns a
	// *DryRunError carrying the upgrade request (errors.Is(err, ErrDryRun)).
	DialWebsocket(ctx context.Context, rawURL string, header http.Header) (*websocket.Conn, *http.Response, error)
}

// ErrDryRun is matched by errors.Is for the error returned instead of a
// connection when the context has dry run enabled.
var ErrDryRun = errors.New("websocket not connected: dry run")

// DryRunError is returned by [Connector.DialWebsocket] under dry run. It
// carries the upgrade request that would have been sent, with the same headers
// as a real connection.
type DryRunError struct {
	// Request is the websocket upgrade request (GET with the Upgrade headers).
	// Its Authorization header is NOT redacted, like the request of an HTTP
	// dry run; redact it before printing it (the SDK's own output does).
	Request *http.Request
}

func (e *DryRunError) Error() string { return ErrDryRun.Error() }

// Is reports whether target is [ErrDryRun].
func (e *DryRunError) Is(target error) bool { return target == ErrDryRun }

// DryRunRequest returns the upgrade request carried by err, or nil when err is
// not a dry-run error.
func DryRunRequest(err error) *http.Request {
	var d *DryRunError
	if errors.As(err, &d) {
		return d.Request
	}
	return nil
}

// IsDryRun reports whether err stands for a connection which was not made
// because of dry run.
func IsDryRun(err error) bool {
	return errors.Is(err, ErrDryRun)
}
