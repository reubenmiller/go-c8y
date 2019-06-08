package c8y_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/internal/pkg/testingutils"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
)

func OperationSenderFactory(client *c8y.Client, deviceID string, t *testing.T) func() {
	return func() {
		_, _, err := client.Operation.Create(
			context.Background(),
			map[string]interface{}{
				"deviceId": deviceID,
				"test_operation": map[string]interface{}{
					"name": "test operation",
					"parameters": map[string]interface{}{
						"value1": 1,
					},
				},
			},
		)
		if err != nil {
			t.Errorf("Failed to create operation. %s", err)
		}
	}
}

func TestRealtimeClient(t *testing.T) {
	client := createTestClient()
	realtime := client.Realtime

	err := realtime.Connect()
	testingutils.Ok(t, err)
}

func TestRealtimeSubscriptions_SubscribeToOperations(t *testing.T) {
	device, err := createRandomTestDevice()

	if err != nil {
		t.Errorf("Device should exist. wanted nil, got %s", err)
	}

	client := createTestClient()
	realtime := client.Realtime
	err = realtime.Connect()
	testingutils.Ok(t, err)

	time.Sleep(5 * time.Second)

	if err != nil {
		t.Errorf("Unknown error")
	}

	ch := make(chan *c8y.Message)
	timerChan := time.NewTimer(time.Second * 5).C

	realtime.Subscribe(c8y.RealtimeOperations(device.ID), ch)
	time.Sleep(2 * time.Second)

	// Create a dummy operation
	sendOperation := func() {
		_, _, err = client.Operation.Create(
			context.Background(),
			map[string]interface{}{
				"deviceId": device.ID,
				"test_operation": map[string]interface{}{
					"name": "test operation",
					"parameters": map[string]interface{}{
						"value1": 1,
					},
				},
			},
		)
		if err != nil {
			t.Errorf("Failed to create operation. %s", err)
		}
	}

	sendOperation()
	sendOperation()

	defer func() {
		close(ch)
		realtime.Close()
	}()

	msgCount := 0
	expectedOpName := "test operation"

	for {
		select {
		case msg := <-ch:
			msgCount++

			if msg.Payload.RealtimeAction != "CREATE" {
				t.Errorf("Unexpected realtime action type. wanted: CREATE, got: %s", msg.Payload.RealtimeAction)
			}

			opName := msg.Payload.Item.Get("test_operation.name").String()
			if opName != expectedOpName {
				t.Errorf("Unexpected operation name. wanted: %s, got: %s", expectedOpName, opName)
			}

			deviceId := msg.Payload.Item.Get("deviceId").String()
			if deviceId != device.ID {
				t.Errorf("Unexpected device id in operation. wanted: %s, got: %s", device.ID, deviceId)
			}

			log.Printf("Received notification")
			log.Printf("ws: [frame]: Channel: %s, %s\n", msg.Channel, string(msg.Payload.Item.Raw))

		case <-timerChan:
			realtime.UnsubscribeAll()

			if msgCount != 2 {
				t.Errorf("Unexpected message count. wanted: 2, got: %d", msgCount)
			}
			realtime.Close()
			return
		}
	}
}

