# Device Resolver Pattern

> **Part of:** [API_DESIGN.md](./API_DESIGN.md) - Cumulocity Go SDK v2  
> **Context:** This document describes the device resolver architecture that enables flexible device reference resolution across services.

## Summary

The device resolver pattern provides a consistent, reusable way for services to resolve device/managed object references. This pattern is used by measurements, operations, events, and alarms services.

**Use this when:** Implementing or using services that need to reference devices/managed objects through various identifiers (name, external ID, query, etc.).

## Architecture

### Core Components

1. **`managedobjects.DeviceResolver`** - Reusable resolver that wraps managed objects service
2. **Service Integration** - Services embed DeviceResolver for consistent device resolution
3. **String-Based Resolution** - All services use the same resolver string formats

### Resolution Formats

All services support these resolver formats for device references:

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

## Implementation Pattern

### For Measurements (Already Implemented)

```go
// Service struct
type Service struct {
    core.Service
    DeviceResolver *managedobjects.DeviceResolver
}

// NewService with device resolver
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
    return &Service{
        Service:        *common,
        DeviceResolver: managedobjects.NewDeviceResolver(moService),
    }
}

// ListOptions with Source field
type ListOptions struct {
    Source string `url:"source,omitempty"` // Supports resolver strings
    // ... other fields
}

// List method with resolution
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[T] {
    if opt.Source != "" && s.DeviceResolver != nil {
        resolutionCtx := ctx
        if ctxhelpers.IsDeferredExecution(ctx) {
            resolutionCtx = context.Background()
        }

        resolvedID, err := s.DeviceResolver.ResolveID(resolutionCtx, opt.Source, nil)
        if err != nil {
            return op.Failed[T](err, true)
        }
        opt.Source = resolvedID
    }
    // ... continue with request
}
```

### For Operations (To Be Implemented)

```go
// Service struct - add DeviceResolver
type Service struct {
    core.Service
    DeviceResolver *managedobjects.DeviceResolver
}

// Update NewService
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
    return &Service{
        Service:        *common,
        DeviceResolver: managedobjects.NewDeviceResolver(moService),
    }
}

// ListOptions - add resolver support to Device field
type ListOptions struct {
    Device string `url:"device,omitempty"` // Supports resolver strings
    // ... other fields
}

// Update List method to resolve Device field
// Same pattern as measurements
```

### For Events (To Be Implemented)

```go
// Service struct - add DeviceResolver
type Service struct {
    core.Service
    DeviceResolver *managedobjects.DeviceResolver
}

// Update NewService
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
    return &Service{
        Service:        *common,
        DeviceResolver: managedobjects.NewDeviceResolver(moService),
    }
}

// ListOptions - add resolver support to Source field
type ListOptions struct {
    Source string `url:"source,omitempty"` // Supports resolver strings
    // ... other fields
}

// Update List method to resolve Source field
// Same pattern as measurements
```

### For Alarms (To Be Implemented)

```go
// Service struct - add DeviceResolver  
type Service struct {
    core.Service
    DeviceResolver *managedobjects.DeviceResolver
}

// Update NewService
func NewService(common *core.Service, moService *managedobjects.Service) *Service {
    return &Service{
        Service:        *common,
        DeviceResolver: managedobjects.NewDeviceResolver(moService),
    }
}

// ListOptions - add resolver support to Source field
type ListOptions struct {
    Source string `url:"source,omitempty"` // Supports resolver strings
    // ... other fields
}

// Update List method to resolve Source field
// Same pattern as measurements
```

## Usage Examples

### Programmatic Usage

```go
// Using helper methods
opts := measurements.ListOptions{
    Source: client.Measurements.DeviceResolver.ByName("device01"),
}

// Using ByExternalID
opts := measurements.ListOptions{
    Source: client.Measurements.DeviceResolver.ByExternalID("c8y_Serial", "ABC123"),
}

// Using ByQuery
opts := measurements.ListOptions{
    Source: client.Measurements.DeviceResolver.ByQuery("type eq 'c8y_Device'"),
}
```

### String-Based Usage (e.g., from CLI)

```go
// User passes resolver string directly
opts := measurements.ListOptions{
    Source: "name:device01",
}

opts := measurements.ListOptions{
    Source: "ext:c8y_Serial:ABC123",
}

opts := measurements.ListOptions{
    Source: "query:type eq 'c8y_Device'",
}
```

## Benefits

1. **Consistency** - All services use the same resolution patterns
2. **Reusability** - DeviceResolver is created once, used by multiple services
3. **Flexibility** - Supports multiple resolution strategies
4. **String-Based** - Easy to pass resolver strings from CLI or config
5. **Deferred Execution Support** - Properly handles deferred contexts
6. **Wildcard Support** - Name lookups support wildcard patterns
7. **Metadata** - Resolution provides rich metadata about matched devices

## Client Initialization

Services must be initialized in the correct order:

```go
// In client.go initialization
c.ManagedObjects = managedobjects.NewService(&c.common)

// Services that use device resolver must be initialized AFTER ManagedObjects
c.Measurements = measurements.NewService(&c.common, c.ManagedObjects)
c.Operations = operations.NewService(&c.common, c.ManagedObjects)  // To be implemented
c.Events = events.NewService(&c.common, c.ManagedObjects)          // To be implemented
c.Alarms = alarms.NewService(&c.common, c.ManagedObjects)          // To be implemented
```

## Testing

Tests verify all resolution patterns work correctly:

```go
func Test_Service_DeviceResolver_ByName(t *testing.T) {
    // Test using device name resolver
    opts := ServiceOptions{
        Source: client.Service.DeviceResolver.ByName("device01"),
    }
    // ... test code
}

func Test_Service_DeviceResolver_StringBased(t *testing.T) {
    // Test string-based resolver (as from CLI)
    opts := ServiceOptions{
        Source: "name:device01",
    }
    // ... test code
}
```

## Future Enhancements

- Consider adding resolver helpers directly to ListOptions for more ergonomic API
- Add resolver caching for repeated lookups
- Support batch resolution for multiple device references
- Add resolver middleware for custom resolution strategies
