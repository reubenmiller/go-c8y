// Package repository provides the software, firmware and configuration
// repository of the tenant. The repository service only groups the three sub-
// services.
//
// Reach it from a client with client.Repository (Software, Firmware,
// Configuration).
//
// Example, with a client and ctx in scope:
//
//	for sw, err := range client.Repository.Software.ListAll(ctx, softwareitems.ListOptions{}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(sw.Get("name").String())
//	}
package repository
