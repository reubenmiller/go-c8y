# Cumulocity Go SDK v2 - API Design

## Design Philosophy

The v2 API is designed with the following core principles:

1. **Type Safety with Flexibility** - Use Go generics for type-safe operations while supporting flexible JSON handling
2. **Explicit Success & Failure** - Rich result types that distinguish between different outcomes (success, created, skipped, duplicate, error)
3. **Context-Driven Behavior** - Control execution behavior (dry run, deferred execution, logging) through context
4. **Developer Experience** - Intuitive APIs with sensible defaults, chainable options, and clear error messages
5. **Extensibility** - Allow customization of requests, responses, and behaviors without breaking core patterns
6. **AI-Friendly** - Clear patterns and conventions that AI assistants can understand and apply correctly

## Architecture Overview

### Core Components

```
Client
├── Services (Operations, Measurements, Alarms, etc.)
│   ├── Core Service (HTTP client, common utilities)
│   ├── Service-Specific Methods (List, Get, Create, Update, Delete)
│   └── Resolvers (Device name → ID, External ID lookup, Query-based)
├── Result<T> (Typed results with metadata)
├── JSONModels (Flexible JSON document wrappers)
└── Context Options (Dry run, deferred execution, etc.)
```

### Client Structure

The `Client` is the entry point to the SDK, providing access to all service operations:

```go
client := api.NewClient(baseURL, username, password)

// Service access
result := client.Operations.List(ctx, operations.ListOptions{
    DeviceID: "12345",
    Status:   "PENDING",
})

result := client.ManagedObjects.Get(ctx, "device-id", managedobjects.GetOptions{})
result := client.Measurements.Create(ctx, measurements.CreateOptions{...})
```

**Key services:**
- `Operations` - Device operations (commands)
- `ManagedObjects` - Inventory/device management
- `Measurements` - Time-series measurement data
- `Events` - Event records
- `Alarms` - Alarm management
- `Applications`, `Binaries`, `Users`, `Tenants`, etc.

## Result Type

The `Result<T>` type is central to v2's design, providing rich metadata about every operation:

```go
type Result[T any] struct {
    Data       T                  // The actual result data
    Status     Status             // Operation outcome (OK, Created, Failed, etc.)
    HTTPStatus int                // HTTP status code
    Err        error              // Error if operation failed
    
    // Operation characteristics
    Retryable  bool               // Whether operation can be retried
    Idempotent bool               // Whether operation is idempotent
    Meta       map[string]any     // Additional metadata (pagination, etc.)
    
    // Tracking
    Attempts   int                // Number of retry attempts
    Duration   time.Duration      // Total operation duration
    RequestID  string             // Correlation ID for debugging
    Timestamp  time.Time          // When operation completed
    
    // Request inspection (see: REQUEST_INSPECTION.md)
    Request    *http.Request      // HTTP request that was (or would be) sent
    Response   []byte             // Raw response body (full un-plucked envelope for collections; empty otherwise)
    
    // Deferred execution (see: DEFERRED_EXECUTION.md)
    executor   func(context.Context) Result[T]
}
```

### Result Status Values

The `Status` field provides semantic information about the operation outcome:

- **`StatusOK`** - Successfully retrieved existing resource
- **`StatusCreated`** - Successfully created new resource (HTTP 201)
- **`StatusUpdated`** - Successfully updated existing resource
- **`StatusNoContent`** - Successful operation with no response body (HTTP 204)
- **`StatusSkipped`** - Operation skipped (e.g., no-op update, already in desired state)
- **`StatusDuplicate`** - Resource already exists (HTTP 409 conflict)
- **`StatusNoMatch`** - No matching resources found (not an error)
- **`StatusFailed`** - Operation failed with error

### Result Usage Patterns

