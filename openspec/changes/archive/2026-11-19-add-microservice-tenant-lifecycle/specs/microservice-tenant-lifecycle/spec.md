## ADDED Requirements

### Requirement: Subscription watching
`Microservice.WatchSubscriptions(ctx, opts WatchOptions)` SHALL poll the microservice's subscriptions every `opts.Interval` (default 30 seconds) until `ctx` is cancelled. After each poll it SHALL update the cached service users and call `OnTenantAdded` for new tenants, `OnTenantRemoved` for tenants no longer subscribed, and `OnCredentialsChanged` for tenants whose service-user credentials changed.

#### Scenario: Startup
- **WHEN** `WatchSubscriptions` starts and tenants `t1` and `t2` are subscribed
- **THEN** `OnTenantAdded` is called for `t1` and then `t2`

#### Scenario: Tenant unsubscribes
- **WHEN** `t2` is no longer in the subscriptions on the next poll
- **THEN** `OnTenantRemoved` is called for `t2` and `GetServiceUser("t2")` returns not found without a refresh

#### Scenario: Service user recreated
- **WHEN** `t1` is still subscribed but its service-user password changed
- **THEN** `OnCredentialsChanged` is called for `t1` and the cache holds the new password

#### Scenario: Callback error
- **WHEN** `OnTenantAdded` returns an error for `t1`
- **THEN** the error is passed to `OnError`, the cache is still updated, and watching continues

#### Scenario: Cancellation
- **WHEN** `ctx` is cancelled
- **THEN** `WatchSubscriptions` returns within one polling interval and makes no further requests

### Requirement: Concurrent per-tenant work
`ForEachTenant(ctx, fn, opts ...pipeline.Options)` SHALL run `fn` for each subscribed tenant with the tenant's service-user credentials in the context, using `pipeline.ForEach`. Without options it SHALL use one worker and stop at the first error.

#### Scenario: Default behavior
- **WHEN** `ForEachTenant(ctx, fn)` runs for 3 tenants and `fn` fails for the second
- **THEN** `fn` is not called for the third tenant and the error is returned

#### Scenario: Concurrent with error tolerance
- **WHEN** `ForEachTenant(ctx, fn, pipeline.Options{Workers: 3})` runs for 3 tenants and `fn` fails for one
- **THEN** `fn` is called for all 3 tenants

### Requirement: Per-tenant settings
`Microservice.Settings(ctx)` SHALL return the microservice settings as seen by the tenant in `ctx`, using that tenant's service user, cached per tenant for `SettingsTTL` (default 10 minutes). The cache entry SHALL be removed when the tenant is removed by `WatchSubscriptions`.

#### Scenario: Tenant override
- **WHEN** tenant `t1` overrides the editable setting `interval` to `30`, and the owner default is `60`
- **THEN** `Settings(ms.WithServiceUser(ctx, "t1"))` returns `interval = 30`

#### Scenario: Cached
- **WHEN** `Settings` is called twice for `t1` within the TTL
- **THEN** the fake server receives one settings request

### Requirement: Per-tenant Notification 2.0 subscription
`Microservice.SubscribeNotifications(ctx, tenant, opts, handler)` SHALL create the Notification 2.0 subscription and token as the tenant's service user, deliver messages to `handler`, acknowledge each message after `handler` returns nil, reconnect after connection loss, and return when `ctx` is cancelled, leaving no goroutines running.

#### Scenario: Messages delivered
- **WHEN** the fake notification server sends 3 messages for tenant `t1`
- **THEN** `handler` receives the 3 messages in order and the 3 messages are acknowledged

#### Scenario: Reconnect
- **WHEN** the WebSocket connection drops
- **THEN** the helper reconnects with a fresh token and continues delivering messages

#### Scenario: Clean shutdown
- **WHEN** `ctx` is cancelled
- **THEN** the call returns and a goroutine leak check (`goleak` or runtime goroutine count) shows no remaining helper goroutines

### Requirement: Legacy realtime helper does not leak
`SubscribeToNotifications` SHALL stop its reader goroutine when the realtime channel closes, and SHALL be marked deprecated in favor of `SubscribeNotifications`.

#### Scenario: Channel closed
- **WHEN** the realtime client closes the subscription channel
- **THEN** the reader goroutine exits

### Requirement: Isolation modes and application key
When `C8Y_MICROSERVICE_ISOLATION` is `PER_TENANT`, `Bootstrap` SHALL use `C8Y_TENANT`, `C8Y_USER`, and `C8Y_PASSWORD` as the only service user and SHALL NOT call the subscriptions endpoint. When `APPLICATION_KEY` is set, every request SHALL carry `X-Cumulocity-Application-Key` with its value.

#### Scenario: PER_TENANT bootstrap
- **WHEN** the isolation is `PER_TENANT` and `C8Y_TENANT=t5`
- **THEN** `GetServiceUser("")` returns the user for `t5` and no subscriptions request is sent

#### Scenario: Application key header
- **WHEN** `APPLICATION_KEY=abc` is set
- **THEN** requests from the microservice client carry `X-Cumulocity-Application-Key: abc`

### Requirement: Notification 2.0 stream delivers messages
`Notification2.SubscribeStream` SHALL deliver the messages the client receives, and closing the stream SHALL stop the client's goroutines.

#### Scenario: Stream receives a message
- **WHEN** the fake notification server pushes one message after `SubscribeStream` connected
- **THEN** the stream yields that message
