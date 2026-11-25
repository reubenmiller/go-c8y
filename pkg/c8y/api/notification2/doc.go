// Package notification2 provides notification 2.0: manage subscriptions and
// tokens, and consume notifications as a typed stream. SubscribeStream creates
// the token and connects the websocket. Under api.WithDryRun the upgrade
// request is reported instead of connecting.
//
// Reach it from a client with client.Notification2.
//
// Example, with a client and ctx in scope:
//
//	created := client.Notification2.Create(ctx, notification2.CreateOptions{
//		Context:      "mo",
//		Subscription: "alarms",
//		Source:       map[string]any{"id": "12345"},
//		SubscriptionFilter: notification2.SubscriptionFilter{Apis: []string{"alarms"}},
//	})
//	fmt.Println(created.Err)
package notification2