```go
// Basic success/error handling
result := client.Operations.Get(ctx, "12345")
if result.Err != nil {
    return result.Err
}
operation := result.Data

// Status-based handling
switch result.Status {
case op.StatusOK:
    fmt.Println("Found:", result.Data.ID())
case op.StatusNoMatch:
    fmt.Println("Device not found")
case op.StatusFailed:
    return result.Err
}

// Unwrap pattern (Go standard)
data, err := result.Unwrap()
if err != nil {
    return err
}

// Functional composition (generic methods, Go 1.27+)
ids := result.Map(func(o jsonmodels.Operation) string {
    return o.ID()
})

// Decode into your own types
type MyDevice struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}
device, err := client.ManagedObjects.Get(ctx, id, managedobjects.GetOptions{}).Decode[MyDevice]()
for d, err := range client.ManagedObjects.ListAll(ctx, opts).As[MyDevice]() { /* ... */ }
level, err := mo.GetAs[int]("c8y_Battery.level")
pos, err := mo.Fragment[model.Position]()
```

Generic methods exist only on value types (`op.Result`, `pagination.Iterator`,
`jsondoc.JSONDoc` and every `jsonmodels` type). Service methods stay
non-generic so services can be wrapped in your own interfaces for mocking.

## JSON Models

The SDK provides flexible JSON handling through the `jsondoc` package, allowing both structured and unstructured access:

### JSONDoc Facade Pattern

```go
type Operation struct {
    jsondoc.Facade  // Embed for flexible JSON access
}

func NewOperation(b []byte) Operation {
    return Operation{Facade: jsondoc.NewFacade(b)}
}

// Strongly-typed accessors
func (o Operation) ID() string {
    return o.GetString("id")
}

func (o Operation) DeviceID() string {
    return o.GetString("deviceId")
}

func (o Operation) Status() string {
    return o.GetString("status")
}

// Raw JSON access
func (o Operation) Bytes() []byte {
    return o.Facade.Bytes()
}

// Custom field access
func (o Operation) GetCustomFragment(path string) any {
    return o.Get(path)
}
```

### Usage Examples

```go
// Structured access
result := client.Operations.Get(ctx, "12345")
op := result.Data
fmt.Println("ID:", op.ID())
fmt.Println("Status:", op.Status())

// Unmarshal to custom struct
type CustomOperation struct {
    ID          string `json:"id"`
    Description string `json:"description"`
    MyFragment  struct {
        Value int `json:"value"`
    } `json:"my_Fragment"`
}

var custom CustomOperation
err := json.Unmarshal(op.Bytes(), &custom)

// Direct custom struct creation (bypassing JSONDoc)
result := client.Operations.Create(ctx, CustomOperation{
    Description: "My operation",
    MyFragment: struct{Value int}{Value: 42},
})
```

## Context-Based Execution Control

The v2 API uses Go's `context.Context` to control execution behavior without polluting method signatures:

### Dry Run Mode

Execute operations without sending HTTP requests, useful for testing and debugging:

```go
// Enable dry run
ctx := api.WithDryRun(context.Background(), true)

// Operation is prepared but not executed
result := client.ManagedObjects.Delete(ctx, "12345", managedobjects.DeleteOptions{})

// Inspect what would be sent
if result.Request != nil {
    fmt.Println("Method:", result.Request.Method)
    fmt.Println("URL:", result.Request.URL)
    fmt.Println("Headers:", result.Request.Header)
}
```

**Security:** Sensitive headers are automatically redacted in dry run logs. See [REQUEST_INSPECTION.md](./REQUEST_INSPECTION.md) for details.

#### Dry run and websockets

Realtime, notification 2.0 and remote access connect over a websocket. Under dry run nothing is dialed. The websocket upgrade request is reported through the same path as an HTTP request: the handler set with `api.WithDryRunHandler`, or a `DRY RUN` log line with `Authorization` redacted (and the `token` query parameter of notification 2.0).

```go
ctx := api.WithDryRun(context.Background(), true)
ctx = api.WithDryRunHandler(ctx, func(req *http.Request) {
    fmt.Println(req.Method, req.URL, req.Header.Get("Upgrade"))
})

result := client.Alarms.SubscribeStream(ctx, "12345")
// result.Err == nil, result.Status == op.StatusSkipped
// result.Request       the upgrade request: GET https://tenant/cep/realtime, Upgrade: websocket
// result.Meta["channels"]  []string{"/alarms/12345"}
// result.Data          a stream which ends immediately, without an error
```

