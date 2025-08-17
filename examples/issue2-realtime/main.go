package main

import (
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func main() {
	// Summary:
	// Subscribe, Unsubscribe and Subscribe to the given device id
	//
	// Usage:	go run main.go -device 12345
	//
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, false)

	// Get arguments
	var deviceID string
	flag.StringVar(&deviceID, "device", "", "Device ID")
	flag.Parse()

	if deviceID == "" {
		panic("-device parameter must not be empty!")
	}

	// Create realtime connection
	err := client.Realtime.Connect()

	if err != nil {
		slog.Error("Could not connect to /cep/realtime", "err", err)
		panic(err)
	}

	// Subscribe to all measurements
	subscriptionPattern := c8y.RealtimeMeasurements(deviceID)
	ch := make(chan *c8y.Message)
	<-client.Realtime.Subscribe(subscriptionPattern, ch)

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	<-time.After(1 * time.Second)

	// TODO: Debug unsubscribe option, it results in empty messages being written to the channel
	<-client.Realtime.UnsubscribeAll()
	// <-client.Realtime.Unsubscribe(subscriptionPattern)

	// ch2 := make(chan *c8y.Message)
	<-client.Realtime.Subscribe(subscriptionPattern, ch)

	for {
		select {
		case msg := <-ch:
			if msg != nil {
				slog.Info("Received measurement", "payload", msg.Payload.Data)
			} else {
				slog.Info("Received empty message")
			}

		case <-signalCh:
			// Enable ctrl-c to stop
			slog.Info("Stopping realtime client")
			return
		}
	}
}
