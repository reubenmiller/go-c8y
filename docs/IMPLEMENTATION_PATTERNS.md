# Go REST SDK Implementation Patterns

> **Related Documentation:**  
> - **[API_DESIGN.md](./API_DESIGN.md)** - User-facing API design and usage guide
> - **[DEFERRED_EXECUTION.md](./DEFERRED_EXECUTION.md)** - Deferred execution pattern
> - **[DEVICE_RESOLVER_PATTERN.md](./DEVICE_RESOLVER_PATTERN.md)** - Device resolution patterns

## Purpose

This document describes the **internal implementation patterns** used in the go-c8y v2 SDK. These patterns support:
- ✅ Composite multi-step flows (pipelines)
- ✅ Upsert and get-or-create semantics
- ✅ Retry logic with backoff
- ✅ Flexible JSON handling (typed models + free-form access)
- ✅ Rich result metadata

For user-facing API documentation, see [API_DESIGN.md](./API_DESIGN.md).

---

## Implementation Status

This document covers **implemented patterns**. Features planned but not yet implemented are listed in the [Future Plans](#future-plans) section.

---

## 1. Result Type with Status Tracking ✅

**Status:** Fully implemented in `op/result.go`

The `Result[T]` type wraps operation results with comprehensive metadata:

```go
type Status string

const (
    StatusOK        Status = "OK"        // Existing resource retrieved
    StatusCreated   Status = "Created"   // New resource created
    StatusUpdated   Status = "Updated"   // Existing resource modified
    StatusNoContent Status = "NoContent" // No response body
    StatusSkipped   Status = "Skipped"   // Operation skipped (e.g., no-op update)
    StatusDuplicate Status = "Duplicate" // Resource already exists (conflict)
    StatusNoMatch   Status = "NoMatch"   // No matches found
    StatusFailed    Status = "Failed"    // Operation failed
)

type Result[T any] struct {
    Data       T                  // The actual result data
    Status     Status             // Operation outcome
    HTTPStatus int                // HTTP status code
    Err        error              // Error if operation failed
    
    // Operation characteristics
    Retryable  bool               // Whether operation can be retried
    Idempotent bool               // Whether operation is idempotent
    Meta       map[string]any     // Additional metadata
    
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

**Usage:**
```go
result := op.OK(alarm, map[string]any{"fromCache": true})
if result.Status == op.StatusCreated {
    log.Info("Created new alarm", "id", result.Data.ID())
}
```

---

## 2. Pipeline Composition ✅

**Status:** Fully implemented in `op/pipeline.go`

Build complex flows using composable `Step[T]` functions:

```go
// Step represents a composable operation
type Step[T any] func(context.Context, T) (T, error)

// Sequential composition
func Pipe[T any](steps ...Step[T]) Step[T]

// Parallel execution with fan-in
func Parallel[T any](steps ...Step[T]) Step[T]

// Parallel execution collecting all results
func ParallelCollect[T any](steps ...Step[T]) func(context.Context, T) ([]T, []error)

// Conditional branching
func If[T any](predicate func(T) bool, thenStep, elseStep Step[T]) Step[T]

// Retry with exponential backoff
func WithRetry[T any](step Step[T], config RetryConfig) Step[T]

// Map/transform step output
func Map[T, U any](step Step[T], transform func(T) U) func(context.Context, T) (U, error)
```

**Example - Complex Device Onboarding:**
```go
onboardDevice := op.Pipe(
    validateDeviceData,
    op.WithRetry(createDevice, retryConfig),
    op.Parallel(
        assignToGroup,
        configureMonitoring,
        setupAlerts,
    ),
    registerWithExternalSystem,
)

result, err := onboardDevice(ctx, deviceData)
```

---

## 3. Get-or-Create Pattern ✅

**Status:** Fully implemented in `op/get_or_create.go` and service methods

Client-side implementation with idempotent create semantics:

```go
// GetOrCreateR executes get-or-create pattern with Result
func GetOrCreateR[T any](
    ctx context.Context,
    finder func(context.Context) op.Result[T],
    creator func(context.Context) op.Result[T],
) op.Result[T]

// GetOrCreateWithFind uses a find operation for non-unique criteria
func GetOrCreateWithFind[T any](
    finder FindFunc[T],
    creator CreateFunc[T],
    matcher func(T) bool,
) func(context.Context, T) (Result[T], error)
```

**Service implementations:**
```go
// Managed Objects
func (s *Service) GetOrCreateByName(ctx context.Context, name, objType string, body map[string]any) op.Result[jsonmodels.ManagedObject]
func (s *Service) GetOrCreateWith(ctx context.Context, body map[string]any, query string) op.Result[jsonmodels.ManagedObject]
func (s *Service) GetOrCreateByFragment(ctx context.Context, fragment string, body map[string]any) op.Result[jsonmodels.ManagedObject]
func (s *Service) GetOrCreateByExternalID(ctx context.Context, opts GetOrCreateByExternalIDOptions) op.Result[jsonmodels.ManagedObject]

// Software
func (s *Service) GetOrCreateByName(ctx context.Context, name, softwareType string, body any) op.Result[jsonmodels.Software]
```

**Example:**
```go
// Get or create device - idempotent operation
result := client.ManagedObjects.GetOrCreateByName(ctx, "sensor-01", "c8y_Device", map[string]any{
    "c8y_IsDevice": struct{}{},
    "c8y_SupportedMeasurements": []string{"c8y_Temperature"},
})

switch result.Status {
case op.StatusOK:
    fmt.Println("Already exists:", result.Data.ID())
case op.StatusCreated:
    fmt.Println("Created new:", result.Data.ID())
case op.StatusFailed:
    return result.Err
}
```

---

## 4. Upsert Pattern ✅

**Status:** Fully implemented in `op/upsert.go`

Get → Update or Create with various conflict resolution strategies:

```go
// Basic upsert
func Upsert[T any](
    getter GetFunc[T],
    updater UpdateFunc[T],
    creator CreateFunc[T],
    keyFunc KeyFunc[T],
) func(context.Context, T) (Result[T], error)

// Upsert with delta calculation (only updates changed fields)
func UpsertWithMerge[T any](
    getter GetFunc[T],
    updater UpdateFunc[T],
    creator CreateFunc[T],
    keyFunc KeyFunc[T],
    mergeFunc MergeFunc[T],
) func(context.Context, T) (Result[T], error)

// Handles 409 Conflict responses with retries
func UpsertWithConflictResolution[T any](
    getter GetFunc[T],
    updater UpdateFunc[T],
    creator CreateFunc[T],
    keyFunc KeyFunc[T],
    maxConflictRetries int,
) func(context.Context, T) (Result[T], error)

// Optimistic locking with version/etag
func UpsertWithOptimisticLocking[T any](
    getter GetFunc[T],
    updater UpdateFunc[T],
    creator CreateFunc[T],
    keyFunc KeyFunc[T],
    versionFunc func(T) string,
    maxRetries int,
) func(context.Context, T) (Result[T], error)

// Batch upsert
func UpsertBatch[T any](
    getter GetFunc[T],
    updater UpdateFunc[T],
    creator CreateFunc[T],
    keyFunc KeyFunc[T],
) func(context.Context, []T) ([]Result[T], error)
```

**Example:**
```go
// Event binary upsert - try create, fallback to update on 409
func (s *Service) Upsert(ctx context.Context, eventID string, opt UploadFileOptions) op.Result[jsonmodels.EventBinary] {
    result := core.Execute(ctx, s.createB(eventID, opt), jsonmodels.NewEventBinary)
    if result.Err == nil {
        return result
    }

    // 409 Conflict - binary already exists, replace it
    if !core.ErrHasStatus(result.Err, 409) {
        return result
    }
    return core.Execute(ctx, s.updateB(eventID, opt), jsonmodels.NewEventBinary)
}
```

**Delta merge example:**
```go
device := model.ManagedObject{
    ID:   "12345",
    Name: "Updated Name",
}

upsertFn := op.UpsertWithMerge(
    getter,
    updater,
    creator,
    func(d model.ManagedObject) string { return d.ID },
    op.DeltaMerge[model.ManagedObject], // Generic merge function
)

result, err := upsertFn(ctx, device)

switch result.Status {
case op.StatusCreated:
    // Device didn't exist, was created
case op.StatusUpdated:
    // Device existed, was updated
case op.StatusSkipped:
    // Device existed but no changes needed
}
```

---

## 5. Retry and Backoff ✅

**Status:** Implemented in `op/pipeline.go`

Configurable retry strategies with exponential backoff:

```go
type RetryConfig struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
    Jitter          bool
    ShouldRetry     func(error) bool
}

