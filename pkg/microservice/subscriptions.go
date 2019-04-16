package microservice

import (
	"errors"
	"log"
	"regexp"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/tidwall/gjson"
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

// SubscribeToOperations subscribes to operations added to the microservice's agent managed object. onMessageFunc is called on every operation
func (m *Microservice) SubscribeToOperations(user c8y.ServiceUser, onMessageFunc func(*c8y.Message)) error {
	return m.SubscribeToNotifications(
		user,
		c8y.RealtimeOperations(m.AgentID),
		onMessageFunc,
	)
}

// SubscribeToNotifications subscribes to c8y notifications on the Microservice's agent managed object
func (m *Microservice) SubscribeToNotifications(user c8y.ServiceUser, realtimeChannel string, onMessageFunc func(*c8y.Message)) error {
	realtime, err := m.NewRealtimeClient(user)

	if err != nil {
		return errors.New("Failed to retrieve valid realtime client")
	}

	realtime.Connect()
	realtime.WaitForConnection()
	ch := make(chan *c8y.Message)

	err = realtime.Subscribe(realtimeChannel, ch)

	go func() {
		defer func() {
			close(ch)
		}()
		for {
			select {
			case msg := <-ch:
				// zap.S().Infof("ws: [frame]: %s\n", string(msg.Payload.Item.Raw))
				if onMessageFunc != nil {
					onMessageFunc(msg)
				}
			}

		}
	}()
	return err
}
