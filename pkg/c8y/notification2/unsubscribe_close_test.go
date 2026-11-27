package notification2

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Closing the client right after Unsubscribe must deliver the message to the
// server, and must not log a warning about a message that could not be sent
// because the connection was already closed.
func Test_UnsubscribeThenClose_SendsMessageWithoutWarning(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	defer slog.SetDefault(previous)

	for i := 0; i < 30; i++ {
		received := make(chan string, 4)
		var wg sync.WaitGroup
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			up := websocket.Upgrader{}
			conn, err := up.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer conn.Close()
				for {
					_, msg, err := conn.ReadMessage()
					if err != nil {
						return
					}
					received <- string(msg)
				}
			}()
		}))

		c, err := NewNotification2ClientE(srv.URL, nil, Subscription{Token: "t"}, ConnectionOptions{})
		require.NoError(t, err)
		require.NoError(t, c.ConnectContext(context.Background()))

		require.NoError(t, c.Unsubscribe())
		require.NoError(t, c.Close())

		select {
		case msg := <-received:
			assert.Equal(t, "unsubscribe_subscriber", msg)
		case <-time.After(5 * time.Second):
			t.Fatal("the server did not receive the unsubscribe message")
		}
		srv.Close()
		wg.Wait()
	}
	assert.False(t, strings.Contains(logs.String(), "Failed to send message"), logs.String())
}
