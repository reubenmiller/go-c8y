// Package measurements covers measurements: create, query, export and delete
// time series values, list the series of a device, and follow measurements in
// realtime with SubscribeStream.
//
// Reach it from a client with client.Measurements.
//
// Example, with a client and ctx in scope:
//
//	created := client.Measurements.Create(ctx, measurements.CreateOptions{
//		Source: managedobjects.ByName("my-device"),
//		Type:   "c8y_TemperatureMeasurement",
//		Fragments: []model.Fragment{
//			model.Frag("c8y_Temperature", map[string]any{"T": map[string]any{"value": 21.5, "unit": "C"}}),
//		},
//	})
//	fmt.Println(created.Err)
package measurements
