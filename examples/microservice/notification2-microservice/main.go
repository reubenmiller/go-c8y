package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	api_notification2 "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/notification2"
)

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `{"status":"UP"}`)
}

func handleRequests() {
	http.HandleFunc("/health", health)
	slog.Error(http.ListenAndServe(":80", nil).Error())
	os.Exit(1)
}

func subscribeToNotifications(ctx context.Context, subscription string, consumer string) {
	c8yClient := api.NewClient(api.ClientOptions{
		BaseURL: authentication.HostFromEnvironment(),
		Auth:    authentication.FromEnvironment(),
	})

	defaultSubscriber := os.Getenv("APPLICATION_NAME")
	if defaultSubscriber == "" {
		defaultSubscriber = "myevent-worker"
	}

	consumer = c8yClient.Notification2.NormalizedConsumer(consumer)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	result := c8yClient.Notification2.SubscribeStream(context.Background(), api_notification2.ClientOptions{
		Consumer: consumer,
		Options: api_notification2.TokenOptions{
			ExpiresInMinutes:  1440,
			Subscriber:        consumer,
			DefaultSubscriber: defaultSubscriber,
			Subscription:      subscription,
			Shared:            false,
		},
	})

	if result.Err != nil {
		slog.Error("Error while creating notification subscription", "error", result.Err)
		panic(result.Err)
	}

	stream := result.Data

	slog.Info("Listening to messages")
	for msg, err := range stream.Items() {
		if err != nil {
			slog.Error("received error", "err", err)
			break
		}
		slog.Info("received message.", "action", msg.Action, "description", msg.Description, "identifier", msg.Identifier, "payload", msg.Data.Bytes())
		if err := stream.AcknowledgeMessage(msg.Identifier); err != nil {
			slog.Warn("Failed to send message ack", "err", err)
		}
	}
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	consumer := "c1"
	go subscribeToNotifications(context.Background(), "CustomEvents", consumer)
	handleRequests()
}
