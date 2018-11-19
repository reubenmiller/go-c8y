package c8y_test

import (
	"context"
	"testing"
)

func TestApplicationService_GetApplicationCollectionByName(t *testing.T) {
	client := createTestClient()

	exampleAppName := "cockpit"

	data, _, err := client.Application.GetApplicationCollectionByName(context.Background(), exampleAppName, nil)

	if err != nil {
		t.Errorf("Unexpected error. want: nil, got: %s", err)
	}
	minApplications := 1
	if len(data.Items) < minApplications {
		t.Errorf("Unexpected amount of applications found. want >=%d, got: %d", minApplications, len(data.Items))
	}

	actualAppName := data.Items[0].Get("name").String()
	if actualAppName != exampleAppName {
		t.Errorf("Wrong application name. want: %s, got: %s", exampleAppName, actualAppName)
	}
}