// Wrap any step with retry logic
func WithRetry[T any](step Step[T], config RetryConfig) Step[T]
```

**Example:**
```go
// Retry device creation with custom config
createWithRetry := op.WithRetry(createDevice, op.RetryConfig{
    MaxAttempts:     5,
    InitialInterval: 200 * time.Millisecond,
    MaxInterval:     10 * time.Second,
    Multiplier:      2.5,
    Jitter:          true,
})

result, err := createWithRetry(ctx, device)
```

---

## 6. Free-form JSON with gjson ✅

**Status:** Implemented via `jsondoc.Facade` pattern

Support both typed and untyped responses:

```go
// JSONDoc facade
type Operation struct {
    jsondoc.Facade  // Flexible JSON access
}

func NewOperation(b []byte) Operation {
    return Operation{Facade: jsondoc.NewFacade(b)}
}

// Strongly-typed accessors
func (o Operation) ID() string {
    return o.GetString("id")
}

func (o Operation) Status() string {
    return o.GetString("status")
}

// Raw JSON access
func (o Operation) Get(path string) any {
    return o.Facade.Get(path)
}

func (o Operation) Bytes() []byte {
    return o.Facade.Bytes()
}
```

**Usage:**
```go
// Typed access
operation := result.Data
fmt.Println(operation.ID())

