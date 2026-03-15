package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	api_notification2 "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/notification2"
)

var (
	verbose      = flag.Bool("verbose", false, "Verbose logging")
	subscription = flag.String("subscription", "", "Subscription")
	subscriber   = flag.String("subscriber", "goclient", "Subscriber")
	consumer     = flag.String("consumer", "app1", "Consumer")
)

func main() {
	flag.Parse()

	if !*verbose {
		handler := slog.New(slog.DiscardHandler)
		slog.SetDefault(handler)
	}

	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	result := client.Notification2.SubscribeStream(ctx, api_notification2.ClientOptions{
		Token:    os.Getenv("NOTIFICATION2_TOKEN"),
		Consumer: *consumer,
		Options: api_notification2.TokenOptions{
			ExpiresInMinutes: 2,
			Subscription:     *subscription,
			Subscriber:       *subscriber,
		},
	})

	if result.Err != nil {
		panic(result.Err)
	}

	stream := result.Data

	slog.Info("Listening to messages")
	for msg, err := range stream.Items() {
		if err != nil {
			slog.Error("received error", "err", err)
			break
		}
		slog.Info("On message", "payload", msg.Data.Bytes())
		if err := stream.AcknowledgeMessage(msg.Identifier); err != nil {
			slog.Warn("Failed to send message ack", "err", err)
		}
	}
}
