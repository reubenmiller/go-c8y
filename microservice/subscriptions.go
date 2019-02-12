package microservice

import (
	"fmt"
	"log"
	"regexp"
	"time"

	c8y "github.com/reubenmiller/go-c8y"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

// doSomethingWithData is an example func which can be used when calling the subscribe to measurements
func doSomethingWithData(dataStr string) {
	log.Printf("debug: %s\n", dataStr)

	fields := gjson.GetMany(dataStr, "data.*.*.value", "data.*.*.unit", "data.source.id")

	result := gjson.Get(dataStr, "data")
	pattern := regexp.MustCompile("^nx_.+_.+")
	var fragmentKey string
	var valueKey []string

	result.ForEach(func(key, value gjson.Result) bool {
		if pattern.MatchString(key.String()) {
			value.ForEach(func(key, value gjson.Result) bool {
				if true {
					valueKey = append(valueKey, key.String())
				}
				return true
			})
			fragmentKey = key.String()
			return false
		}
		return true
	})

	if len(valueKey) > 0 {
		log.Printf("data point: [%s] %s.%s=%s\n", fields[2], fragmentKey, valueKey[0], fields[0].String())
	}
}

// SubscribeToMeasurements subscribes to c8y measurements for the given device. ID can be a * to include measurements from all devices.
func (m *Microservice) SubscribeToMeasurements(ID string, onMessageFunc func(*c8y.Message) error) {
	client := m.Client
	time.Sleep(1 * time.Second)
	go func() {
		client.Realtime.Connect()
	}()

	client.Realtime.WaitForConnection()
	ch := make(chan *c8y.Message)

	_ = client.Realtime.Subscribe(fmt.Sprintf("/measurements/%s", ID), ch)

	go func() {
		defer func() {
			close(ch)
			client.Realtime.Close()
		}()
		for {
			select {
			case msg := <-ch:
				zap.S().Infof("ws: [frame]: %s\n", string(msg.Data))
				if onMessageFunc != nil {
					fmt.Println("calling func")
					onMessageFunc(msg)
				}
			}

		}
	}()
}
