package c8y_api_test

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"
	"github.com/reubenmiller/go-c8y/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/measurements"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/operations"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/realtime"
)

func OperationSenderFactory(client *c8y_api.Client, deviceID string, t *testing.T) func() {
	return func() {
		result := client.Operations.Create(
			context.Background(),
			operations.CreateOptions{
				DeviceID:    deviceID,
				Description: "Test operation",
				AdditionalProperties: map[string]any{
					"test_operation": map[string]any{
						"name": "test operation",
						"parameters": map[string]any{
							"value1": 1,
						},
					},
				},
			},
		)
		require.NoError(t, result.Err)
	}
}

func TestRealtimeClient(t *testing.T) {
	client := testcore.CreateTestClient(t)
	err := client.Realtime().Connect()
	assert.NoError(t, err)
}

func TestRealtimeSubscriptions_SubscribeToOperations(t *testing.T) {
	client := testcore.CreateTestClient(t)
	device := testcore.CreateDeviceAgent(t, client)

	realtime := client.Realtime()
	connectErr := realtime.Connect()
	assert.NoError(t, connectErr)

	time.Sleep(5 * time.Second)

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream := client.Operations.SubscribeStream(ctxTimeout, device.Data.ID())
	require.NoError(t, stream.Err)
	defer stream.Data.Close()

	time.Sleep(2 * time.Second)

	sendOperation := OperationSenderFactory(client, device.Data.ID(), t)
	sendOperation()
	sendOperation()

	msgCount := 0
	expectedOpName := "test operation"

	for item, err := range stream.Data.Items() {
		if err != nil {
			break
		}
		msgCount++

		assert.Equal(t, "CREATE", item.Action, "Unexpected realtime action type")

		opName := item.Data.Get("test_operation.name").String()
		assert.Equal(t, expectedOpName, opName)

		assert.Equal(t, device.Data.ID(), item.Data.DeviceID())

		slog.Info("Received notification")
		slog.Info("ws: [frame]", "chanel", item.Channel, "payload", item.Data.Bytes())

		if msgCount >= 2 {
			break
		}
	}

	assert.Equal(t, msgCount, 2)
}

func TestRealtimeSubscriptions_SubscribeToMeasurements(t *testing.T) {
	client := testcore.CreateTestClient(t)
	device := testcore.CreateDevice(t, client)

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream := client.Measurements.SubscribeStream(ctxTimeout, device.Data.ID())
	require.NoError(t, stream.Err)

	// Given time for CEP engine to work
	time.Sleep(2 * time.Second)

	// Create a dummy measurement
	sendMeasurement := func(value float64) {
		result := client.Measurements.Create(
			context.Background(),
			measurements.CreateOptions{
				Source: device.Data.ID(),
				Type:   "testMeasurement1",
				Time:   time.Now(),
				AdditionalProperties: map[string]any{
					"c8y_Test": map[string]any{
						"Measurement1": map[string]any{
							"value": value,
							"unit":  "Counter",
						},
					},
				},
			},
		)
		require.NoError(t, result.Err, "Failed to create measurement")
	}

	expectedMeasValue := 1.0
	sendMeasurement(expectedMeasValue)
	sendMeasurement(expectedMeasValue)

	msgCount := 0
	for item, err := range stream.Data.Items() {
		if err != nil {
			break
		}
		msgCount++

		assert.Equal(t, "CREATE", item.Action, "Unexpected realtime action type")

		measValue := item.Data.Get("c8y_Test.Measurement1.value").Float()
		assert.EqualValues(t, expectedMeasValue, measValue, "Unexpected measurement value")

		assert.Equal(t, device.Data.ID(), item.Data.SourceID(), "source.id should match")

		slog.Info("ws: [frame]", "channel", item.Channel, "payload", item.Data.Bytes())
	}

}

func TestRealtimeSubscriptions_Unsubscribe(t *testing.T) {
	// Issue #2: https://github.com/reubenmiller/go-c8y/issues/2
	// A subscribe -> unsubscribe -> subscribe should not result in duplicate
	// items on the channel
	// https://www.ardanlabs.com/blog/2017/10/the-behavior-of-channels.html
	client := testcore.CreateTestClient(t)
	device := testcore.CreateDeviceAgent(t, client)

	realtimeC := client.Realtime()

	// Create a dummy operation
	sendOperation := OperationSenderFactory(client, device.Data.ID(), t)

	err := realtimeC.Connect()
	require.NoError(t, err)

	ch := make(chan *realtime.Message)
	timerChan := time.NewTimer(time.Second * 15).C

	done := make(chan bool)
	msgCount := 0

	go func() {
		expectedOpName := "test operation"
		for {
			select {
			case msg := <-ch:
				msgCount++

				if msg.Payload.RealtimeAction != "CREATE" {
					t.Errorf("Unexpected realtime action type. wanted: CREATE, got: %s", msg.Payload.RealtimeAction)
				}

				opName := msg.Payload.Data.Get("test_operation.name").String()
				if opName != expectedOpName {
					t.Errorf("Unexpected operation name. wanted: %s, got: %s", expectedOpName, opName)
				}

				deviceId := msg.Payload.Data.Get("deviceId").String()
				if deviceId != device.Data.ID() {
					t.Errorf("Unexpected device id in operation. wanted: %s, got: %s", device.Data.ID(), deviceId)
				}
				slog.Info("ws: [frame]", "channel", msg.Channel, "message", string(msg.Payload.Data.Bytes()))

			case <-timerChan:
				// Stop subscribing to operations after x seconds, and then compare the result
				realtimeC.UnsubscribeAll()

				time.Sleep(2 * time.Second)

				slog.Info("Test: Closing realtime client")
				realtimeC.Close()
				done <- true
				return
			}
		}
	}()

	subscriptionPattern := realtime.Operations(device.Data.ID())

	err = <-realtimeC.Subscribe(context.Background(), subscriptionPattern, ch)
	assert.NoError(t, err)

	// Unsubscribe then resubscribe, this should not lead to duplicated messages
	err = <-realtimeC.Unsubscribe(subscriptionPattern)
	assert.NoError(t, err)

	sendOperation() // Subscription should not count as the sub

	time.Sleep(2 * time.Second)
	err = <-realtimeC.Subscribe(context.Background(), subscriptionPattern, ch)
	testingutils.Ok(t, err)

	sendOperation()
	sendOperation()

	defer func() {
		slog.Info("Test: Closing channel")
		close(ch)
	}()

	// Wait for done signal, then check the total message count
	<-done
	if msgCount != 2 {
		t.Errorf("Unexpected message count. wanted: 2, got: %d", msgCount)
	}
}
