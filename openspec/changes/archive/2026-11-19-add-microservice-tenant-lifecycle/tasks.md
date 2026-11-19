## 1. Subscription watching

- [x] 1.1 Add `WatchOptions` (`Interval`, `OnTenantAdded`, `OnTenantRemoved`, `OnCredentialsChanged`, `OnError`)
- [x] 1.2 Implement `WatchSubscriptions` with the diff rules and deterministic callback order
- [x] 1.3 Refresh immediately when a service-user request returns `401`
- [x] 1.4 Test platform with a subscription list that changes between polls (pkg/microservice/lifecycle_test.go)
- [x] 1.5 Offline tests for the watch scenarios, including cancellation

## 2. Per-tenant work

- [x] 2.1 Change `ForEachTenant` to `ForEachTenant(ctx, fn, opts ...pipeline.Options)`: with options it uses `pipeline.ForEach`; without options it keeps the sequential stop-at-first-error loop (`pipeline.ForEach` with `MaxErrors: 1` still dispatched the next tenant)
- [x] 2.2 Tests for the default and concurrent scenarios

## 3. Settings

- [x] 3.1 Implement `Settings(ctx)` with a per-tenant cache and `SettingsTTL`
- [x] 3.2 Evict on `OnTenantRemoved`
- [x] 3.3 Offline tests for override and caching

## 4. Notifications

- [x] 4.1 Implement `SubscribeNotifications` on `pkg/c8y/notification2` (subscription and token as service user, acknowledgements, reconnect, cancellation); fix `SubscribeStream` delivery and `Notification2Client` goroutine leaks found on the way
- [x] 4.2 Fix the reader goroutine in `SubscribeToNotifications`; add a deprecation notice
- [x] 4.3 Tests with a fake WebSocket server and a goroutine leak check

## 5. Isolation and application key

- [x] 5.1 Read `C8Y_MICROSERVICE_ISOLATION` in `Bootstrap`; implement `PER_TENANT` mode
- [x] 5.2 Add the `X-Cumulocity-Application-Key` middleware when `APPLICATION_KEY` is set
- [x] 5.3 Tests for both scenarios

## 6. Documentation

- [x] 6.1 Update `docs/MICROSERVICE_TENANT_CONTEXT.md` with a lifecycle section and the Java SDK mapping
- [x] 6.2 Rewrite `examples/microservice/notification2-microservice` to use `WatchSubscriptions` and `SubscribeNotifications` (the multi-tenant example already shows `RequireRoles`)