The same applies to the other `SubscribeStream` and `Subscribe` methods, `realtime.Client.SubscribeStream` and `Notification2.SubscribeStream` (`Meta["consumer"]` and `Meta["url"]`, token redacted). Lower level code gets an error for which `errors.Is(err, wsdial.ErrDryRun)` is true from `Connect`/`ConnectContext`/`DialWebsocket`; `wsdial.DryRunRequest(err)` is the request. For remote access, `RemoteAccessClient.DialContext` returns that error, `ListenServeContext` and `ServeContext` return `nil`, and `RemoteAccessClient.DryRunRequest()` returns the request.

As for HTTP, the request returned to the caller still has the real `Authorization` header: redact it before you print it.

#### Websocket connection settings

`Client.WebsocketDialer()`, `Client.WebsocketHeader(ctx)` and `Client.DialWebsocket(ctx, url, header)` give a websocket the settings of an HTTP call: TLS (`InsecureSkipVerify`, root CAs, client certificates), proxy, the current authorization (token sources and refresh, certificate tokens, `api.WithAuth`), `api.WithTenant` (selects the tenant of Basic credentials), `api.WithExtraHeaders` and the request middleware. The upgrade request is passed through the client's request middleware and the headers it ends with are sent. Limits: only request-side changes apply (headers, host); the response side sees a synthetic empty `101` response; a transport which is not an `*http.Transport` falls back to the proxy from the environment. Realtime reads the authorization again on every (re)connect, so a token refreshed since is used.

### Deferred Execution

Prepare operations (including parameter resolution) without sending, then execute after user confirmation:

```go
// Prepare operation with full parameter resolution
ctx := api.WithDeferredExecution(context.Background(), true)
prepared := client.ManagedObjects.Delete(ctx, "name:my-device")

// Inspect the prepared request (device name already resolved to ID)
fmt.Printf("About to delete: %s\n", prepared.Request.URL.Path)

// Get user confirmation
if confirmPrompt("Delete this device?") {
    // Execute the prepared operation
    result := prepared.Execute(context.Background())
    if result.Err != nil {
        return result.Err
    }
}
```

See [DEFERRED_EXECUTION.md](./DEFERRED_EXECUTION.md) for detailed patterns and examples.

### Header Redaction Control

Control whether sensitive headers are visible in logs (for debugging):

```go
// Disable header redaction for debugging (⚠️ use carefully)
ctx := api.WithRedactHeaders(context.Background(), false)

// Now Authorization headers will be visible in logs
result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})
```

See [REQUEST_INSPECTION.md](./REQUEST_INSPECTION.md) for security considerations.

## Resource Resolution

The SDK provides consistent patterns for resolving resource references across services:

### Resolver String Formats

Most services support flexible resource reference formats:

```go
// Direct ID (no lookup)
"12345"

// Name-based lookup (with wildcard support)
"name:deviceName"
"name:device*"

// External ID lookup
"ext:c8y_Serial:ABC123"
"ext:c8y_IMEI:123456789"

// Query-based lookup (inventory query)
"query:type eq 'c8y_Device'"
"query:name eq 'device01' and type eq 'c8y_Device'"
```

### Service Integration

Services that reference devices (Operations, Measurements, Events, Alarms) integrate the DeviceResolver:

```go
// Operations service with device resolution
result := client.Operations.List(ctx, operations.ListOptions{
    DeviceID: "name:my-device",  // Automatically resolved to ID
    Status:   "PENDING",
})

// Measurements with external ID lookup
result := client.Measurements.List(ctx, measurements.ListOptions{
    Source: "ext:c8y_Serial:ABC123",  // Resolved to device ID
})

// Events with query-based lookup
result := client.Events.List(ctx, events.ListOptions{
    Source: "query:name eq 'sensor01'",  // Query resolves to device ID
})
```

See [DEVICE_RESOLVER_PATTERN.md](./DEVICE_RESOLVER_PATTERN.md) for implementation details.

## Pagination

The v2 API provides flexible pagination approaches:

### Collection Results

List operations return collection results with pagination metadata:

