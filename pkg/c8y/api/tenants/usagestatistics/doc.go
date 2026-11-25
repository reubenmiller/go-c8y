// Package usagestatistics covers usage statistics of a tenant: requests,
// storage and device counts per day, and the summary over a date range.
//
// Reach it from a client with client.Tenants.UsageStatistics.
//
// Example, with a client and ctx in scope:
//
//	result := client.Tenants.UsageStatistics.List(ctx, usagestatistics.ListOptions{})
//	fmt.Println(result.Err)
package usagestatistics
