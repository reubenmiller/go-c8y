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

type subscribeRealtimeCmd struct {
	deviceID string
	timeout  int

	*baseCmd
}

func newSubscribeRealtimeCmd() *subscribeRealtimeCmd {
	ccmd := &subscribeRealtimeCmd{}

	cmd := &cobra.Command{
		Use:   "measurements",
		Short: "Subscribe to realtime measurement events",
		Long:  `Subscribe to realtime measurements for a specific device`,
		Example: `
			TODO
		`,
		RunE: ccmd.subscribeRealtime,
	}

	// Flags
	cmd.Flags().StringVarP(&ccmd.deviceID, "deviceID", "d", "", "name (accepts wildcards)")
	cmd.Flags().IntVarP(&ccmd.timeout, "timeout", "t", 30, "Timeout in seconds")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *subscribeRealtimeCmd) subscribeRealtime(cmd *cobra.Command, args []string) error {
	return n.doSubscribeRealtime(n.deviceID, n.timeout)
}

func (n *subscribeRealtimeCmd) doSubscribeRealtime(deviceID string, timeout int) error {
	// Create realtime connection
	err := client.Realtime.Connect()

	if err != nil {
		return newSystemErrorF("Could not connect to /cep/realtime. %s", err)
	}

	// Subscribe to all measurements
	subscriptionPattern := c8y.RealtimeMeasurements(deviceID)
	ch := make(chan *c8y.Message)
	<-client.Realtime.Subscribe(subscriptionPattern, ch)

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	<-client.Realtime.Subscribe(subscriptionPattern, ch)

	time.AfterFunc(time.Duration(timeout)*time.Second, func() {
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
			return nil
		}
	}
}
