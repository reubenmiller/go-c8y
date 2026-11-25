// Package retentionrules covers retention rules: how long the platform keeps
// alarms, events, measurements and operations.
//
// Reach it from a client with client.RetentionRules.
//
// Example, with a client and ctx in scope:
//
//	for rule, err := range client.RetentionRules.ListAll(ctx, retentionrules.ListOptions{}).Items() {
//		if err != nil {
//			break
//		}
//		fmt.Println(rule.Get("dataType").String(), rule.Get("maximumAge").Int())
//	}
package retentionrules
