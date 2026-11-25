// Package model covers request side models: typed fragments, enums (alarm
// severity and status, application types) and bulk registration records.
// Fragments are passed to the Create options of the services.
//
// Use it in the options of the services.
//
// Example, with a client and ctx in scope:
//
//	fragments := []model.Fragment{
//		model.Frag("c8y_Restart", map[string]any{}),
//	}
//	fmt.Println(len(fragments), model.AlarmSeverityCritical)
package model
