## Context

- `Bootstrap` calls `RefreshServiceUsers` once. `GetServiceUser` refreshes on a cache miss. Nothing removes unsubscribed tenants or notices recreated service users.
- `ForEachTenant` loops over a snapshot of `ServiceUsers` sequentially and returns the first error.
- `pipeline.ForEach` already provides workers, delay, `MaxErrors`, `ErrorThreshold`, and progress callbacks. It continues after item errors unless `MaxErrors` is set.
- `Configuration` is one viper instance for the whole process; per-tenant overrides of editable settings are not visible.
- `SubscribeToNotifications` starts a reader goroutine with `close(ch)` in its own `defer`, so it never exits and ignores cancellation.

## Goals / Non-Goals

**Goals:**
- React to subscription changes without restarting the microservice.
- Concurrency for per-tenant work using the SDK's existing pipeline options.
- Per-tenant settings and notifications with clean shutdown.

**Non-Goals:**
- A Spring-like dependency-injection container or tenant-scoped beans. Go code holds per-tenant state in maps keyed by tenant, set up in `OnTenantAdded`.
- Leader election across replicas (each replica watches independently; document it).

## Decisions

**Polling, not push.** The platform offers no subscription-change notification to the microservice, so `WatchSubscriptions` polls (default every 30 seconds; configurable). The Java SDK also polls, controlled by `C8Y.bootstrap.delay`; its default value was not confirmed from the published docs, so check it before finalizing the default here. It runs the callbacks for all tenants found on the first poll, like the Java SDK's startup events.

**Diff by tenant and credential.** Added: tenant not in the previous set. Removed: tenant missing from the new set. Credentials changed: same tenant, different user name or password (unsubscribe/subscribe cycle). Callbacks run sequentially per poll, in a deterministic order (sorted by tenant ID), and a callback error is logged and passed to `OnError` without stopping the watch.

**`ForEachTenant` keeps its default.** New signature `ForEachTenant(ctx, fn, opts ...pipeline.Options)`. Without options it runs with one worker and `MaxErrors: 1`, matching today's behavior. A variadic parameter keeps existing callers compiling. rill stays internal; nothing new is exposed.

**Settings per tenant with a cache.** `Settings(ctx)` reads `GET /application/currentApplication/settings` using the service user of the tenant in `ctx` (bootstrap user when none), caches per tenant for `SettingsTTL` (default 10 minutes, the Java SDK's value), and evicts the entry in `OnTenantRemoved`. Keys starting with `credentials.` are returned as received; godoc explains that the platform decrypts them only for the owning service user.

**Notification 2.0 helper owns its lifecycle.** `SubscribeNotifications(ctx, tenant, Notification2Options, handler)` creates the subscription and token as the tenant's service user, connects with `pkg/c8y/notification2`, reconnects with backoff, refreshes the token before expiry, and returns when `ctx` is cancelled. Used with `WatchSubscriptions`, users start it in `OnTenantAdded` and cancel it in `OnTenantRemoved`.

**Isolation mode from the environment.** With `C8Y_MICROSERVICE_ISOLATION=PER_TENANT`, `Bootstrap` uses `C8Y_TENANT`/`C8Y_USER`/`C8Y_PASSWORD` as the single service user and does not call the subscriptions endpoint. `X-Cumulocity-Application-Key` is added by client middleware when `APPLICATION_KEY` is set.

## Risks / Trade-offs

- [Polling load with many replicas] → One request per replica per interval; interval is configurable.
- [Callbacks slow down the watch loop] → Document that callbacks should start goroutines for long work; the watch logs callbacks that run longer than the interval.
- [Credentials rotate between polls] → `GetServiceUser` still refreshes on a miss, and a `401` with service-user credentials triggers an immediate refresh.

## Open Questions

- Does the platform's bootstrap user see subscription removals immediately, or with a delay? Affects the documented reaction time.
- Should `PER_TENANT` mode skip `WatchSubscriptions` entirely, or poll the single tenant's service user for credential rotation?
