package c8y_test

import (
	"testing"
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