// Custom field access
customValue := operation.Get("c8y_CustomFragment.value")

// Unmarshal to custom struct
type CustomOp struct {
    ID string `json:"id"`
    MyField int `json:"myField"`
}
var custom CustomOp
json.Unmarshal(operation.Bytes(), &custom)
```

---

## 7. Error Handling ✅

**Status:** Implemented in `core` and `op` packages

Categorized errors with rich context:

```go
// Standard error values
var (
    ErrBadRequest          = Error{Code: 400}
    ErrUnauthorized        = Error{Code: 401}
    ErrForbidden           = Error{Code: 403}
    ErrNotFound            = Error{Code: 404}
    ErrConflict            = Error{Code: 409}
    ErrTooManyRequests     = Error{Code: 429}
    ErrInternalServer      = Error{Code: 500}
)

// Check error status
func ErrHasStatus(err error, code ...int) bool
```

**Result error information:**
```go
result := client.Operations.Create(ctx, op)
if result.Err != nil {
    fmt.Printf("Error: %v\n", result.Err)
    fmt.Printf("HTTP Status: %d\n", result.HTTPStatus)
    fmt.Printf("Retryable: %v\n", result.Retryable)
    fmt.Printf("Attempts: %d\n", result.Attempts)
    fmt.Printf("Duration: %v\n", result.Duration)
}
```

---

## 8. Service Integration

Services provide high-level methods that use these patterns:

```go
type Service struct {
    core.Service
    DeviceResolver *managedobjects.DeviceResolver
}

// Standard CRUD operations return Result[T]
func (s *Service) Get(ctx context.Context, id string) op.Result[T]
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[T]
func (s *Service) Create(ctx context.Context, body any) op.Result[T]
func (s *Service) Update(ctx context.Context, id string, body any) op.Result[T]
func (s *Service) Delete(ctx context.Context, id string) op.Result[T]

