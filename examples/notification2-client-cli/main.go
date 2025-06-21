package main

import (
	"context"
	"flag"
	"io"
	"log"
	"os"
	"os/signal"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/c8y/notification2"
)

var (
	verbose      = flag.Bool("verbose", false, "Verbose logging")
	subscription = flag.String("subscription", "", "Subscription")
	subscriber   = flag.String("subscriber", "goclient", "Subscriber")
	consumer     = flag.String("consumer", "app1", "Consumer")
)

func main() {
	var err error
	flag.Parse()

	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, true)

	c8yToken := os.Getenv("C8Y_TOKEN")
	if c8yToken != "" {
		client.SetToken(c8yToken)
	}

	notificationClient, err := client.Notification2.CreateClient(context.Background(), c8y.Notification2ClientOptions{
		Token:    os.Getenv("NOTIFICATION2_TOKEN"),
		Consumer: *consumer,
		Options: c8y.Notification2TokenOptions{
			ExpiresInMinutes: 2,
			Subscription:     *subscription,
			Subscriber:       *subscriber,
		},
	})
	if err != nil {
		panic(err)
	}

	err = notificationClient.Connect()

	if err != nil {
		panic(err)
	}

	ch := make(chan notification2.Message)
	notificationClient.Register("*", ch)

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	log.Printf("Listening to messages")

	for {
		select {
		case msg := <-ch:
			log.Printf("On message: %s", msg.Payload)
			if err := notificationClient.SendMessageAck(msg.Identifier); err != nil {
				log.Printf("Failed to send message ack: %s", err)
			}

		case <-signalCh:
			// Enable ctrl-c to stop
			log.Printf("Stopping client")
			notificationClient.Close()
			return
		}
	}
}
