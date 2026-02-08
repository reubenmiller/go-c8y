package main

import (
	"context"
	"fmt"
	"iter"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/operations"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pipeline"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y_api.NewClient(c8y_api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	if enabledDebug, err := strconv.ParseBool(os.Getenv("DEBUG")); err == nil {
		client.Client.SetDebug(enabledDebug)
	}

	// Add API call statistics middleware
	stats := c8y_api.NewStatsMap()
	client.Client.AddResponseMiddleware(c8y_api.MiddlewareCountByMethodAndPath(stats))

	ctx := context.Background()

	// Search for devices, then fetch the list of operations that are still PENDING,
	// and then set the operation to FAILED.
	//
	// go-c8y-cli equivalent:
	//   c8y devices list --type "c8y_Linux" |
	//       c8y operations list --status PENDING --includeAll |
	//       c8y operations update --status FAILED --failureReason "Cancelled stale operation" --workers 5

	// Step 1: Stream all matching devices
	allDevices := client.Devices.ListAll(ctx, devices.ListOptions{
		Query: "has(isFake)",
		PaginationOptions: pagination.PaginationOptions{
			MaxItems: 10,
		},
	}).Items()

	// Step 2: Throttle devices to limit how fast ListAll calls are made
	// throttledDevices := pipeline.Throttle(allDevices, 1000*time.Millisecond)

	// Step 3: Expand each device into its pending operations (flat, no nesting)
	pendingOps := pipeline.Expand(allDevices, func(device jsonmodels.ManagedObject) iter.Seq2[jsonmodels.Operation, error] {
		pending := client.Operations.ListAll(ctx, operations.ListOptions{
			DeviceID: device.ID(),
			Status:   "PENDING",
		}).Items()

		executing := client.Operations.ListAll(ctx, operations.ListOptions{
			DeviceID: device.ID(),
			Status:   "EXECUTING",
		}).Items()
		return pipeline.Concat(pending, executing)
	})

	pendingOps2 := pipeline.Expand(pendingOps, func(operation jsonmodels.Operation) iter.Seq2[jsonmodels.Operation, error] {
		item := client.Operations.Update(ctx, operation.ID(), map[string]any{
			"status": "EXECUTING",
		})
		// conditional
		if item.Data.Exists("c8y_Command") {
			// skip
			return pipeline.EmptyOf(operation)
		}
		// op.SingleWithItem preserves the original operation on error
		// This ensures OnError callback receives the actual operation, not a zero-value
		return op.SingleWithItem(operation, item)
	})

	// Step 4: Update each pending operation to FAILED
	// Delay adds a per-worker pause after each request to avoid overwhelming the platform.
	err := pipeline.ForEach(ctx, pendingOps2, pipeline.Options{
		Workers:   2,
		Delay:     1000 * time.Millisecond,
		MaxErrors: 2,
		OnProgress: func(stats pipeline.Stats) {
			msg := fmt.Sprintf("\rOperations updated: %d (failed: %d)", stats.Completed, stats.Failed)
			if stats.LastError != nil {
				msg += fmt.Sprintf(" | Last error: %v", stats.LastError)
			}
			fmt.Print(msg)
		},
		OnError: func(item any, err error) {
			// Correlate the error with the specific item that failed
			if op, ok := item.(jsonmodels.Operation); ok {
				log.Printf("Failed to update operation %s: %v", op.ID(), err)
			} else {
				log.Printf("Failed to process item: %v", err)
			}
		},
	},
		func(ctx context.Context, op jsonmodels.Operation) error {
			result := client.Operations.Update(c8y_api.WithDryRun(ctx, false), op.ID(), map[string]any{
				"status":        "FAILED",
				"failureReason": "Cancelled stale operation",
			})
			return result.Err
		},
	)

	fmt.Println() // newline after progress

	fmt.Println("\nAPI call statistics:")
	for method, paths := range stats.All() {
		for path, count := range paths {
			fmt.Printf("%s %s: %d\n", method, path, count)
		}
	}

	if err != nil {
		// Display detailed error information
		switch e := err.(type) {
		case *pipeline.PipelineError:
			log.Printf("Pipeline completed with %d errors out of %d items", e.Failed, e.Completed)
			log.Printf("Sample errors:")
			for i, sampleErr := range e.SampleErrors {
				log.Printf("  %d: %v", i+1, sampleErr)
			}
		case *pipeline.AbortError:
			log.Printf("Pipeline aborted: %s", e.Reason)
			log.Printf("Failed: %d/%d", e.Failed, e.Completed)
			if len(e.SampleErrors) > 0 {
				log.Printf("Sample errors:")
				for i, sampleErr := range e.SampleErrors {
					log.Printf("  %d: %v", i+1, sampleErr)
				}
			}
		default:
			log.Fatalf("Pipeline failed: %v", err)
		}
	}
	fmt.Println("Done")
}
