// Command notification2-microservice is a MULTI_TENANT microservice that
// consumes Notification 2.0 messages for every subscribed tenant.
//
// WatchSubscriptions reports tenants as they subscribe and unsubscribe. For
// each tenant it starts SubscribeNotifications (running as the tenant's
// service user) and cancels it when the tenant unsubscribes.
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	api_notification2 "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/notification2"
	"github.com/reubenmiller/go-c8y/v2/pkg/microservice"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	ms, err := microservice.New(microservice.Options{})
	if err != nil {
		log.Fatalf("Could not create the microservice: %v", err)
	}
	if err := ms.Bootstrap(ctx); err != nil {
		slog.Error("Failed to bootstrap microservice", "err", err)
		os.Exit(1)
	}

	subscriber := os.Getenv("APPLICATION_NAME")
	if subscriber == "" {
		subscriber = "myevent-worker"
	}
	options := api_notification2.ClientOptions{
		Consumer: ms.Client.Notification2.NormalizedConsumer("c1"),
		Options: api_notification2.TokenOptions{
			ExpiresInMinutes:  1440,
			Subscriber:        ms.Client.Notification2.NormalizedConsumer(subscriber),
			DefaultSubscriber: subscriber,
			Subscription:      "CustomEvents",
		},
	}

	// One consumer per tenant, cancelled when the tenant unsubscribes.
	var mu sync.Mutex
	consumers := map[string]context.CancelFunc{}

	go func() {
		err := ms.WatchSubscriptions(ctx, microservice.WatchOptions{
			OnTenantAdded: func(_ context.Context, user model.ServiceUser) error {
				tenantCtx, cancel := context.WithCancel(ctx)
				mu.Lock()
				consumers[user.Tenant] = cancel
				mu.Unlock()
				go func() {
					_ = ms.SubscribeNotifications(tenantCtx, user.Tenant, options,
						func(_ context.Context, msg microservice.NotificationMessage) error {
							slog.Info("Received message", "tenant", user.Tenant, "action", msg.Action, "description", msg.Description)
							return nil
						})
				}()
				slog.Info("Started notifications", "tenant", user.Tenant)
				return nil
			},
			OnTenantRemoved: func(_ context.Context, tenant string) error {
				mu.Lock()
				if cancel, ok := consumers[tenant]; ok {
					cancel()
					delete(consumers, tenant)
				}
				mu.Unlock()
				slog.Info("Stopped notifications", "tenant", tenant)
				return nil
			},
		})
		if err != nil {
			slog.Error("Subscription watch stopped", "err", err)
		}
	}()

	mux := http.NewServeMux()
	ms.RegisterHealthEndpoints(mux)
	server := &http.Server{Addr: ":80", Handler: mux}
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("HTTP server failed", "err", err)
		os.Exit(1)
	}
}