```go
result := client.Operations.List(ctx, operations.ListOptions{
    DeviceID:          "12345",
    PaginationOptions: pagination.PaginationOptions{
        PageSize: 100,
    },
})

// Access items
for _, op := range result.Data.Items() {
    fmt.Println(op.ID(), op.Status())
}

// Pagination metadata available in Meta
if stats, ok := result.Meta["statistics"].(map[string]any); ok {
    fmt.Println("Total pages:", stats["totalPages"])
    fmt.Println("Current page:", stats["currentPage"])
}
```

### Iterator Pattern

For retrieving all items across pages:

```go
// ListAll returns an iterator that automatically handles pagination
iterator := client.Operations.ListAll(ctx, operations.ListOptions{
    DeviceID: "12345",
    Status:   "PENDING",
})

// Iterate over all operations
for op := range iterator.Items() {
    fmt.Printf("%s: %s\n", op.ID(), op.Status())
}

// Check for errors
if iterator.Err() != nil {
    return iterator.Err()
}
```

### Custom Unmarshaling with Iterators

Transform items during iteration:

```go
type CustomMeasurement struct {
    ID          string  `json:"id"`
    Temperature float64 `json:"c8y_Temperature.T.value"`
}

// Get collection result
result := client.Measurements.List(ctx, measurements.ListOptions{
    Source: "12345",
})

// Transform items during iteration
for m := range result.IterAs[CustomMeasurement]() {
    fmt.Printf("Temp: %.2f\n", m.Temperature)
}
```

## Tenant-scoped endpoints

Some endpoints have a `{tenantId}` path parameter, for example `/tenant/tenants/{tenantId}/options` or `/user/{tenantId}/users`. Their options structs have a `Tenant` (or `TenantID`) field, and the SDK picks the value in this order:

1. `api.WithTenant(ctx, "t123")` on the request context. It wins over the options field and does not change which credentials are used.
2. The `Tenant` field in the options, when it is not empty.
3. The client's tenant (`ClientOptions.Auth.Tenant`, or the tenant found by `Client.ResolveTenant`).

So in most code you leave `Tenant` empty:

```go
// Lists users of the client's own tenant
own := client.Users.List(ctx, users.ListOptions{})

// Lists users of a subtenant with the same credentials
sub := client.Users.List(api.WithTenant(ctx, "t456"), users.ListOptions{})
```

## Listing applications

`client.Applications.List` calls `GET /application/applications` and filters on the server with the query parameters in `applications.ListOptions` (`Name`, `Owner`, `Subscriber`, `ProvidedFor`, `Tenant`, `User`, `Type`, and more). The filters can be combined.

The `ListBy*` methods call separate endpoints that each take one value in the path:

| Method | Endpoint | Returns |
|---|---|---|
| `ListByName` | `/application/applicationsByName/{name}` | Applications with that name |
| `ListByTenant` | `/application/applicationsByTenant/{tenantId}` | Applications the tenant owns or is subscribed to |
| `ListByOwner` | `/application/applicationsByOwner/{tenantId}` | Applications the tenant owns |
| `ListByUser` | `/application/applicationsByUser/{username}` | Applications the user can access |

Use `List` when you need to combine filters or need a filter that has no `ListBy*` method (`Type`, `ProvidedFor`, `Availability`). Use a `ListBy*` method to match the endpoint other Cumulocity clients use for the same lookup.

## Service Architecture

### Service Structure Pattern

All services follow a consistent structure:

```go
type Service struct {
    svc httpcore.Service              // HTTP client (internal, not exported)
    DeviceResolver *managedobjects.DeviceResolver  // For device reference resolution (where applicable)
}

func NewService(common *httpcore.Service, moService *managedobjects.Service) *Service {
    return &Service{
        svc:        *common,
        DeviceResolver: managedobjects.NewDeviceResolver(moService),
    }
}
```

### Method Patterns

Services implement standard CRUD operations:

