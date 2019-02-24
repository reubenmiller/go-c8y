package c8y_test

import (
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
	client := createTestClient()
	realtime := client.Realtime

	// err := realtime.Connect()
	var err error

	go func() {
		realtime.Connect()
	}()

	// m.Client.Realtime.WaitForConnection()

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
			log.Printf("ws: [frame]: Channel: %s, %s\n", msg.Channel, string(msg.Data))

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
