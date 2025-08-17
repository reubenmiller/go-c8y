package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, false)

	err := client.Realtime.Connect()

	if err != nil {
		slog.Error("Could not connect to /cep/realtime", "err", err)
		panic(err)
	}

	ch := make(chan *c8y.Message)
	client.Realtime.Subscribe(c8y.RealtimeMeasurements(), ch)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	c1, cancel := context.WithCancel(context.Background())

	exitCh := make(chan struct{})
	go func(ctx context.Context) {
		for {
			select {
			case msg := <-ch:
				slog.Info("Received measurement", "payload", msg.Payload.Data)

			case <-ctx.Done():
				slog.Info("Stopping realtime client")
				client.Realtime.UnsubscribeAll()
				client.Realtime.Close()
				exitCh <- struct{}{}
				return
			}
		}
	}(c1)

	defer func() {
		close(ch)
		client.Realtime.Close()
	}()
	go func() {
		<-signalCh
		cancel()
	}()
	<-exitCh

	// Get list of alarms with MAJOR Severity
	alarmCollection, _, err := client.Alarm.GetAlarms(
		context.Background(),
		&c8y.AlarmCollectionOptions{
			Severity: "MAJOR",
		},
	)

	// Always check for errors
	if err != nil {
		slog.Error("Could not retrieve alarms", "err", err)
		panic(err)
	}

	slog.Info("Alarms", "total", len(alarmCollection.Alarms))
}
