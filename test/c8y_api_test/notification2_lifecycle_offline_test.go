package api_test

import (
	"context"
	"encoding/json"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	notif "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/notification2"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/notification2"
	"github.com/reubenmiller/go-c8y/v2/pkg/microservice"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func pushTestMessage(t *testing.T, conn interface {
	PushMessage(identifier, description, action string, payload []byte) error
}, id string) {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{"id": id})
	require.NoError(t, conn.PushMessage("msg-"+id, "/t12345/measurements/"+id, "CREATE", payload))
}

// SubscribeStream delivers messages (it used to register a channel that was
// never read).
func TestNotification2Offline_SubscribeStreamDelivers(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res := client.Notification2.SubscribeStream(ctx, notif.ClientOptions{
		Consumer: "streamConsumer",
		Options:  notif.TokenOptions{Subscription: "streamSub"},
	})
	require.NoError(t, res.Err)
	stream := res.Data
	defer stream.Close()

	conn := srv.Notification2.WaitForConnection(2 * time.Second)
	require.NotNil(t, conn)
	pushTestMessage(t, conn, "1")

	for msg, err := range stream.Items() {
		require.NoError(t, err)
		assert.Equal(t, "CREATE", msg.Action)
		assert.Equal(t, "1", msg.Data.Get("id").String())
		break
	}
}

func TestNotification2Offline_SubscribeNotificationsPerTenant(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	ms := &microservice.Microservice{Client: client}
	ms.SetServiceUsers([]model.ServiceUser{{Tenant: "t12345", Username: "service_app", Password: "pw"}})

	// The client sends no request until it is used (NewClient does not log in),
	// so open its keep-alive connection first: its goroutines are not a leak.
	_ = client.Tenants.Current.Get(context.Background(), currenttenant.GetOptions{})
	baseline := runtime.NumGoroutine()

	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	var got []string
	done := make(chan error, 1)
	go func() {
		done <- ms.SubscribeNotifications(ctx, "t12345", notif.ClientOptions{
			Consumer: "tenantConsumer",
			Options:  notif.TokenOptions{Subscription: "tenantSub"},
		}, func(ctx context.Context, msg microservice.NotificationMessage) error {
			mu.Lock()
			got = append(got, msg.Data.Get("id").String())
			mu.Unlock()
			return nil
		})
	}()
	received := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}

	conn := srv.Notification2.WaitForConnection(3 * time.Second)
	require.NotNil(t, conn)
	for _, id := range []string{"1", "2", "3"} {
		pushTestMessage(t, conn, id)
	}
	require.Eventually(t, func() bool { return len(received()) == 3 }, 3*time.Second, 10*time.Millisecond)
	assert.Equal(t, []string{"1", "2", "3"}, received(), "messages are delivered in order")
	require.Eventually(t, func() bool { return len(conn.Acks()) == 3 }, 3*time.Second, 10*time.Millisecond, "handled messages are acknowledged")

	// The server drops the connection; the client reconnects (after
	// notification2.MinimumRetryInterval, 5s) and keeps delivering.
	conn.Close()
	reconnected := srv.Notification2.WaitForConnection(time.Duration(notification2.MinimumRetryInterval+5) * time.Second)
	require.NotNil(t, reconnected, "the client reconnects after the connection drops")
	pushTestMessage(t, reconnected, "4")
	require.Eventually(t, func() bool { return len(received()) == 4 }, 3*time.Second, 10*time.Millisecond)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("SubscribeNotifications did not return after cancellation")
	}
	assert.Eventually(t, func() bool { return runtime.NumGoroutine() <= baseline+2 }, 3*time.Second, 50*time.Millisecond,
		"no client goroutines are left running (baseline %d, now %d)", baseline, runtime.NumGoroutine())
}
