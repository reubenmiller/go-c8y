// Package notification2 is the websocket consumer of Cumulocity notification
// 2.0 subscriptions. The service api.Client.Notification2 creates tokens and
// clients; its SubscribeStream returns a stream of messages.
//
// A client created by the service connects with the settings of the API client
// (TLS, proxy, request middleware, extra headers) through SetConnector, and
// honors dry run: ConnectContext returns an error matching wsdial.ErrDryRun
// without dialing, and DryRunStream turns it into a skipped result with a
// stream which ends immediately. The token in the URL authenticates the
// connection, so no Authorization header is sent and the token is redacted in
// dry-run output.
package notification2
