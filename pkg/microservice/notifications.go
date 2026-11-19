package microservice

import (
	"context"
	"log/slog"
	"time"

	notification2api "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/notification2"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/notification2"
)

// NotificationMessage is one Notification 2.0 message.
type NotificationMessage = notification2.StreamData[jsondoc.JSONDoc]

// Reconnect backoff for SubscribeNotifications. Variables so tests can
// shorten them.
var (
	notificationMinBackoff = time.Second
	notificationMaxBackoff = 30 * time.Second
)

// SubscribeNotifications receives Notification 2.0 messages for tenant until
// ctx is cancelled. The subscription and token are created as the tenant's
// service user, so opts.Options (subscription name, filters) apply to that
// tenant. After a connection loss it reconnects with a fresh token, with
// backoff. When ctx is cancelled it unsubscribes, stops the client, and
// returns nil.
//
// Typical use with WatchSubscriptions: start it in OnTenantAdded with a
// per-tenant context, and cancel that context in OnTenantRemoved.
//
// A message is acknowledged after handler returns nil; when handler returns an
// error the message is not acknowledged, so the platform delivers it again.
// Return quickly or hand work to another goroutine.
func (m *Microservice) SubscribeNotifications(ctx context.Context, tenant string, opts notification2api.ClientOptions, handler func(ctx context.Context, msg NotificationMessage) error) error {
	backoff := notificationMinBackoff
	wait := func() bool {
		select {
		case <-ctx.Done():
			return false
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, notificationMaxBackoff)
		return true
	}

	for ctx.Err() == nil {
		tenantCtx := m.WithServiceUser(ctx, tenant)
		res := m.Client.Notification2.SubscribeStream(tenantCtx, opts)
		if res.Err != nil {
			slog.Warn("Notification subscription failed, retrying", "tenant", tenant, "err", res.Err, "retryIn", backoff)
			if !wait() {
				return nil
			}
			continue
		}
		backoff = notificationMinBackoff

		stream := res.Data
		for msg, err := range stream.Items() {
			if err != nil {
				break
			}
			if herr := handler(ctx, msg); herr != nil {
				// Not acknowledged: the platform redelivers it.
				slog.Warn("Notification handler failed", "tenant", tenant, "err", herr)
				continue
			}
			if aerr := stream.AcknowledgeMessage(msg.Identifier); aerr != nil {
				slog.Warn("Failed to acknowledge notification", "tenant", tenant, "err", aerr)
			}
		}
		_ = stream.Close()
		if ctx.Err() != nil {
			return nil
		}
		slog.Info("Notification stream ended, reconnecting", "tenant", tenant)
		if !wait() {
			return nil
		}
	}
	return nil
}
