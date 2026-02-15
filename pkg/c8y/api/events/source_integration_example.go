package events

// Example of how to integrate source resolution into the events API

/*
import (
	"context"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/source"
)

// OPTION 1: Add SourceRef field alongside Source (Recommended for backward compatibility)

type ListOptions struct {
	// Direct source ID (backward compatible)
	Source string `url:"source,omitempty"`

	// Optional: Source resolver (resolved before request)
	// If set, this takes precedence over Source field
	SourceRef source.Resolver `url:"-"`

	// ... rest of fields
	CreatedFrom time.Time `url:"createdFrom,omitempty,omitzero"`
	// etc.

	pagination.PaginationOptions
}

// List events - with automatic source resolution
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Event] {
	opts := opt

	// Resolve source if SourceRef is provided
	if opts.SourceRef != nil {
		id, err := opts.SourceRef.ResolveID(ctx)
		if err != nil {
			return op.Failed[jsonmodels.Event](err)
		}
		opts.Source = id
	}

	return core.ExecuteReturnCollection(ctx, s.ListB(opts), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewEvent)
}

// OPTION 2: Replace Source with any type (more flexible but breaking change)

type ListOptionsV2 struct {
	// Source can be either string (direct ID) or source.Resolver
	Source any `url:"-"` // Don't URL encode, we handle it manually

	// Internal resolved source for URL encoding
	sourceResolved string `url:"source,omitempty"`

	// ... rest of fields
}

func (s *Service) ListV2(ctx context.Context, opt ListOptionsV2) op.Result[jsonmodels.Event] {
	opts := opt

	// Resolve source (handles both string and Resolver)
	id, err := source.Resolve(ctx, opts.Source)
	if err != nil {
		return op.Failed[jsonmodels.Event](err)
	}
	opts.sourceResolved = id

	return core.ExecuteReturnCollection(ctx, s.ListB(opts), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewEvent)
}

// USAGE EXAMPLES:

func ExampleUsage(client *api.Client) {
	ctx := context.Background()

	// 1. Direct ID (backward compatible)
	events1 := client.Events.List(ctx, ListOptions{
		Source: "12345",
	})

	// 2. By external ID
	events2 := client.Events.List(ctx, ListOptions{
		SourceRef: client.Source.ByExternalID("c8y_Serial", "ABC123"),
	})

	// 3. By device name
	events3 := client.Events.List(ctx, ListOptions{
		SourceRef: client.Source.ByName("MyDevice"),
	})

	// 4. By inventory query
	events4 := client.Events.List(ctx, ListOptions{
		SourceRef: client.Source.ByQuery("name eq 'MyDevice' and type eq 'c8y_Device'"),
	})

	// 5. Custom resolver
	events5 := client.Events.List(ctx, ListOptions{
		SourceRef: client.Source.Custom("cached-device", func(ctx context.Context) (string, error) {
			// Your custom logic here
			return getCachedDeviceID(), nil
		}),
	})

	// 6. User-defined resolver (no client needed)
	customResolver := source.Custom{
		Description: "config-file",
		Resolve: func(ctx context.Context) (string, error) {
			return readDeviceIDFromConfig(), nil
		},
	}
	events6 := client.Events.List(ctx, ListOptions{
		SourceRef: customResolver,
	})
}

// FOR CLI: The client can parse source strings

func ExampleCLIUsage(client *api.Client, sourceArg string) {
	ctx := context.Background()

	// Parse from CLI argument
	// Supports: "12345", "id:12345", "ext:type:id", "name:MyDevice", "query:..."
	sourceRef, err := client.Source.Parse(sourceArg)
	if err != nil {
		log.Fatal(err)
	}

	events := client.Events.List(ctx, ListOptions{
		SourceRef: sourceRef,
	})
}
*/
