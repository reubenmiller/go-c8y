package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/jsonUtilities"
	"github.com/spf13/cobra"
)

func subscribe(channelPattern string, timeoutSec int64, maxMessages int64, cmd *cobra.Command) error {

	if err := client.Realtime.Connect(); err != nil {
		log.Fatalf("Could not connect to /cep/realtime. %s", err)
	}

	msgCh := make(chan (*c8y.Message))

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	Logger.Infof("Listenening to subscriptions: %s", channelPattern)

	client.Realtime.Subscribe(channelPattern, msgCh)

	timeoutDuration := time.Duration(timeoutSec) * time.Second
	timeoutCh := time.After(timeoutDuration)

	defer func() {
		// client.Realtime.UnsubscribeAll()
		client.Realtime.Disconnect()
	}()

	var receivedCounter int64

	for {
		select {
		case <-timeoutCh:
			Logger.Info("Duration has expired. Stopping realtime client")
			return nil
		case msg := <-msgCh:

			data := jsonUtilities.UnescapeJSON(msg.Payload.Data)

			// show data on console
			cmd.Printf("%s\n", data)

			// return data from cli
			fmt.Printf("%s\n", data)

			receivedCounter++

			if maxMessages != 0 && receivedCounter >= maxMessages {
				return nil
			}

		case <-signalCh:
			// Enable ctrl-c to stop
			log.Printf("Stopping realtime client")
			return nil
		}
	}
}
