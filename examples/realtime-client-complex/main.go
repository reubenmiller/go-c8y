package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	c8y "github.com/reubenmiller/go-c8y"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, false)

	err := client.Realtime.Connect()

	if err != nil {
		log.Fatalf("Could not connect to /cep/realtime. %s", err)
	}
	client.Realtime.WaitForConnection()

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
				log.Printf("Received measurement. %s", msg.Payload.Data)

			case <-ctx.Done():
				log.Printf("Stopping realtime client")
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
		select {
		case <-signalCh:
			cancel()
			return
		}
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
		log.Fatalf("Could not retrieve alarms. %s", err)
	}

	log.Printf("Total alarms: %d", len(alarmCollection.Alarms))
}
