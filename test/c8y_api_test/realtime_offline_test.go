package api_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForRealtimeConn polls the fake server until a realtime websocket
// connection appears or the deadline elapses.
func waitForRealtimeConn(t *testing.T, srv *fakeserver.FakeServer, timeout time.Duration) *fakeserver.RealtimeConnection {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conns := srv.Realtime.Connections()
		if len(conns) > 0 {
			return conns[0]
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for realtime connection")
	return nil
}

func TestRealtimeOffline_Connect(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	rt := client.Realtime()

	err := rt.Connect()
	require.NoError(t, err)
	defer rt.Close()

	assert.True(t, rt.IsConnected())

	conn := waitForRealtimeConn(t, srv, 2*time.Second)
	assert.NotNil(t, conn)
}

func TestRealtimeOffline_SubscribeReceivePush(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	rt := client.Realtime()

	require.NoError(t, rt.Connect())
	defer rt.Close()

	ch := make(chan *realtime.Message, 4)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	subErr := <-rt.Subscribe(ctx, realtime.Operations("12345"), ch)
	require.NoError(t, subErr)

	// Allow subscription to register on the fake server.
	conn := waitForRealtimeConn(t, srv, 2*time.Second)

	// Wait until the subscription is registered.
	require.Eventually(t, func() bool {
		return slices.Contains(conn.Subscriptions(), "/operations/12345")
	}, 2*time.Second, 50*time.Millisecond)

	// Push a fake CREATE notification on the subscribed channel.
	require.NoError(t, conn.PushData("/operations/12345", "CREATE", map[string]any{
		"deviceId": "12345",
		"test_operation": map[string]any{
			"name": "test operation",
		},
	}))

	select {
	case msg := <-ch:
		assert.Equal(t, "/operations/12345", msg.Channel)
		assert.Equal(t, "CREATE", msg.Payload.RealtimeAction)
		assert.Equal(t, "12345", msg.Payload.Data.Get("deviceId").String())
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for realtime message")
	}

	// Unsubscribe.
	require.NoError(t, <-rt.Unsubscribe(realtime.Operations("12345")))
}

func TestRealtimeOffline_SubscribeStream(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	rt := client.Realtime()
	require.NoError(t, rt.Connect())
	defer rt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := client.Measurements.SubscribeStream(ctx, "55555")
	require.NoError(t, stream.Err)
	defer stream.Data.Close()

	conn := waitForRealtimeConn(t, srv, 2*time.Second)
	require.Eventually(t, func() bool {
		return slices.Contains(conn.Subscriptions(), "/measurements/55555")
	}, 2*time.Second, 50*time.Millisecond)

	require.NoError(t, conn.PushData("/measurements/55555", "CREATE", map[string]any{
		"source": map[string]any{"id": "55555"},
		"c8y_Test": map[string]any{
			"Measurement1": map[string]any{
				"value": 1.5,
				"unit":  "C",
			},
		},
	}))

	done := make(chan struct{})
	go func() {
		defer close(done)
		for item, err := range stream.Data.Items() {
			if err != nil {
				return
			}
			assert.Equal(t, "CREATE", item.Action)
			assert.Equal(t, "55555", item.Data.SourceID())
			return
		}
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for stream item")
	}
}

func TestRealtimeOffline_SubscribeWildcard(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	rt := client.Realtime()
	require.NoError(t, rt.Connect())
	defer rt.Close()

	ch := make(chan *realtime.Message, 4)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, <-rt.Subscribe(ctx, realtime.Events("*"), ch))

	conn := waitForRealtimeConn(t, srv, 2*time.Second)
	require.Eventually(t, func() bool {
		return slices.Contains(conn.Subscriptions(), "/events/*")
	}, 2*time.Second, 50*time.Millisecond)

	payload, _ := json.Marshal(map[string]string{"type": "ping"})
	require.NoError(t, conn.PushData("/events/12345", "CREATE", json.RawMessage(payload)))

	select {
	case msg := <-ch:
		assert.Equal(t, "/events/12345", msg.Channel)
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for wildcard message")
	}
}

func TestRealtimeOffline_ExplicitUnsubscribe(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	rt := client.Realtime()
	require.NoError(t, rt.Connect())
	defer rt.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := make(chan *realtime.Message, 4)
	require.NoError(t, <-rt.Subscribe(ctx, realtime.Alarms("99"), ch))

	conn := waitForRealtimeConn(t, srv, 2*time.Second)
	require.Eventually(t, func() bool {
		return slices.Contains(conn.Subscriptions(), "/alarms/99")
	}, 2*time.Second, 50*time.Millisecond)

	require.NoError(t, <-rt.Unsubscribe(realtime.Alarms("99")))

	require.Eventually(t, func() bool {
		return !slices.Contains(conn.Subscriptions(), "/alarms/99")
	}, 3*time.Second, 50*time.Millisecond)
}

func TestRealtimeOffline_UnsubscribeAll(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)
	rt := client.Realtime()
	require.NoError(t, rt.Connect())
	defer rt.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ch1 := make(chan *realtime.Message, 4)
	ch2 := make(chan *realtime.Message, 4)
	require.NoError(t, <-rt.Subscribe(ctx, realtime.Operations("aa"), ch1))
	require.NoError(t, <-rt.Subscribe(ctx, realtime.Operations("bb"), ch2))

	conn := waitForRealtimeConn(t, srv, 2*time.Second)
	require.Eventually(t, func() bool {
		s := conn.Subscriptions()
		return slices.Contains(s, "/operations/aa") && slices.Contains(s, "/operations/bb")
	}, 2*time.Second, 50*time.Millisecond)

	require.NoError(t, <-rt.UnsubscribeAll())

	require.Eventually(t, func() bool {
		return len(conn.Subscriptions()) == 0
	}, 3*time.Second, 50*time.Millisecond)
}

func TestRealtimeOffline_PackageSelectorHelpers(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	assert.Equal(t, "/alarms/1", realtime.Alarms("1"))
	assert.Equal(t, "/alarms/*", realtime.Alarms())
	assert.Equal(t, "/alarmsWithChildren/1", realtime.AlarmsWithChildren("1"))
	assert.Equal(t, "/events/2", realtime.Events("2"))
	assert.Equal(t, "/managedobjects/3", realtime.ManagedObjects("3"))
	assert.Equal(t, "/measurements/4", realtime.Measurements("4"))
	assert.Equal(t, "/operations/5", realtime.Operations("5"))
}
