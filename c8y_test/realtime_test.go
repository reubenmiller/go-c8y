package c8y_test

import (
	"context"
	"log"
	"testing"
	"time"

	c8y "github.com/reubenmiller/go-c8y"
)

func TestRealtimeClient(t *testing.T) {
	client := createTestClient()
	realtime := client.Realtime

	err := realtime.Connect()

	err = realtime.WaitForConnection()

	if err != nil {
		t.Errorf("Unknown error")
	}
}

func TestRealtimeSubscriptions(t *testing.T) {
	_, err := createTestDevice()

	if err != nil {
		t.Errorf("Device should exist. wanted nil, got %s", err)
	}

	client := createTestClient()
	realtime := client.Realtime

	go func() {
		realtime.Connect()
	}()

	err = realtime.WaitForConnection()

	if err != nil {
		t.Errorf("Unknown error")
	}

	ch := make(chan *c8y.Message)
	tickChan := time.NewTicker(time.Millisecond * 5000).C
	timerChan := time.NewTimer(time.Second * 70).C

	realtime.Subscribe("/operations/7708558328", ch)
	// realtime.Subscribe("/measurements/7858248941", ch)
	// _ = m.Client.Realtime.Subscribe(fmt.Sprintf("/operations/%s", m.AgentID), ch)
	defer func() {
		close(ch)
		realtime.Close()
	}()

	for {
		select {
		case msg := <-ch:
			log.Printf("Received notification")
			log.Printf("ws: [frame]: Channel: %s, %s\n", msg.Channel, string(msg.Payload.Item.Raw))

		case <-tickChan:
			if realtime.IsConnected() {
				log.Printf("[WatchDog]: Websocket is already connected")
			} else {
				log.Printf("[WatchDog]: Websocket has been closed...reconnecting")
				// realtime.Connect()
			}

		case <-timerChan:
			log.Printf("[Unsubscribe]")
			// realtime.UnsubscribeAll()
		}
	}
}

func TestRealtimeSubscriptions_SubscribeToOperations(t *testing.T) {
	device, err := createTestDevice()

	if err != nil {
		t.Errorf("Device should exist. wanted nil, got %s", err)
	}

	client := createTestClient()
	realtime := client.Realtime

	go func() {
		realtime.Connect()
	}()

	err = realtime.WaitForConnection()

	if err != nil {
		t.Errorf("Unknown error")
	}

	ch := make(chan *c8y.Message)
	timerChan := time.NewTimer(time.Second * 5).C

	realtime.Subscribe("/operations/"+device.ID, ch)

	// Create a dummy operation
	sendOperation := func() {
		_, _, err = client.Operation.CreateOperation(
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
	device, err := createTestDevice()

	if err != nil {
		t.Errorf("Device should exist. wanted nil, got %s", err)
	}

	client := createTestClient()
	realtime := client.Realtime

	go func() {
		realtime.Connect()
	}()

	err = realtime.WaitForConnection()

	if err != nil {
		t.Errorf("Unknown error")
	}

	ch := make(chan *c8y.Message)
	timerChan := time.NewTimer(time.Second * 5).C

	realtime.Subscribe("/measurements/"+device.ID, ch)

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

	defer func() {
		close(ch)
		realtime.Close()
	}()

	msgCount := 0

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