```go
// Get single resource
Get(ctx context.Context, ID string, opts ...GetOptions) op.Result[T]

// List resources (paginated)
List(ctx context.Context, opts ListOptions) op.Result[T]

// List all resources (iterator)
ListAll(ctx context.Context, opts ListOptions) *Iterator[T]

// Create resource
Create(ctx context.Context, body any) op.Result[T]

// Update resource
Update(ctx context.Context, ID string, body any) op.Result[T]

// Delete resource
Delete(ctx context.Context, ID string) op.Result[T]
```

### Options Pattern

Operations accept option structs for parameters:

```go
type ListOptions struct {
    DeviceID          string              // Service-specific fields
    Status            string
    PaginationOptions pagination.PaginationOptions  // Embedded pagination
}

// Usage
result := client.Operations.List(ctx, operations.ListOptions{
    DeviceID: "12345",
    Status:   "PENDING",
    PaginationOptions: pagination.PaginationOptions{
        PageSize:    50,
        CurrentPage: 1,
    },
})
```

## Advanced Patterns

### GetOrCreate Pattern

Idempotent operations that create if not exists:

```go
// Also available: GetOrCreateByFragment, GetOrCreateByExternalID and
// GetOrCreateWith (custom query).
result := client.ManagedObjects.GetOrCreateByName(ctx, "my-device", "c8y_Device", map[string]any{
    "c8y_IsDevice": map[string]any{},
    // Additional properties...
})

// Result.Status indicates what happened
switch result.Status {
case op.StatusOK:
    fmt.Println("Already existed:", result.Data.ID())
case op.StatusCreated:
    fmt.Println("Created new:", result.Data.ID())
case op.StatusFailed:
    return result.Err
}
```

### Custom Request Modification

Per-request headers and query parameters are set on the context:

```go
ctx = api.WithExtraHeaders(ctx, map[string]string{"X-Custom-Header": "value"})
ctx = api.WithExtraQueryParams(ctx, url.Values{"withChildren": []string{"true"}})
result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{})
```

To change every request a client sends, use `api.Middleware`, a standard
`http.RoundTripper` wrapper:

```go
client := api.NewClient(api.ClientOptions{
    Middleware: []api.Middleware{api.CountRequests(&requests)},
})
```

`client.Resty()` returns the underlying resty client for cases these cannot
cover. It is not covered by the v2 compatibility guarantee.

### Retry and Idempotency

The Result type tracks retry behavior:

```go
result := client.Operations.Create(ctx, op)

// Check if operation can be retried
if result.Err != nil && result.Retryable {
    // Safe to retry
    result = client.Operations.Create(ctx, op)
}

// Check if operation is idempotent
if result.Idempotent {
    // Same request can be sent multiple times safely
}
```

## Error Handling

### Error Values

Errors are `*api.Error` (an alias of `core.Error`) values carrying the HTTP status, the Cumulocity error type and message, and the raw body. The SDK provides sentinels for common statuses:

```go
var (
    ErrBadRequest          = Error{Code: 400}
    ErrUnauthorized        = Error{Code: 401}
    ErrForbidden           = Error{Code: 403}
    ErrNotFound            = Error{Code: 404}
    ErrConflict            = Error{Code: 409}
    ErrUnprocessableEntity = Error{Code: 422}
    ErrServer500           = Error{Code: 500} // also 501, 502, 503
)
```

### Checking errors

Match an error by status with `errors.Is`, or use the helpers, which also see through `fmt.Errorf("...: %w", err)` wrapping:

```go
result := client.Alarms.Get(ctx, id)
switch {
case errors.Is(result.Err, api.ErrNotFound): // same as api.IsNotFound(result.Err)
    // the alarm does not exist
case api.IsUnauthorized(result.Err), api.IsForbidden(result.Err):
    // bad credentials or missing permissions
case api.IsConflict(result.Err):
    // the resource already exists
case result.Err != nil:
    if code, ok := api.StatusCode(result.Err); ok {
        log.Printf("request failed with HTTP %d", code)
    }
}
```

Notes:

