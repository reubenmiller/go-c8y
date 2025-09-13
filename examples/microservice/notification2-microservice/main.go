package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/notification2"
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
	c8yClient := c8y.NewClientFromEnvironment(nil, true)

	defaultSubscriber := os.Getenv("APPLICATION_NAME")
	if defaultSubscriber == "" {
		defaultSubscriber = "myevent-worker"
	}

	consumer = c8yClient.Notification2.NormalizedConsumer(consumer)

	notificationClient, err := c8yClient.Notification2.CreateClient(ctx, c8y.Notification2ClientOptions{
		Consumer: consumer,
		Options: c8y.Notification2TokenOptions{
			ExpiresInMinutes:  1440,
			Subscriber:        consumer,
			DefaultSubscriber: defaultSubscriber,
			Subscription:      subscription,
			Shared:            false,
		},
	})
	if err != nil {
		slog.Error("Error while creating notification subscription", "error", err.Error())
		panic(err)
	}

	// connect and send all received messages to channel
	err = notificationClient.Connect()
	if err != nil {
		slog.Error("Error while connecting to notification subscription", "error", err.Error())
	}
	ch := make(chan notification2.Message)
	notificationClient.Register("*", ch)

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	for {
		select {
		case msg := <-ch:
			slog.Info("received message.", "action", msg.Action, "description", msg.Description, "identifier", msg.Identifier, "payload", msg.Payload)
			if err := notificationClient.SendMessageAck(msg.Identifier); err != nil {
				slog.Warn("Failed to send message ack: %s", "error", err)
			}
		case <-signalCh:
			// Enable ctrl-c to stop
			slog.Info("Stopping client")
			notificationClient.Close()
			return
		}
	}
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	for _, item := range os.Environ() {
		if !strings.Contains(item, "PASSWORD") {
			slog.Info("env.", "var", item)
		}
	}

	consumer := os.Getenv("HOSTNAME2")
	if consumer == "" {
		consumer = "c1"
	}
	go subscribeToNotifications(context.Background(), "CustomEvents", consumer)
	handleRequests()
}
