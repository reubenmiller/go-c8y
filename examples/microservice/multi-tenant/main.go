// Example of a MULTI_TENANT microservice using only the standard library's
// net/http server.
//
// It demonstrates the two tenant-context features of the SDK:
//
//  1. Incoming requests: the TenantContext middleware reads the caller's
//     credentials (forwarded by the Cumulocity platform) and binds them to the
//     request context, so every API call made with r.Context() automatically
//     executes on behalf of the caller's tenant.
//
//  2. Background jobs: ForEachTenant runs a function once per subscribed
//     tenant with a context carrying that tenant's service user.
package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/microservice"
)

func main() {
	ms := microservice.New(microservice.Options{})
	if err := ms.Bootstrap(context.Background()); err != nil {
		log.Fatalf("Failed to bootstrap microservice: %s", err)
	}

	mux := http.NewServeMux()

	// Standard endpoints: /health, /env, /prometheus, /logfile
	ms.RegisterHealthEndpoints(mux)

	// Authenticated, tenant-scoped endpoint. The middleware binds the caller's
	// tenant to the request context; the handler below needs no tenant logic.
	requireTenant := ms.TenantContext()
	mux.Handle("GET /devices", requireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Executes as the caller's tenant because r.Context() carries the
		// caller's credentials.
		result := ms.Client.Devices.List(r.Context(), devices.ListOptions{
			PaginationOptions: pagination.NewPaginationOptions(10),
		})
		if result.Err != nil {
			http.Error(w, result.Err.Error(), http.StatusBadGateway)
			return
		}

		names := []string{}
		for item, err := range op.Iter2(result) {
			if err != nil {
				continue
			}
			names = append(names, item.Name())
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tenant":  microservice.TenantFromContext(r.Context()),
			"devices": names,
		})
	})))

	// Background job: report the device count of every subscribed tenant.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for ; ; <-ticker.C {
			err := ms.ForEachTenant(context.Background(), func(ctx context.Context, user model.ServiceUser) error {
				result := ms.Client.Devices.List(ctx, devices.ListOptions{
					PaginationOptions: pagination.PaginationOptions{PageSize: 1, WithTotalPages: true},
				})
				if result.Err != nil {
					return result.Err
				}
				slog.Info("Tenant device count", "tenant", user.Tenant, "total", result.Data.Get("statistics.totalPages").Int())
				return nil
			})
			if err != nil {
				slog.Warn("Per-tenant job failed", "err", err)
			}
		}
	}()

	port := ms.Config.GetString("server.port")
	slog.Info("Starting microservice", "port", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
