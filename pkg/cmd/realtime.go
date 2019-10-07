package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

var realtimeCmd = &cobra.Command{
	Use:   "realtime",
	Short: "Query inventory managed objects",
	Long:  `Query inventory managed objects using Cumulocity's query language`,
	Run: func(cmd *cobra.Command, args []string) {
		// do nothing
	},
}

var measurementsCmd = &cobra.Command{
	Use:   "measurements",
	Short: "Subscribe to realtime measurement events",
	Long:  `Subscribe to realtime measurements for a specific device`,
	Run: func(cmd *cobra.Command, args []string) {

		// Create realtime connection
		err := client.Realtime.Connect()

		if err != nil {
			log.Fatalf("Could not connect to /cep/realtime. %s", err)
		}

		// Subscribe to all measurements
		subscriptionPattern := c8y.RealtimeMeasurements(measurementsArgDeviceID)
		ch := make(chan *c8y.Message)
		<-client.Realtime.Subscribe(subscriptionPattern, ch)

		// Enable ctrl-c stop signal
		signalCh := make(chan os.Signal, 1)
		signal.Notify(signalCh, os.Interrupt)

		<-client.Realtime.Subscribe(subscriptionPattern, ch)

		time.AfterFunc(time.Duration(measurementsArgTimeout)*time.Second, func() {
			// Exit after the timeout
			signalCh <- os.Kill
		})

		for {
			select {
			case msg := <-ch:
				if msg != nil {
					fmt.Printf("%s\n", msg.Payload.Data)
				} else {
					log.Printf("Received empty message")
				}

			case <-signalCh:
				// Enable ctrl-c to stop
				log.Printf("Stopping realtime client")
				client.Realtime.Disconnect()
				return
			}
		}

	},
}

var (
	measurementsArgDeviceID string
	measurementsArgTimeout  int
)

func init() {
	rootCmd.AddCommand(realtimeCmd)

	// Flags
	measurementsCmd.Flags().StringVarP(&measurementsArgDeviceID, "deviceID", "d", "", "name (accepts wildcards)")
	measurementsCmd.Flags().IntVarP(&measurementsArgTimeout, "timeout", "t", 30, "Timeout in seconds")
	realtimeCmd.AddCommand(measurementsCmd)
}
