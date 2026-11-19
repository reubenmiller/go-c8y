## Why

`pkg/microservice` can already run code per tenant (`ForEachTenant`, `WithServiceUser`, `TenantContext`), but it cannot react to tenants subscribing or unsubscribing. Service users are loaded at bootstrap and refreshed only when a lookup misses, so unsubscribed tenants stay in the cache. The Java SDK polls subscriptions and fires added/removed events, reads settings per tenant with a cache, and runs per-tenant work concurrently. Without these, per-tenant setup (Notification 2.0 subscriptions, caches) cannot be built reliably. The CometD helper `SubscribeToNotifications` also leaks a goroutine that never exits. Review: docs/V2_STABILIZATION_REVIEW.md §4.3 to §4.6 and §4.8.

## What Changes

- New `Microservice.WatchSubscriptions(ctx, WatchOptions)` that polls `GET /application/currentApplication/subscriptions`, diffs the result, updates the service-user cache, and calls `OnTenantAdded`, `OnTenantRemoved`, and `OnCredentialsChanged`.
- `ForEachTenant` accepts optional `pipeline.Options` (workers, delay, error limits). With no options it keeps today's behavior: one tenant at a time, stop at the first error.
- New per-tenant settings: `Microservice.Settings(ctx)` reads the settings of the tenant in `ctx`, with a cache (default 10 minutes).
- New Notification 2.0 helper per tenant that creates the subscription and token as the tenant's service user, reconnects, and stops when the context is cancelled or the tenant is removed.
- Fix the goroutine leak in `SubscribeToNotifications` and deprecate it in favor of the Notification 2.0 helper.
- Support `PER_TENANT` isolation (`C8Y_TENANT`, `C8Y_USER`, `C8Y_PASSWORD` as the only service user) and send `X-Cumulocity-Application-Key` when `APPLICATION_KEY` is set.

Additive; can land in v2.x. Depends on validate-microservice-callers for the request side.

## Capabilities

### New Capabilities
- `microservice-tenant-lifecycle`: tracking subscribed tenants, running work per tenant, per-tenant settings and notifications, and isolation-mode handling.

### Modified Capabilities
<!-- none: no existing specs -->

## Impact

- `pkg/microservice` (`microservice.go`, `subscriptions.go`, new `watch.go`, `settings.go`, `notifications.go`).
- `pkg/c8y/notification2` (used by the new helper).
- `examples/microservice/multi-tenant`, `examples/microservice/notification2-microservice`, `docs/MICROSERVICE_TENANT_CONTEXT.md`.
