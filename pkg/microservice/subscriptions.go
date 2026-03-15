package microservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
)

// SubscribeToOperations subscribes to operations added to the microservice's agent managed object. onMessageFunc is called on every operation
func (m *Microservice) SubscribeToOperations(user model.ServiceUser, onMessageFunc func(*realtime.Message)) error {
	return m.SubscribeToNotifications(
		user,
		realtime.Operations(m.AgentID),
		onMessageFunc,
	)
}

// SubscribeToNotifications subscribes to c8y notifications on the Microservice's agent managed object
func (m *Microservice) SubscribeToNotifications(user model.ServiceUser, realtimeChannel string, onMessageFunc func(*realtime.Message)) error {
	realtimeClient, err := m.NewRealtimeClient(user)

	if err != nil {
		return errors.New("Failed to retrieve valid realtime client")
	}

	if connErr := realtimeClient.Connect(); connErr != nil {
		return fmt.Errorf("Failed to connect. %s", connErr)
	}
	ch := make(chan *realtime.Message)

	err = <-realtimeClient.Subscribe(context.Background(), realtimeChannel, ch)

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
