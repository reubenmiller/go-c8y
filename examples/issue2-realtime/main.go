package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
)

func main() {
	// Summary:
	// Subscribe, Unsubscribe and Subscribe to the given device id
	//
	// Usage:	go run main.go -device 12345
	//
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	// Get arguments
	var deviceID string
	flag.StringVar(&deviceID, "device", "", "Device ID")
	flag.Parse()

	if deviceID == "" {
		panic("-device parameter must not be empty!")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Subscribe to all measurements
	result := client.Measurements.SubscribeStream(ctx, deviceID)
	if result.Err != nil {
		log.Fatalf("Failed to setup subscription. %s", result.Err)
	}

	<-time.After(1 * time.Second)

	for msg, err := range result.Data.Items() {
		if err != nil {
			slog.Error("received error", "err", err)
			break
		}
		slog.Info("Received measurement", "payload", msg.Data.Bytes())
	}
}
