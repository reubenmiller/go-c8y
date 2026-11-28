package microservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
)

// SubscribeToOperations subscribes to operations added to the microservice's agent managed object. onMessageFunc is called on every operation
//
// Deprecated: Use SubscribeNotifications (Notification 2.0).
func (m *Microservice) SubscribeToOperations(user model.ServiceUser, onMessageFunc func(*realtime.Message)) error {
	return m.SubscribeToNotifications(
		user,
		realtime.Operations(m.AgentID),
		onMessageFunc,
	)
}

// SubscribeToNotifications subscribes to c8y notifications on the Microservice's agent managed object
// over the legacy CometD realtime API. The reader goroutine stops when the
// realtime client closes the subscription channel.
//
// Deprecated: Use SubscribeNotifications (Notification 2.0), which is
// tenant-aware and stops when its context is cancelled.
func (m *Microservice) SubscribeToNotifications(user model.ServiceUser, realtimeChannel string, onMessageFunc func(*realtime.Message)) error {
	realtimeClient, err := m.NewRealtimeClient(user)

	if err != nil {
		return errors.New("failed to retrieve valid realtime client")
	}

	if connErr := realtimeClient.Connect(); connErr != nil {
		return fmt.Errorf("failed to connect. %w", connErr)
	}
	ch := make(chan *realtime.Message)

	err = <-realtimeClient.Subscribe(context.Background(), realtimeChannel, ch)

	// The realtime client owns ch and closes it when the subscription ends,
	// which stops this goroutine.
	go func() {
		for msg := range ch {
			if onMessageFunc != nil {
				onMessageFunc(msg)
			}
		}
	}()
	return err
}
