// Package api is the Go client for the Cumulocity REST API.
//
// Create a client with [NewClient], or with [NewClientFromEnvironment] which
// reads C8Y_HOST, C8Y_TENANT, C8Y_USER and C8Y_PASSWORD (or C8Y_TOKEN). Both
// return an error, and never panic, when the base URL is empty or invalid. The
// services hang off the client, one field per API: client.Alarms,
// client.Events, client.Measurements, client.Operations,
// client.ManagedObjects, client.Devices, client.Binaries, client.Repository,
// client.Applications, client.Microservices, client.Users, client.Tenants,
// client.Notification2, client.RemoteAccess and more. Related services are
// nested, for example client.Applications.Versions and
// client.Users.Current. The package of each service has its own
// documentation.
//
//	client, err := api.NewClientFromEnvironment(api.ClientOptions{})
//	if err != nil {
//		return err
//	}
//
//	result := client.Alarms.List(ctx, alarms.ListOptions{
//		Severity: []model.AlarmSeverity{model.AlarmSeverityMajor},
//	})
//	if result.Err != nil {
//		return result.Err
//	}
//	for alarm, err := range result.Items() {
//		fmt.Println(alarm.Text())
//	}
//
// # Results
//
// Every call returns an [op.Result] which holds the data, the error (Err),
// the HTTP status, the request and metadata. A failed call does not panic.
// Check Err, or use Result.Unwrap and Result.First.
//
// # Errors
//
// Classify an error with errors.Is(err, [ErrNotFound]) or with [IsNotFound],
// [IsUnauthorized], [IsForbidden], [IsConflict] and [StatusCode].
//
// # Pagination
//
// Collections are read a page at a time with List, or across all pages with
// ListAll, which returns an iterator. See package pagination.
//
// # Context options
//
// The context carries per call behaviour without changing the method
// signatures: [WithDryRun] (build and report the request without sending it),
// [WithDryRunHandler], [WithRequestHook], [WithAuth], tenant selection and
// extra headers. See also [Client.Request] for endpoints without a typed
// method.
//
// # Realtime
//
// Services which support it have a SubscribeStream method (alarms, events,
// measurements, operations, managed objects, Notification 2.0). Package
// realtime has the lower level client.
package api