// GetOrCreate patterns
func (s *Service) GetOrCreateByName(ctx, name, objType string, body any) op.Result[T]
func (s *Service) GetOrCreateWith(ctx, body any, query string) op.Result[T]
```

---

## 9. Best Practices

When implementing or using these patterns:

1. **Always use context** - Pass context for cancellation and timeout
2. **Check Result.Status** - Don't just check errors, inspect status for business logic
3. **Use GetOrCreate for idempotent setup** - Safer than manual existence checks
4. **Apply retry at the right level** - Per-step or per-pipeline as needed
5. **Track idempotency** - Use Result.Idempotent and idempotency keys
6. **Log Result.RequestID** - Essential for distributed tracing
7. **Prefer typed models** - Fall back to gjson only when necessary
8. **Test with Result assertions** - Verify status, not just errors
9. **Use deferred execution for confirmations** - See DEFERRED_EXECUTION.md
10. **Leverage device resolvers** - See DEVICE_RESOLVER_PATTERN.md

---

## Future Plans

The following features are planned but not yet implemented:

### Repository Pattern with Interfaces

Generic repository interfaces for CRUD operations:

```go
type Getter[T any] interface {
    GetR(ctx context.Context, key string, opts ...RequestOption) (Result[T], error)
}

type Repository[T any] interface {
    Getter[T]
    Finder[T]
    Creator[T]
    Updater[T]
    Deleter
}
```

**Status:** Not implemented. Services currently provide methods directly without interface abstraction.

**Considerations:**
- Would enable more abstract patterns but adds complexity
- Current service-based approach is working well
- Could be added if strong use case emerges

### Request Configuration Options

Context-based options for request customization:

```go
type RequestOption func(*RequestConfig)

func WithHeaders(headers map[string]string) RequestOption
func WithTimeout(timeout time.Duration) RequestOption
func WithRetry(config RetryConfig) RequestOption

// Usage
result := client.Operations.Get(ctx, id, 
    op.WithTenant("t12345"),
    op.WithTimeout(5*time.Second),
)
```

**Status:** Partially implemented. Some services support options, but not standardized across all services.

**Considerations:**
- Would provide more flexible request customization
- Need to design consistent pattern across all services
- May conflict with context-based patterns already in use

### Code Generation

Automated generation of services and repositories from templates:

```bash
go generate ./pkg/c8y/c8y_api/...
```

**Status:** Not implemented. All services manually implemented.

**Considerations:**
- Could improve consistency and reduce boilerplate
- Would need to maintain templates as patterns evolve
- Manual implementation currently provides good flexibility

### Composable Services

Service composition for more specific types:

```go
// ManagedObjects => Devices
type DeviceService struct {
    *managedobjects.Service
    // Device-specific methods
}

// Applications => Microservices
type MicroserviceService struct {
    *applications.Service
    // Microservice-specific methods
}
```

**Status:** Not implemented. Some specialization exists (e.g., software/firmwareversions) but not a general pattern.

**Considerations:**
- Could provide better type safety for specific resource types
- Need to determine which resources benefit from specialization
- Current approach with device types works reasonably well

### Advanced Upsert Features

- **Conditional upsert** - Only upsert if conditions met
- **Bulk upsert with transaction semantics** - All-or-nothing batch operations
- **Upsert with cascade** - Update related resources automatically

**Status:** Basic upsert and batch upsert implemented. Advanced features not yet needed.

### Enhanced Pipeline Features

- **Pipeline visualization** - Generate diagrams of pipeline flows
- **Pipeline metrics** - Automatic timing and performance tracking per step
- **Pipeline debugging** - Step-by-step execution mode
- **Pipeline composition DSL** - Builder pattern for complex pipelines

**Status:** Basic pipeline composition works well. Advanced features could improve debugging.

---

## Related Documentation

- **[API_DESIGN.md](./API_DESIGN.md)** - Complete user-facing API documentation
- **[V2.md](./V2.md)** - API overview and implementation status  
- **[DEFERRED_EXECUTION.md](./DEFERRED_EXECUTION.md)** - Deferred execution for user confirmations
- **[DEVICE_RESOLVER_PATTERN.md](./DEVICE_RESOLVER_PATTERN.md)** - Device resolution architecture
- **[REQUEST_INSPECTION.md](./REQUEST_INSPECTION.md)** - Request debugging and security
