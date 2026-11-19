## 1. Subscription watching

- [ ] 1.1 Add `WatchOptions` (`Interval`, `OnTenantAdded`, `OnTenantRemoved`, `OnCredentialsChanged`, `OnError`)
- [ ] 1.2 Implement `WatchSubscriptions` with the diff rules and deterministic callback order
- [ ] 1.3 Refresh immediately when a service-user request returns `401`
- [ ] 1.4 Fake server support for changing the subscription list between polls
- [ ] 1.5 Offline tests for the watch scenarios, including cancellation

## 2. Per-tenant work

- [ ] 2.1 Change `ForEachTenant` to `ForEachTenant(ctx, fn, opts ...pipeline.Options)` on top of `pipeline.ForEach`, defaulting to one worker and `MaxErrors: 1`
- [ ] 2.2 Tests for the default and concurrent scenarios

## 3. Settings

- [ ] 3.1 Implement `Settings(ctx)` with a per-tenant cache and `SettingsTTL`
- [ ] 3.2 Evict on `OnTenantRemoved`
- [ ] 3.3 Offline tests for override and caching

## 4. Notifications

- [ ] 4.1 Implement `SubscribeNotifications` on `pkg/c8y/notification2` (subscription and token as service user, reconnect, token refresh, cancellation)
- [ ] 4.2 Fix the reader goroutine in `SubscribeToNotifications`; add a deprecation notice
- [ ] 4.3 Tests with a fake WebSocket server and a goroutine leak check

## 5. Isolation and application key

- [ ] 5.1 Read `C8Y_MICROSERVICE_ISOLATION` in `Bootstrap`; implement `PER_TENANT` mode
- [ ] 5.2 Add the `X-Cumulocity-Application-Key` middleware when `APPLICATION_KEY` is set
- [ ] 5.3 Tests for both scenarios

## 6. Documentation

- [ ] 6.1 Update `docs/MICROSERVICE_TENANT_CONTEXT.md` with a lifecycle section and the Java SDK mapping
- [ ] 6.2 Update `examples/microservice/multi-tenant` and `examples/microservice/notification2-microservice` to use `WatchSubscriptions` and `SubscribeNotifications`
