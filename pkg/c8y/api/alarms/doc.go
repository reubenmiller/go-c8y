// Package alarms covers alarms: create, query, update and delete alarms, and
// follow them in realtime with SubscribeStream. Create takes a typed device
// reference (an id, name:, ext: or query: reference) and Upsert de-duplicates
// on source, type and status.
//
// Reach it from a client with client.Alarms.
//
// Example, with a client and ctx in scope:
//
//	created := client.Alarms.Create(ctx, alarms.CreateOptions{
//		Source:   managedobjects.ByName("my-device"),
//		Type:     "example_HighTemperature",
//		Text:     "Temperature is too high",
//		Severity: model.AlarmSeverityMajor,
//	})
//	fmt.Println(created.Data.ID(), created.Err)
package alarms
