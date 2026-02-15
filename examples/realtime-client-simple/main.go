package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	result := client.Measurements.SubscribeStream(ctx, "*")
	if result.Err != nil {
		log.Fatalf("Failed to setup subscription. %s", result.Err)
	}

	for msg, err := range result.Data.Items() {
		if err != nil {
			slog.Error("received error", "err", err)
			break
		}
		slog.Info("Received measurement", "payload", msg.Data.Bytes())
	}
}
