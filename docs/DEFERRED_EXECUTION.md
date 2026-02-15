# Deferred Execution

> **Part of:** [API_DESIGN.md](./API_DESIGN.md) - Cumulocity Go SDK v2  
> **Context:** This document describes the deferred execution pattern, a context-driven execution control feature in the v2 API.

## Summary

Deferred execution allows you to prepare an operation (including parameter resolution) without actually sending the HTTP request. This is useful for implementing confirmation prompts before destructive operations.

**Use this when:** You need to show users what will happen before executing a potentially destructive operation (delete, update, etc.).

## Overview

When deferred execution is enabled, operations will:
1. ✅ Resolve all parameters (including source resolution, e.g., device name → ID)
2. ✅ Build the complete HTTP request
3. ⏸️ **Not send the request** - execution is deferred until you explicitly call `.Execute()`

This differs from dry run mode, which mocks the HTTP call entirely. Deferred execution prepares everything for a real request but waits for your confirmation.

## Use Case: Confirmation Prompts

A common pattern in go-c8y-cli is prompting users before destructive operations:

```go
// 1. Prepare the operation (resolves parameters, builds request)
ctx := api.WithDeferredExecution(context.Background(), true)
prepared := client.ManagedObjects.Delete(ctx, "device-name")

// 2. Inspect the prepared request
if prepared.Request != nil {
    // Extract device ID from the URL
    parts := strings.Split(prepared.Request.URL.Path, "/")
    deviceID := parts[len(parts)-1]
    
    fmt.Printf("About to delete managed object:\n")
    fmt.Printf("  ID: %s\n", deviceID)
    fmt.Printf("  URL: %s\n", prepared.Request.URL)
}

// 3. Get user confirmation
if !confirmPrompt("Delete this device?") {
    fmt.Println("Operation cancelled")
    return // Don't execute
}

// 4. Execute the operation
result := prepared.Execute(context.Background())
if result.Err != nil {
    return result.Err
}

fmt.Println("Device deleted successfully")
```

## API Reference

### Context Functions

```go
// Enable deferred execution
ctx = api.WithDeferredExecution(ctx, true)

// Check if deferred execution is enabled
if api.IsDeferredExecution(ctx) {
    // ...
}
```

### Result Methods

```go
// Check if result has deferred execution
if result.IsDeferred() {
    // Inspect result.Request before executing
}

// Execute the deferred operation
result := prepared.Execute(ctx)
```

### Example: Parameter Resolution

Deferred execution still resolves parameters through the `source.Resolver` interface:

```go
// Device name will be resolved to ID during preparation
ctx := api.WithDeferredExecution(context.Background(), true)
prepared := client.ManagedObjects.Delete(ctx, "my-device-name")

// The request URL contains the resolved ID
fmt.Println(prepared.Request.URL.Path)
// Output: /inventory/managedObjects/12345

// User confirms, then execute
result := prepared.Execute(context.Background())
```

## Comparison: Dry Run vs Deferred Execution

| Feature | Dry Run | Deferred Execution |
|---------|---------|-------------------|
| **Parameter Resolution** | ✅ Yes | ✅ Yes |
| **HTTP Request Building** | ✅ Yes | ✅ Yes |
| **HTTP Call** | ❌ Mocked (returns mock data) | ⏸️ Deferred (waits for `.Execute()`) |
| **Response** | Mock response | Real response after execution |
| **Use Case** | Testing, debugging | User confirmation prompts |
| **Can be cancelled** | N/A (already mocked) | ✅ Just don't call `.Execute()` |

## Complete Example: Delete with Confirmation

```go
package main

import (
    "bufio"
    "context"
    "fmt"
    "os"
    "strings"

    "github.com/reubenmiller/go-c8y/pkg/c8y/api"
    "github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
)

func deleteWithConfirmation(client *api.Client, deviceID string) error {
    // Prepare the delete operation
    ctx := api.WithDeferredExecution(context.Background(), true)
    prepared := client.ManagedObjects.Delete(ctx, deviceID, managedobjects.DeleteOptions{})

    // Check if preparation succeeded
    if prepared.Err != nil {
        return fmt.Errorf("failed to prepare delete: %w", prepared.Err)
    }

    // Inspect the prepared request
    fmt.Printf("Prepared DELETE request:\n")
    fmt.Printf("  Method: %s\n", prepared.Request.Method)
    fmt.Printf("  URL: %s\n", prepared.Request.URL)
    fmt.Printf("  Device ID: %s\n", deviceID)

    // Optionally fetch device info for confirmation
    device, err := client.ManagedObjects.Get(context.Background(), deviceID, managedobjects.GetOptions{})
    if err == nil {
        fmt.Printf("  Device Name: %s\n", device.Data.Name)
        fmt.Printf("  Device Type: %s\n", device.Data.Type)
    }

    // Prompt for confirmation
    fmt.Print("\nConfirm deletion? [y/N]: ")
    reader := bufio.NewReader(os.Stdin)
    response, _ := reader.ReadString('\n')
    response = strings.ToLower(strings.TrimSpace(response))

    if response != "y" && response != "yes" {
        fmt.Println("Operation cancelled")
        return nil
    }

    // Execute the prepared operation
    fmt.Println("Executing delete...")
    result := prepared.Execute(context.Background())
    
    if result.Err != nil {
        return fmt.Errorf("delete failed: %w", result.Err)
    }

    fmt.Println("Device deleted successfully")
    return nil
}
```

## Cancellation

To cancel a deferred operation, simply don't call `.Execute()`:

```go
prepared := client.ManagedObjects.Delete(ctx, deviceID, options)

if !userConfirmed() {
    // Just return - operation is cancelled
    return nil
}

// Only execute if user confirmed
result := prepared.Execute(ctx)
```

## Implementation Notes

- Deferred execution uses dry run mode internally to build the request without sending it
- The prepared `Result` contains the fully-built HTTP request in `result.Request`
- Calling `.Execute()` on an already-executed result returns itself (idempotent)
- All execute functions (`ExecuteReturnResult`, `ExecuteReturnCollection`, `ExecuteBinaryResponse`, `ExecuteNoResult`) support deferred execution

## Best Practices

1. **Always check `prepared.Err`** before inspecting the request
2. **Use meaningful context** when calling `.Execute()` - you may want to add timeout or cancellation
3. **Extract information from `prepared.Request`** for display to users
4. **Combine with dry run** if you want to test without any side effects: deferred execution still resolves parameters, which might trigger lookups
