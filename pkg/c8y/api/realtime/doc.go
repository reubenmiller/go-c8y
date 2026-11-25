// Package realtime is the Cumulocity realtime (Bayeux) client and the typed
// streams the services build on it (client.Alarms.SubscribeStream and so on).
//
// Use it through api.Client.Realtime(): the client then connects with the
// settings of the API client (TLS, proxy, current authorization, request
// middleware) and respects dry run:
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//	defer cancel()
//	result := client.Realtime().SubscribeStream(ctx, realtime.Alarms("*"))
//
// ConnectContext aborts connecting when its context is cancelled. Under
// api.WithDryRun nothing is connected: the upgrade request goes to the dry-run
// handler (or is logged), and the subscribe methods return a skipped result
// with the channels in Meta[DryRunMetaChannels] and a stream which ends
// immediately. See docs/API_DESIGN.md, "Dry run and websockets".
package realtime
