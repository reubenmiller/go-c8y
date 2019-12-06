package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jeremywohl/flatten"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

var (
	device     = flag.String("device", "", "Device id or name to subscribe to. Accepts wildcards but only the first result will be used")
	series     = flag.String("series", "", "Series filter. Only show values from this series. Accepts wildcards. Only valid if channel is set to measurements")
	channel    = flag.String("channel", "measurements", "Channel type. i.e. measurements, events, operations etc.")
	duration   = flag.Int64("duration", 60*60, "Duration in seconds that the realtime client should run for")
	verbose    = flag.Bool("verbose", false, "Verbose logging")
	jsonOutput = flag.Bool("json", false, "Display all output data as json, including measurements")
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

	c8y.SilenceLogger()

	timeoutDuration := time.Duration(*duration) * time.Second

	// Create the client from the following environment variables
	// C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD
	client := c8y.NewClientFromEnvironment(nil, false)

	// Create realtime connection
	err = client.Realtime.Connect()

	if err != nil {
		log.Fatalf("Could not connect to /cep/realtime. %s", err)
	}

	// Subscribe to all measurements
	ch := make(chan *c8y.Message)

	// TODO: only look up device if a custom channel is being used
	deviceID, err = findDevice(client, *device)

	if err != nil {
		panic(err)
	}

	channelPattern := ""
	switch *channel {
	case "alarms":
		channelPattern = c8y.RealtimeAlarms(deviceID)
	case "events":
		channelPattern = c8y.RealtimeEvents(deviceID)
	case "measurements":
		channelPattern = c8y.RealtimeMeasurements(deviceID)

	case "operations":
		fallthrough
	case "devicecontrol":
		channelPattern = c8y.RealtimeOperations(deviceID)
	default:
		// Custom channel as defined by the user
		channelPattern = *channel
	}

	client.Realtime.Subscribe(channelPattern, ch)

	// Enable ctrl-c stop signal
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)

	log.Printf("Listenening to subscriptions: %s", channelPattern)

	timeoutCh := time.After(timeoutDuration)

	for {
		select {
		case <-timeoutCh:
			log.Printf("Duration has expired. Stopping realtime client")
			return
		case msg := <-ch:
			if bytes.Contains(msg.Payload.Data, []byte(*series)) {
				if *channel == "measurements" && !*jsonOutput {
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
				} else {
					fmt.Printf("%s\n", msg.Payload.Data)
				}
			}

		case <-signalCh:
			// Enable ctrl-c to stop
			log.Printf("Stopping realtime client")
			return
		}
	}
}

func findDevice(client *c8y.Client, device string) (string, error) {
	if device == "" || device == "*" {
		return device, nil
	}

	// device is an id
	pattern := regexp.MustCompile("^\\d+$")
	if pattern.MatchString(device) {
		return device, nil
	}

	// lookup device by name
	devices, _, err := client.Inventory.GetDevicesByName(context.Background(), device, c8y.NewPaginationOptions(1))

	if err != nil {
		return "", fmt.Errorf("failed to send request to Cumulocity. '%s'", err)
	}

	if len(devices.ManagedObjects) == 0 {
		return "", fmt.Errorf("Could not find a device with the name '%s'", device)
	}

	return devices.ManagedObjects[0].ID, nil
}
