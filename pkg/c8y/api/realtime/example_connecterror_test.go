package realtime_test

import (
	"errors"
	"fmt"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
)

func ExampleConnectError() {
	// An error returned by Connect or Subscribe when the websocket cannot be
	// opened. Recognise it with errors.As; do not match its text.
	err := fmt.Errorf("subscribe: %w", &realtime.ConnectError{
		URL: "wss://tenant.example.com/notification/realtime",
		Err: errors.New("connection refused"),
	})

	var connErr *realtime.ConnectError
	if errors.As(err, &connErr) {
		fmt.Println(connErr.URL)
		fmt.Println(connErr.Err)
	}
	// Output:
	// wss://tenant.example.com/notification/realtime
	// connection refused
}
