package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"github.com/jeremywohl/flatten"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

var (
	device  = flag.String("device", "", "Device id to subscribe to")
	series  = flag.String("series", "", "Device id to subscribe to")
	verbose = flag.Bool("verbose", false, "Verbose logging")
)

func getBetweenBytes(msg, startsWith, endsWith []byte) (string, error) {
	startIdx := bytes.Index(msg, startsWith)
	if startIdx == -1 {
		log.Printf("Could not find startsWith string %s", startsWith)
		return "", nil
	}
	startIdx += len(startsWith)
	endIdx := bytes.Index(msg[startIdx:len(msg)], endsWith)
	endIdx = startIdx + endIdx

	val, err := strconv.Unquote(`"` + string(msg[startIdx:endIdx]) + `"`)

	log.Printf("getUnit: startIdx=%d, endIdx=%d", startIdx, endIdx)
	log.Printf("getUnit: val=%s, err=%s, raw=%s", val, err, msg[startIdx:endIdx])

	return val, err
}

func getReverseBetweenBytes(msg, endsWith, startsWith []byte) (string, error) {
	endIdx := bytes.Index(msg, endsWith)
	if endIdx == -1 {
		log.Printf("Could not find startsWith string %s", endsWith)
		return "", nil
	}
	startIdx := bytes.Index(msg[0:endIdx], startsWith)
	endIdx = startIdx + endIdx

	val, err := strconv.Unquote(`"` + string(msg[startIdx:endIdx]) + `"`)

	log.Printf("getUnit: startIdx=%d, endIdx=%d", startIdx, endIdx)
	log.Printf("getUnit: val=%s, err=%s, raw=%s", val, err, msg[startIdx:endIdx])

	return val, err
}

type MeasurementSeries struct {
	Timestamp string  `json:"timestamp"`
	SourceID  string  `json:"sourceId"`
	Fragment  string  `json:"fragment"`
	Series    string  `json:"series"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

func parseMeasurementSeries(data []byte) ([]MeasurementSeries, error) {
	series := make([]MeasurementSeries, 0)
	jsonMap := make(map[string]interface{})

	if err := json.Unmarshal(data, &jsonMap); err != nil {
		return series, err
	}

	// fmt.Printf("payload: %s", data)

	flat, _ := flatten.Flatten(jsonMap, "", flatten.DotStyle)

	valueKeys := []string{}
	unitKeys := []string{}

	for key := range flat {
		if strings.HasSuffix(key, "value") {
			valueKeys = append(valueKeys, key)
		}
		if strings.HasSuffix(key, "unit") {
			unitKeys = append(unitKeys, key)
		}
	}

	for i, key := range valueKeys {
		if i < len(unitKeys) {
			keyParts := strings.Split(key, ".")
			unit, _ := flat[unitKeys[i]].(string)
			series = append(series, MeasurementSeries{
				SourceID:  flat["source.id"].(string),
				Timestamp: flat["time"].(string),
				Fragment:  keyParts[0],
				Series:    keyParts[1],
				Value:     flat[key].(float64),
				Unit:      unit,
			})
		}
	}

	return series, nil
}

func main() {
	var err error
	var deviceID string
	flag.Parse()

	if !*verbose {
		log.SetOutput(io.Discard)
	}

	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, false)

	if *device != "" && *device != "*" {
		devices, _, err := client.Inventory.GetDevicesByName(context.Background(), *device, c8y.NewPaginationOptions(1))

		if err != nil {
			panic(err)
		}

		if len(devices.ManagedObjects) == 0 {
			log.Panicf("Could not find a device with the name '%s'", *device)
		}

		deviceID = devices.ManagedObjects[0].ID
	} else {
		fmt.Println("Using a wildcard")
		deviceID = "*"
	}

	// Create realtime connection
	err = client.Realtime.Connect()

	if err != nil {
		log.Fatalf("Could not connect to /cep/realtime. %s", err)
	}

	// Subscribe to all measurements
	ch := make(chan *c8y.Message)
	client.Realtime.Subscribe(c8y.RealtimeMeasurements(deviceID), ch)

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	log.Printf("Listening to subscriptions")

	for {
		select {
		case msg := <-ch:
			if bytes.Contains(msg.Payload.Data, []byte(*series)) {
				series, err := parseMeasurementSeries(msg.Payload.Data)

				if err != nil {
					return
				}
				for _, iSeries := range series {
					if deviceID == "*" {
						fmt.Printf("%-15s\t%s\t%-40s\t%.2f %s\n", iSeries.SourceID, iSeries.Timestamp, iSeries.Fragment+"."+iSeries.Series, iSeries.Value, iSeries.Unit)
					} else {
						fmt.Printf("%-15s\t%s\t%-40s\t%.2f %s\n", *device, iSeries.Timestamp, iSeries.Fragment+"."+iSeries.Series, iSeries.Value, iSeries.Unit)
					}
				}
			}

		case <-signalCh:
			// Enable ctrl-c to stop
			log.Printf("Stopping realtime client")
			return
		}
	}
}
