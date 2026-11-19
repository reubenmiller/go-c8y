## ADDED Requirements

### Requirement: Public API uses no HTTP library types
The SDK SHALL NOT expose resty types in any exported function signature, struct field, method, or promoted field of an embedded type, with the single exception of `(*api.Client).Resty()`.

#### Scenario: Exported API scan
- **WHEN** `go doc -all` is run for every package under `pkg/`
- **THEN** the only reference to a `resty.` type is the `Resty()` method on `api.Client`

#### Scenario: Escape hatch is documented
- **WHEN** a user reads the godoc for `(*api.Client).Resty()`
- **THEN** it states that the returned client is not covered by the v2 compatibility guarantee

### Requirement: RoundTripper middleware
The client SHALL accept middleware of type `api.Middleware`, defined as `func(next http.RoundTripper) http.RoundTripper`, and SHALL apply it to every HTTP request the client sends.

#### Scenario: Middleware adds a header
- **WHEN** a client is created with a middleware that sets the header `X-Test: 1`
- **THEN** every request received by the fake server carries `X-Test: 1`

#### Scenario: Middleware order
- **WHEN** two middlewares A and B are registered in that order
- **THEN** A sees the request before B, and B sees the response before A

### Requirement: Built-in middleware is internal
The middleware that `NewClient` installs itself, and the unused middleware helpers, SHALL NOT be exported. Package `api` SHALL export exactly two middleware constructors for users: `CountRequests(counter *int64) Middleware` and `CountByMethodAndPath(stats *StatsMap) Middleware`.

#### Scenario: Internal middleware is not callable
- **WHEN** user code calls `api.MiddlewareRemoveEmptyTenantID()`
- **THEN** the code does not compile

#### Scenario: Request counter
- **WHEN** a client is created with `api.CountRequests(&n)` and sends 3 requests that succeed on the first attempt
- **THEN** `n` is 3

#### Scenario: Counter counts retry attempts
- **WHEN** a request fails once with `503` and succeeds on the retry
- **THEN** `api.CountRequests` counts 2

#### Scenario: Dry-run requests are not counted
- **WHEN** a request is sent with `api.WithDryRun(ctx, true)`
- **THEN** neither counter changes

#### Scenario: Statistics by method and path
- **WHEN** a client with `api.CountByMethodAndPath(stats)` sends 2 `GET /inventory/managedObjects` requests
- **THEN** `stats` reports 2 for `GET` and `/inventory/managedObjects`

### Requirement: Request and response hooks use standard library types
`RequestOptions.PrepareRequest` SHALL have type `func(*http.Request) error`, and `RequestOptions.OnResponse` SHALL have type `func(*http.Response) error`. The response body passed to `OnResponse` SHALL be readable without consuming the body returned to the caller.

#### Scenario: PrepareRequest changes the request
- **WHEN** `PrepareRequest` adds the query parameter `withParents=true`
- **THEN** the request sent to the server includes `withParents=true`

#### Scenario: PrepareRequest error aborts the request
- **WHEN** `PrepareRequest` returns an error
- **THEN** no request is sent and the returned error wraps the hook's error

#### Scenario: OnResponse reads the body
- **WHEN** `OnResponse` reads the whole response body
- **THEN** the caller still receives the full response body

### Requirement: Raw responses are SDK-owned
Functions that return a raw HTTP response (`core.ExecuteResponseOnly`, `api.RequestResult`) SHALL return SDK-owned copies of the response body and the sent request, so their contents do not change after the call returns.

#### Scenario: Request body stays stable after later requests
- **WHEN** a caller keeps the request returned by `core.ExecuteResponseOnly` and the client then sends 100 more requests with different bodies
- **THEN** reading the kept request's body returns the original body

### Requirement: Service types do not expose core internals
Exported service types SHALL NOT embed `core.Service`; the HTTP client and service internals SHALL be reachable only through unexported fields.

#### Scenario: No promoted client field
- **WHEN** user code tries to access `client.Alarms.Client`
- **THEN** the code does not compile