* The status is available for every HTTP error response, also when the body is not JSON (an HTML page from a proxy, plain text, or an empty body). The message then falls back to the status text.
* "Not found" is one concept. `errors.Is(err, api.ErrNotFound)` is true for a real 404, for "no result" from `Result.First()` on an empty collection (`op.ErrNotFound`), and for the not-found errors the SDK builds itself, such as when looking up an item by name (`core.ErrNotFound`).
* `api.StatusCode(err)` returns `ok == false` for errors without an HTTP status, such as a transport failure or a validation error. Use `errors.Is(err, context.DeadlineExceeded)` for timeouts.
* Library code never exits or panics on a runtime failure. The realtime and notification2 clients return connection and handshake errors from `Connect`. See `Client.Err()` of the realtime client for failures which stop it later.

```go
// Check error status
if api.ErrHasStatus(result.Err, 404) {
    // Handle not found
}
```

### Deleting resources that do not exist

Deletes are idempotent: deleting a resource that is already gone is not an error. The result has `Err == nil`, `Status == op.StatusSkipped`, and `HTTPStatus == 404`. Delete-by-name and delete-by-external-ID behave the same way when the lookup finds nothing. Other errors, such as `401`, `403`, or `5xx`, are still returned.

```go
result := client.ManagedObjects.Delete(ctx, id, managedobjects.DeleteOptions{})
if result.Err != nil {
    return result.Err // the delete failed
}
if result.Status == op.StatusSkipped {
    // the managed object did not exist
}
```

### Error Information

Results provide comprehensive error context:

```go
result := client.Operations.Create(ctx, op)
if result.Err != nil {
    fmt.Printf("Error: %v\n", result.Err)
    fmt.Printf("HTTP Status: %d\n", result.HTTPStatus)
    fmt.Printf("Retryable: %v\n", result.Retryable)
    fmt.Printf("Attempts: %d\n", result.Attempts)
    fmt.Printf("Duration: %v\n", result.Duration)
    
    // Access request that failed
    if result.Request != nil {
        fmt.Printf("Failed URL: %s\n", result.Request.URL)
    }
}
```

## Authentication

The v2 client supports multiple authentication methods via `ClientOptions`:

```go
// Username / password (OAI-Secure — exchanges credentials for a short-lived token)
client := api.NewClient(api.ClientOptions{
    BaseURL: "https://mytenant.cumulocity.com",
    Auth: authentication.AuthOptions{
        Username: "admin",
        Password: "secret",
    },
})

// Bearer token (skip login, use a pre-obtained token directly)
client := api.NewClient(api.ClientOptions{
    BaseURL: "https://mytenant.cumulocity.com",
    Auth: authentication.AuthOptions{
        Token: "eyJ...",
    },
})

// Client certificate
client := api.NewClient(api.ClientOptions{
    BaseURL: "https://mytenant.cumulocity.com",
    Auth: authentication.AuthOptions{
        Certificate:    "/path/to/cert.pem",
        CertificateKey: "/path/to/key.pem",
    },
})

// Populate from environment variables (C8Y_BASEURL, C8Y_USER, C8Y_PASSWORD, C8Y_TOKEN)
client := api.NewClientFromEnvironment(api.ClientOptions{})
```

### LoginWithOptions

After creating a client, call `LoginWithOptions` to complete authentication and obtain a
session token. The `Method` and `Preference` fields control which flow is used:

```go
// Strict: exactly one method — fails if prerequisites are absent
_, err := client.LoginWithOptions(ctx, api.LoginOptions{
    Method: authentication.LoginMethodOAuth2Internal, // BASIC | OAUTH2_INTERNAL | CERTIFICATE | OAUTH2_DEVICE_FLOW | OAUTH2_BROWSER_FLOW
})

// Preference: ordered fallback list — unavailable methods are silently skipped
_, err := client.LoginWithOptions(ctx, api.LoginOptions{
    Preference: []authentication.LoginMethod{
        authentication.LoginMethodOAuth2Internal,
        authentication.LoginMethodOAuth2DeviceFlow,
    },
    OnSuccess: func(m authentication.LoginMethod) {
        fmt.Println("Logged in via:", m)
    },
})
```

### SSO — Browser Flow (Authorization Code)

