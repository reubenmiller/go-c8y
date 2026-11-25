// Package applications covers applications of the platform: hosted web
// applications, external ones and microservices. List, create, upload
// binaries, subscribe tenants and resolve an application by id or name.
// Versions are under client.Applications.Versions.
//
// Reach it from a client with client.Applications.
//
// Example, with a client and ctx in scope:
//
//	result := client.Applications.ListByName(ctx, applications.ListByNameOptions{Name: "cockpit"})
//	for app, err := range result.Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(app.ID(), app.Name())
//	}
package applications