func TestRealtimeSubscriptions_SubscribeToMeasurements(t *testing.T) {
	device, err := createRandomTestDevice()

	if err != nil {
		t.Errorf("Device should exist. wanted nil, got %s", err)
	}

	client := createTestClient()
	realtime := client.Realtime

	err = realtime.Connect()
	testingutils.Ok(t, err)

	ch := make(chan *c8y.Message)

	realtime.Subscribe(c8y.RealtimeMeasurements(device.ID), ch)

	// Given time for CEP engine to work
	time.Sleep(2 * time.Second)

	// Create a dummy measurement
	sendMeasurement := func(value float64) {
		meas, err := c8y.NewSimpleMeasurementRepresentation(
			c8y.SimpleMeasurementOptions{
				SourceID:            device.ID,
				Type:                "testMeasurement1",
				ValueFragmentType:   "c8y_Test",
				ValueFragmentSeries: "Measurement1",
				Value:               value,
				Unit:                "Counter",
			},
		)

		if err != nil {
			t.Errorf("Failed to create measurement. %s", err)
		}
		_, _, err = client.Measurement.Create(
			context.Background(),
			*meas,
		)
		if err != nil {
			t.Errorf("Failed to create measurement. %s", err)
		}
	}

	expectedMeasValue := 1.0
	sendMeasurement(expectedMeasValue)
	sendMeasurement(expectedMeasValue)

	timerChan := time.NewTimer(time.Second * 5).C

	defer func() {
		close(ch)
		// realtime.Close()
	}()

	msgCount := 0

	/* go func() {
		select {
		case <- time.After(5 * time.Second):
			realtime.Disconnect()
		}
	}() */

	for {
		select {
		case msg := <-ch:
			msgCount++

			if msg.Payload.RealtimeAction != "CREATE" {
				t.Errorf("Unexpected realtime action type. wanted: CREATE, got: %s", msg.Payload.RealtimeAction)
			}

			measValue := msg.Payload.Item.Get("c8y_Test.Measurement1.value").Float()
			if measValue != expectedMeasValue {
				t.Errorf("Unexpected measurement value. wanted: %f, got: %f", expectedMeasValue, measValue)
			}

			deviceId := msg.Payload.Item.Get("source.id").String()
			if deviceId != device.ID {
				t.Errorf("Unexpected device id in operation. wanted: %s, got: %s", device.ID, deviceId)
			}

			log.Printf("ws: [frame]: Channel: %s, %s\n", msg.Channel, string(msg.Payload.Item.Raw))

		case <-timerChan:
			realtime.UnsubscribeAll()
			time.Sleep(5 * time.Second)
			realtime.Disconnect()

			testingutils.Equals(t, 2, msgCount)
			time.Sleep(10 * time.Second)
			// realtime.Close()
			return
		}
	}
}

func TestRealtimeSubscriptions_Unsubscribe(t *testing.T) {
	// Issue #2: https://github.com/reubenmiller/go-c8y/issues/2
	// A subscribe -> unsubscribe -> subscribe should not result in duplicate
	// items on the channel
	// https://www.ardanlabs.com/blog/2017/10/the-behavior-of-channels.html
	device, err := createRandomTestDevice()

	if err != nil {
		t.Errorf("Device should exist. wanted nil, got %s", err)
	}

	client := createTestClient()
	realtime := client.Realtime

	// Create a dummy operation
	sendOperation := OperationSenderFactory(client, device.ID, t)

	err = realtime.Connect()
	testingutils.Ok(t, err)

	ch := make(chan *c8y.Message)
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

				opName := msg.Payload.Item.Get("test_operation.name").String()
				if opName != expectedOpName {
					t.Errorf("Unexpected operation name. wanted: %s, got: %s", expectedOpName, opName)
				}

				deviceId := msg.Payload.Item.Get("deviceId").String()
				if deviceId != device.ID {
					t.Errorf("Unexpected device id in operation. wanted: %s, got: %s", device.ID, deviceId)
				}
				log.Printf("ws: [frame]: Channel: %s, %s\n", msg.Channel, string(msg.Payload.Item.Raw))

			case <-timerChan:
				// Stop subscribing to operations after x seconds, and then compare the result
				realtime.UnsubscribeAll()

				time.Sleep(2 * time.Second)

				log.Printf("Test: Closing realtime client")
				realtime.Close()
				done <- true
				return
			}
		}
	}()

	subcriptionPattern := c8y.RealtimeOperations(device.ID)

	err = <-realtime.Subscribe(subcriptionPattern, ch)
	testingutils.Ok(t, err)

	// Unsubscribe then resubscribe, this should not lead to duplicated messages
	err = <-realtime.Unsubscribe(subcriptionPattern)
	testingutils.Ok(t, err)

	sendOperation() // Subscription should not count as the sub

	time.Sleep(2 * time.Second)
	err = <-realtime.Subscribe(subcriptionPattern, ch)
	testingutils.Ok(t, err)

	sendOperation()
	sendOperation()

	defer func() {
		log.Printf("Test: Closing channel")
		close(ch)
	}()

	// Wait for done signal, then check the total message count
	<-done
	if msgCount != 2 {
		t.Errorf("Unexpected message count. wanted: 2, got: %d", msgCount)
	}
}
