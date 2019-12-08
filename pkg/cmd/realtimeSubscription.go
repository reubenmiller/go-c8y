package cmd

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/spf13/cobra"
)

type subscribeRealtimeCmd struct {
	device   string
	deviceID string
	timeout  int

	*baseCmd
}

func newSubscribeRealtimeCmd() *subscribeRealtimeCmd {
	ccmd := &subscribeRealtimeCmd{}

	cmd := &cobra.Command{
		Use:   "measurements",
		Short: "Subscribe to realtime measurements",
		Long:  `Subscribe to realtime measurements for a specific device`,
		Example: `
			TODO
		`,
		RunE: ccmd.subscribeRealtime,
	}

	// Flags
	cmd.Flags().StringVarP(&ccmd.device, "device", "d", "", "name (accepts wildcards)")
	cmd.Flags().IntVarP(&ccmd.timeout, "timeout", "t", 30, "Timeout in seconds")

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *subscribeRealtimeCmd) subscribeRealtime(cmd *cobra.Command, args []string) error {
	if n.cmd.Flags().Changed("device") {
		deviceInputValues, deviceValue, err := getFormattedDeviceSlice(cmd, args, "device")

		if err != nil {
			return newUserError("no matching devices found", deviceInputValues, err)
		}

		if len(deviceValue) == 0 {
			return newUserError("no matching devices found", deviceInputValues)
		}

		for _, item := range deviceValue {
			if item != "" {
				n.deviceID = newIDValue(item).GetID()
				break
			}
		}
	}

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

	filterText := []byte("")

	for {
		select {
		case msg := <-ch:
			if msg != nil {
				if len(filterText) == 0 || bytes.Contains(msg.Payload.Data, filterText) {
					fmt.Printf("%s\n", msg.Payload.Data)
				}
			} else {
				Logger.Debug("Stopping realtime client")
				log.Printf("Received empty message")
			}

		case <-signalCh:
			// Enable ctrl-c to stop
			Logger.Info("Stopping realtime client")
			client.Realtime.Disconnect()
			return nil
		}
	}
}