```go
_, err := client.LoginWithOptions(ctx, api.LoginOptions{
    Method: authentication.LoginMethodOAuth2BrowserFlow,
    BrowserFlow: &api.BrowserFlowOptions{
        // Must match the redirect URI registered in the SSO provider.
        // Accepted forms: "127.0.0.1:5001", "127.0.0.1:5001/callback",
        // or "http://127.0.0.1:5001/callback" (explicit).
        CallbackURL: "http://127.0.0.1:5001/callback",
    },
})
```

### SSO — Device Flow (RFC 8628)

```go
_, err := client.LoginWithOptions(ctx, api.LoginOptions{
    Method: authentication.LoginMethodOAuth2DeviceFlow,
    DeviceFlow: &api.DeviceFlowOptions{
        // Zero-value causes OAuth2 endpoints to be auto-discovered.
        DisplayFunc: func(code *device.CodeResponse) error {
            fmt.Fprintf(os.Stderr, "Visit %s and enter code: %s\n",
                code.VerificationURI, code.UserCode)
            return nil
        },
    },
})
```

### TFA / MFA Callbacks (OAUTH2_INTERNAL)

```go
_, err := client.LoginWithOptions(ctx, api.LoginOptions{
    Method: authentication.LoginMethodOAuth2Internal,

    // Called once during first-time TOTP enrollment — display the QR code.
    QRCode: func(otpauthURL, secret string) {
        fmt.Println("Scan:", otpauthURL)
    },

    // Called interactively when a TOTP code is required.
    TOTPCode: func(ctx context.Context) (string, error) {
        fmt.Fprint(os.Stderr, "TOTP code: ")
        var code string
        fmt.Scan(&code)
        return code, nil
    },

    // Called when the server issues an SMS PIN challenge.
    SMSCode: func(ctx context.Context, ch api.SMSChallenge) (string, error) {
        fmt.Fprint(os.Stderr, "SMS PIN: ")
        var pin string
        fmt.Scan(&pin)
        return pin, nil
    },

    // Called when the server forces a password change at login time.
    PasswordChange: func(ctx context.Context, ch api.PasswordChangeChallenge) (string, string, error) {
        return "user@example.com", "newP@ssw0rd", nil
    },
})
```

See [examples/cli-login-preference/main.go](../examples/cli-login-preference/main.go) for a
complete, runnable demonstration of all login methods and callbacks.

### Certificate Login

`LoginMethodCertificate` exchanges a client certificate for an access token.
The transport depends on the private key, because the platform returns the
HTTPS token encrypted to the certificate's public key, which it can only do
securely for RSA keys:

| Key | Transport (default `CertificateTransportAuto`) |
|---|---|
| RSA | HTTPS: `POST /tenant/oauth/certificate` on the standard port; the JWE token (`RSA-OAEP-256` / `A256GCM`) is decrypted with the private key |
| EC, Ed25519 | mTLS: `POST /devicecontrol/deviceAccessToken` on `ClientOptions.MTLSPort` (default 8443) |

`Auto` never falls back from one transport to the other. Set
`AuthOptions.CertificateTransport` to `CertificateTransportHTTPS` or
`CertificateTransportMTLS` to force one; forcing HTTPS with a non-RSA key fails
before sending anything. If mTLS fails because port 8443 is blocked, the error
says so.

```go
client := api.NewClient(api.ClientOptions{
    BaseURL: "https://example.cumulocity.com",
    Auth: authentication.AuthOptions{
        Tenant:         "t12345",            // required for the HTTPS exchange
        Certificate:    "user.crt",          // file path or PEM (leaf or chain)
        CertificateKey: "user.key",
    },
})
_, err := client.LoginWithOptions(ctx, api.LoginOptions{
    Method:   authentication.LoginMethodCertificate,
    TOTPCode: promptForTOTP, // only needed when TFA is active for the user
})
```

