// Package features provides feature toggles of the platform, and the per
// tenant overrides of a toggle (client.Features.Tenants).
//
// Reach it from a client with client.Features.
//
// Example, with a client and ctx in scope:
//
//	result := client.Features.List(ctx)
//	for feature, err := range result.Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(feature.Get("key").String(), feature.Get("active").Bool())
//	}
package features
