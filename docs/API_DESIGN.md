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

// Functional composition
result2 := op.MapResult(result, func(op jsonmodels.Operation) string {
    return op.ID()
})
```

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
result := client.Operations.Delete(ctx, "12345")

// Inspect what would be sent
if result.Request != nil {
    fmt.Println("Method:", result.Request.Method)
    fmt.Println("URL:", result.Request.URL)
    fmt.Println("Headers:", result.Request.Header)
}
```

**Security:** Sensitive headers are automatically redacted in dry run logs. See [REQUEST_INSPECTION.md](./REQUEST_INSPECTION.md) for details.

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

## Service Architecture

### Service Structure Pattern

All services follow a consistent structure:

```go
type Service struct {
    core.Service                             // Base service with HTTP client
    DeviceResolver *managedobjects.DeviceResolver  // For device reference resolution (where applicable)
}

func NewService(common *core.Service, moService *managedobjects.Service) *Service {
    return &Service{
        Service:        *common,
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
result := client.ManagedObjects.GetOrCreate(ctx, managedobjects.GetOrCreateOptions{
    Name: "my-device",
    Type: "c8y_Device",
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

Services allow request customization:

```go
result := client.ManagedObjects.Get(ctx, "12345", managedobjects.GetOptions{
    RequestModifier: func(r *resty.Request) *resty.Request {
        return r.
            SetHeader("X-Custom-Header", "value").
            SetQueryParam("withChildren", "true")
    },
})
```

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

The SDK provides standard error values for common scenarios:

```go
var (
    ErrBadRequest          = Error{Code: 400}
    ErrUnauthorized        = Error{Code: 401}
    ErrForbidden           = Error{Code: 403}
    ErrNotFound            = Error{Code: 404}
    ErrConflict            = Error{Code: 409}
    ErrUnprocessableEntity = Error{Code: 422}
    ErrTooManyRequests     = Error{Code: 429}
    ErrInternalServer      = Error{Code: 500}
)

// Check error status
if api.ErrHasStatus(result.Err, 404) {
    // Handle not found
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

The v2 client supports multiple authentication methods:

```go
// Basic authentication
client := api.NewClient(baseURL, username, password)

// Bearer token
client := api.NewClientFromToken(baseURL, token)

// OAuth2 / PKCE (for SSO)
// TODO: Document OAuth2 setup

// Custom authentication
client.Auth = authentication.AuthOptions{
    // Custom auth configuration
}
```

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
- [ ] OAuth2 / SSO authentication
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
