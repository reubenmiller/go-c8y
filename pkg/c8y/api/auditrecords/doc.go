// Package auditrecords covers audit records: read and create the entries of
// the tenant audit log.
//
// Reach it from a client with client.AuditRecords.
//
// Example, with a client and ctx in scope:
//
//	result := client.AuditRecords.List(ctx, auditrecords.ListOptions{})
//	for record, err := range result.Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(record.Get("type").String(), record.Get("text").String())
//	}
package auditrecords
