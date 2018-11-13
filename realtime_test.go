package c8y

import (
	"os"
	"testing"
)

func TestRealtimeClient(t *testing.T) {
	host := os.Getenv("C8Y_HOST")
	tenant := os.Getenv("C8Y_TENANT")
	username := os.Getenv("C8Y_USERNAME")
	password := os.Getenv("C8Y_PASSWORD")

	if tenant == "" || username == "" || password == "" {
		t.Errorf("Missing Cumulocity C8Y_TENANT, C8Y_USERNAME, C8Y_PASSWORD environement variable which are required for this test")
	}
	client := NewRealtimeClient(host, nil, tenant, username, password)

	err := client.Connect()

	err = client.WaitForConnection()

	if err != nil {
		t.Errorf("Unknown error")
	}
}
