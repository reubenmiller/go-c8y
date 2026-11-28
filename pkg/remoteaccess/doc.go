// Package remoteaccess proxies local TCP connections, a unix socket or stdio to
// a device through the Cumulocity cloud remote access feature.
//
// The client connects with the settings of the api.Client it is created with:
// TLS, proxy, the current authorization (a renewed token is used for every
// new connection), request middleware and the context options of the context
// given to DialContext, ListenServeContext and ServeContext. Cancelling that
// context aborts connecting, and ServeContext stops serving.
//
// Under api.WithDryRun no websocket is connected: the upgrade request goes to the
// dry-run handler (or is logged), Client.DryRunRequest returns it,
// and ListenServeContext and ServeContext return nil.
package remoteaccess