The HTTPS exchange only works for accounts that opted in to certificate
authentication (`requiredAuthType` `CERTIFICATES`); device users need no
opt-in. `client.EnableCertificateAuth(ctx, api.EnableCertificateAuthOptions{})`
opts the current user (or, with `Target: api.CertificateAuthServiceUser`, a
microservice's service user) in. Because the platform cannot issue a
certificate token before the opt-in, it first checks the key and certificate
locally and verifies the chain against the tenant's trusted certificates. After
the switch, password logins stop working. To undo it: the "forgot password"
email flow for users, `Applications.ResetAuthType` by a tenant administrator for
service users.

For multi-tenant microservices whose service users use certificates, set
`ClientOptions.ContextCertificateSource` together with `ContextTokenExchange`:
per-tenant tokens then come from the certificate exchange instead of the
password.

## Migration from v1

### Key Differences

1. **Result type instead of direct returns**
   ```go
   // v1
   op, resp, err := client.Operation.Get(ctx, "12345")
   
   // v2
   result := client.Operations.Get(ctx, "12345")
   op := result.Data
   err := result.Err
   ```

2. **Service names (singular → plural)**
   ```go
   // v1: client.Operation, client.Measurement
   // v2: client.Operations, client.Measurements
   ```

3. **Options structs instead of variadic parameters**
   ```go
   // v1
   ops, resp, err := client.Operation.GetOperations(ctx, deviceID, WithStatus("PENDING"))
   
   // v2
   result := client.Operations.List(ctx, operations.ListOptions{
       DeviceID: deviceID,
       Status:   "PENDING",
   })
   ```

4. **JSONDoc models instead of structs**
   ```go
   // v1
   type Operation struct {
       ID     string `json:"id"`
       Status string `json:"status"`
   }
   
   // v2
   op := result.Data  // jsonmodels.Operation
   id := op.ID()      // Method access
   status := op.Status()
   ```

## Testing and Debugging

### Dry Run for Testing

```go
func TestOperationCreation(t *testing.T) {
    client := setupTestClient()
    ctx := api.WithDryRun(context.Background(), true)
    
    result := client.Operations.Create(ctx, operations.CreateOptions{
        DeviceID:    "test-device",
        Description: "Test operation",
    })
    
    // No HTTP request sent, but request is prepared
    assert.NotNil(t, result.Request)
    assert.Equal(t, "POST", result.Request.Method)
    assert.Contains(t, result.Request.URL.Path, "/devicecontrol/operations")
}
```

### Request Inspection

See [REQUEST_INSPECTION.md](./REQUEST_INSPECTION.md) for:
- Inspecting prepared requests
- Formatting as curl commands
- Logging request details
- Security considerations for header redaction

## Related Documentation

- **[DEFERRED_EXECUTION.md](./DEFERRED_EXECUTION.md)** - Detailed guide on deferred execution pattern for user confirmation prompts
- **[DEVICE_RESOLVER_PATTERN.md](./DEVICE_RESOLVER_PATTERN.md)** - Device/managed object resolution architecture and implementation
- **[REQUEST_INSPECTION.md](./REQUEST_INSPECTION.md)** - Request inspection, dry run, and security considerations

## Implementation Status

Core features:
- [x] Generic Result type with rich metadata
- [x] JSONDoc models with flexible access
- [x] Context-based execution control (dry run, deferred execution)
- [x] Request inspection and debugging
- [x] Device resolver pattern
- [x] Pagination and iterators
- [x] Custom response unmarshaling
- [x] GetOrCreate patterns
- [x] Free-form request bodies
- [x] Custom request modification

Authentication:
- [x] Basic authentication
- [x] Bearer token authentication
- [x] OAuth2 / SSO — browser flow (Authorization Code)
- [x] OAuth2 / SSO — device flow (RFC 8628)
- [x] Client certificate authentication
- [x] TFA callbacks (TOTP, SMS PIN, QR enrollment, forced password change)
- [x] Ordered login-method preference list with automatic fallback
- [ ] Extensible token renewal

Services:
- [x] Operations (with device resolver)
- [x] ManagedObjects
- [x] Measurements (with device resolver)
- [x] Events (with device resolver)
- [x] Alarms (with device resolver)
- [x] Applications
- [x] Binaries
- [x] Users, Tenants, etc.
- [ ] Additional specialized services

Advanced features:
- [ ] Composable services (ManagedObjects → Devices)
- [ ] Upsert operations
- [ ] Extensible middleware
- [ ] Response streaming for large datasets
