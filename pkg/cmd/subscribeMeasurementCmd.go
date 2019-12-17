// TODO

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

type subscribeMeasurementCmd struct {
	*baseCmd

	flagTimeoutSec int64
	flagCount      int64
}

func newSubscribeMeasurementCmd() *subscribeMeasurementCmd {
	ccmd := &subscribeMeasurementCmd{}

	cmd := &cobra.Command{
		Use:   "subscribe",
		Short: "Subscribe to realtime measurements",
		Long:  `Subscribe to realtime measurements`,
		Example: `
$ c8y measurements subscribe --device 12345
Subscribe to measurements (in realtime) for device 12345
		`,
		RunE: ccmd.subscribeMeasurement,
	}

	cmd.SilenceUsage = true

	cmd.Flags().StringSlice("device", []string{""}, "Device ID")
	cmd.Flags().Int64Var(&ccmd.flagTimeoutSec, "timeout", 30, "Timeout in seconds")
	cmd.Flags().Int64Var(&ccmd.flagCount, "count", 0, "Max number of realtime notifications to wait for")

	// Required flags

	ccmd.baseCmd = newBaseCmd(cmd)

	return ccmd
}

func (n *subscribeMeasurementCmd) subscribeMeasurement(cmd *cobra.Command, args []string) error {

	// options
	device := "*"

	if cmd.Flags().Changed("device") {
		deviceInputValues, deviceValue, err := getFormattedDeviceSlice(cmd, args, "device")

		if err != nil {
			return newUserError("no matching devices found", deviceInputValues, err)
		}

		if len(deviceValue) == 0 {
			return newUserError("no matching devices found", deviceInputValues)
		}

		for _, item := range deviceValue {
			if item != "" {
				device = newIDValue(item).GetID()
			}
		}
	}

	// filter and selectors
	// filters := getFilterFlag(cmd, "filter")

	// Common outputfile option
	// Write results to file?
	/* outputfile := ""
	if v, err := getOutputFileFlag(cmd, "outputFile"); err == nil {
		outputfile = v
	} else {
		return err
	} */

	return subscribe(c8y.RealtimeMeasurements(device), n.flagTimeoutSec, n.flagCount)
}

func subscribe(channelPattern string, timeoutSec int64, maxMessages int64) error {

	// TODO:
	// client.Realtime = c8y.NewRealtimeClient(client.BaseURL.String(), nil, client.TenantName, client.Username, client.Password)
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
			fmt.Printf("%s\n", msg.Payload.Data)
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
