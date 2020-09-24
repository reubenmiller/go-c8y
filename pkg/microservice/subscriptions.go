package microservice

import (
	"errors"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

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

	if connErr := realtime.Connect(); connErr != nil {
		return fmt.Errorf("Failed to connect. %s", connErr)
	}
	ch := make(chan *c8y.Message)

	err = <-realtime.Subscribe(realtimeChannel, ch)

	go func() {
		defer func() {
			close(ch)
		}()
		for {
			msg := <-ch
			if onMessageFunc != nil {
				onMessageFunc(msg)
			}
		}
	}()
	return err
}
