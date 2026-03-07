package managedobjects

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/source"
)

// DeviceRef is a typed reference to a device or managed object. Construct it
// using the package-level helpers ByID, ByName, ByExternalID, or ByQuery, or
// cast a variable string with DeviceRef(id) when needed.
//
// Example:
//
//	Source: managedobjects.ByName("myDevice")
//	Source: managedobjects.ByExternalID("c8y_Serial", "ABC123")
//	Source: managedobjects.DeviceRef(someDynamicStringVar) // explicit cast
type DeviceRef string

// ByID creates a direct-ID device reference. No resolution is performed;
// the provided id is used as-is in the API call.
func ByID(id string) DeviceRef { return DeviceRef(id) }

// ByName creates a device reference resolved by managed-object name.
// Supports wildcard patterns using "*".
func ByName(name string) DeviceRef { return DeviceRef("name:" + name) }

// ByExternalID creates a device reference resolved by external ID.
// externalType is e.g. "c8y_Serial"; externalID is the value.
func ByExternalID(externalType, externalID string) DeviceRef {
	return DeviceRef("ext:" + externalType + ":" + externalID)
}

// ByQuery creates a device reference resolved by an inventory query.
// The query must return exactly one result.
func ByQuery(query string) DeviceRef { return DeviceRef("query:" + query) }

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

// ResolveID resolves a DeviceRef that may contain a resolver scheme.
// Supports all managed object resolver patterns:
//   - "12345" -> "12345" (direct ID)
//   - "name:deviceName" -> "<id>" (lookup by name)
//   - "ext:c8y_Serial:ABC123" -> "<id>" (lookup by external ID)
//   - "query:type eq 'c8y_Device'" -> "<id>" (lookup by inventory query)
//
// If meta is not nil, it will be populated with metadata about the resolution.
func (r *DeviceResolver) ResolveID(ctx context.Context, id DeviceRef, meta map[string]any) (string, error) {
	if r.moService == nil {
		return "", fmt.Errorf("managed objects service not configured")
	}
	return r.moService.ResolveID(ctx, string(id), meta)
}

// ByID returns a direct ID reference (no lookup needed).
func (r *DeviceResolver) ByID(id string) DeviceRef {
	if r.moService == nil {
		return DeviceRef(id)
	}
	return DeviceRef(r.moService.ByID(id))
}

// ByName creates a name-based lookup reference.
// Supports wildcard patterns using "*".
func (r *DeviceResolver) ByName(name string) DeviceRef {
	if r.moService == nil {
		return DeviceRef(fmt.Sprintf("name:%s", name))
	}
	return DeviceRef(r.moService.ByName(name))
}

// ByExternalID creates an external ID-based lookup reference.
func (r *DeviceResolver) ByExternalID(externalType, externalID string) DeviceRef {
	if r.moService == nil {
		return DeviceRef(fmt.Sprintf("ext:%s:%s", externalType, externalID))
	}
	return DeviceRef(r.moService.ByExternalID(externalType, externalID))
}

// ByQuery creates a query-based lookup reference.
// The query should return exactly one result.
func (r *DeviceResolver) ByQuery(query string) DeviceRef {
	if r.moService == nil {
		return DeviceRef(fmt.Sprintf("query:%s", query))
	}
	return DeviceRef(r.moService.ByQuery(query))
}

// RegisterResolver allows registering custom ID resolvers
// Example: RegisterResolver("custom", myResolver)
// Then use: ResolveID(ctx, "custom:value")
func (r *DeviceResolver) RegisterResolver(scheme string, resolver source.Resolver) {
	if r.moService != nil {
		r.moService.RegisterResolver(scheme, resolver)
	}
}
