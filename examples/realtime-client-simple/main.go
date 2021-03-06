package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func main() {
	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, false)

	// Create realtime connection
	err := client.Realtime.Connect()

	if err != nil {
		log.Fatalf("Could not connect to /cep/realtime. %s", err)
	}

	// Subscribe to all measurements
	ch := make(chan *c8y.Message)
	client.Realtime.Subscribe(c8y.RealtimeMeasurements(), ch)

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	for {
		select {
		case msg := <-ch:
			log.Printf("Received measurement. %s", msg.Payload.Data)

		case <-signalCh:
			// Enable ctrl-c to stop
			log.Printf("Stopping realtime client")
			return
		}
	}
}
