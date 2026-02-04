package managedobjects

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/source"
)

// DeviceResolver provides device/managed object resolution capabilities for other services.
// This allows measurements, operations, events, and alarms to resolve device references
// using the same patterns as the managed objects service.
//
// Usage:
//
//	resolver := managedobjects.NewDeviceResolver(moService)
//	id, err := resolver.ResolveID(ctx, "name:myDevice", nil)
type DeviceResolver struct {
	moService *Service
}

// NewDeviceResolver creates a new device resolver that uses the managed objects service
// for resolution. This allows other services (measurements, operations, events, alarms)
// to resolve device references using managed object patterns.
func NewDeviceResolver(moService *Service) *DeviceResolver {
	return &DeviceResolver{
		moService: moService,
	}
}

// ResolveID resolves a device ID string that may contain a resolver scheme.
// Supports all managed object resolver patterns:
//   - "12345" -> "12345" (direct ID)
//   - "name:deviceName" -> "<id>" (lookup by name)
//   - "ext:c8y_Serial:ABC123" -> "<id>" (lookup by external ID)
//   - "query:type eq 'c8y_Device'" -> "<id>" (lookup by inventory query)
//
// If meta is not nil, it will be populated with metadata about the resolution.
func (r *DeviceResolver) ResolveID(ctx context.Context, id string, meta map[string]any) (string, error) {
	if r.moService == nil {
		return "", fmt.Errorf("managed objects service not configured")
	}
	return r.moService.ResolveID(ctx, id, meta)
}

// ByID returns a direct ID reference (no lookup needed).
// Returns: "12345"
func (r *DeviceResolver) ByID(id string) string {
	if r.moService == nil {
		return id
	}
	return r.moService.ByID(id)
}

// ByName creates a name-based lookup reference string.
// Supports wildcard patterns using "*".
// Returns: "name:deviceName"
func (r *DeviceResolver) ByName(name string) string {
	if r.moService == nil {
		return fmt.Sprintf("name:%s", name)
	}
	return r.moService.ByName(name)
}

// ByExternalID creates an external ID-based lookup reference string.
// Returns: "ext:type:externalID"
func (r *DeviceResolver) ByExternalID(externalType, externalID string) string {
	if r.moService == nil {
		return fmt.Sprintf("ext:%s:%s", externalType, externalID)
	}
	return r.moService.ByExternalID(externalType, externalID)
}

// ByQuery creates a query-based lookup reference string.
// The query should return exactly one result.
// Returns: "query:..."
func (r *DeviceResolver) ByQuery(query string) string {
	if r.moService == nil {
		return fmt.Sprintf("query:%s", query)
	}
	return r.moService.ByQuery(query)
}

// RegisterResolver allows registering custom ID resolvers
// Example: RegisterResolver("custom", myResolver)
// Then use: ResolveID(ctx, "custom:value")
func (r *DeviceResolver) RegisterResolver(scheme string, resolver source.Resolver) {
	if r.moService != nil {
		r.moService.RegisterResolver(scheme, resolver)
	}
}
