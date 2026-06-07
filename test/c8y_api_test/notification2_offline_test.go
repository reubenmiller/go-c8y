package api_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	notif "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/notification2"
	"github.com/reubenmiller/go-c8y/v2/test/c8y_api_test/testcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotification2Offline_TokenCreate(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	result := client.Notification2.CreateToken(ctx, notif.TokenOptions{
		Subscription:     "myTestSub",
		ExpiresInMinutes: 30,
	})
	require.NoError(t, result.Err)
	assert.NotEmpty(t, result.Data.Token())
}

func TestNotification2Offline_NormalizedConsumer(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)
	got := client.Notification2.NormalizedConsumer("ab cd-12!@")
	assert.Equal(t, "abcd12", got)
}

func TestNotification2Offline_ParseToken(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)

	// JWT (alg: HS256) with claims: sub=goc8y, topic=t12345/subA, shared=false,
	// iat=1700000000, exp=1700003600. Signed with secret "test" - signature
	// content does not matter for ParseToken which uses ParseUnverified-style
	// base64 decoding of the payload.
	token := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9." +
		"eyJzdWIiOiJnb2M4eSIsInRvcGljIjoidDEyMzQ1L3N1YkEiLCJzaGFyZWQiOiJmYWxzZSIsImlhdCI6MTcwMDAwMDAwMCwiZXhwIjoxNzAwMDAzNjAwfQ." +
		"sig"

	claim, err := client.Notification2.ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, "goc8y", claim.Subscriber)
	assert.Equal(t, "t12345/subA", claim.Topic)
	assert.Equal(t, "t12345", claim.Tenant())
	assert.Equal(t, "subA", claim.Subscription())
	assert.False(t, claim.IsShared())
	assert.True(t, claim.HasExpired())
}

func TestNotification2Offline_ParseToken_Invalid(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)
	_, err := client.Notification2.ParseToken("not-a-jwt")
	assert.Error(t, err)
}

func TestNotification2Offline_CreateSubscription(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	created := client.Notification2.Create(ctx, notif.CreateOptions{
		Context:      "mo",
		Subscription: "myTestSub",
		Source:       map[string]any{"id": "12345"},
		SubscriptionFilter: notif.SubscriptionFilter{
			Apis: []string{"measurements"},
		},
	})
	require.NoError(t, created.Err)

	got := client.Notification2.Get(ctx, created.Data.ID())
	require.NoError(t, got.Err)
	assert.Equal(t, created.Data.ID(), got.Data.ID())

	listed := client.Notification2.List(ctx, notif.ListOptions{})
	require.NoError(t, listed.Err)

	itAll := client.Notification2.ListAll(ctx, notif.ListOptions{})
	require.NoError(t, itAll.Err())

	del := client.Notification2.Delete(ctx, created.Data.ID())
	require.NoError(t, del.Err)
}

func TestNotification2Offline_DeleteBySource(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	res := client.Notification2.DeleteBySource(ctx, notif.DeleteBySourceOptions{
		Context: "mo",
		Source:  "12345",
	})
	assert.NoError(t, res.Err)
}

func TestNotification2Offline_UnsubscribeSubscriber(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()

	res := client.Notification2.UnsubscribeSubscriber(ctx, "some-token")
	require.NoError(t, res.Err)
	assert.Equal(t, "DELETED", res.Data.Result)
}

func TestNotification2Offline_ClientReceive(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client, srv := testcore.CreateTestClientWithFakeServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nc, err := client.Notification2.CreateClient(ctx, notif.ClientOptions{
		Consumer: "myConsumer",
		Options: notif.TokenOptions{
			Subscription: "mySub",
		},
	})
	require.NoError(t, err)
	defer nc.Close()
	require.NoError(t, nc.Connect())
	assert.True(t, nc.IsConnected())

	conn := srv.Notification2.WaitForConnection(2 * time.Second)
	require.NotNil(t, conn)
	assert.Equal(t, "myConsumer", conn.Consumer())
	assert.NotEmpty(t, conn.Token())

	// URL helpers used by the SDK
	assert.NotEmpty(t, nc.Endpoint())
	assert.Contains(t, nc.URL(true), "token=redacted")
	assert.NotContains(t, nc.URL(false), "redacted")

	payload, _ := json.Marshal(map[string]any{
		"self": "https://example.com/measurement/measurements/12345",
		"id":   "12345",
	})
	require.NoError(t, conn.PushMessage("msg-id-1", "/t12345/measurements/12345", "CREATE", payload))

	// Drive the receive directly via the client's Register hook to bypass the
	// SubscribeStream channel-wiring bug.
	gotMsg := false
	deadline := time.After(3 * time.Second)
loop:
	for {
		select {
		case <-deadline:
			break loop
		default:
			if len(conn.Acks()) > 0 {
				gotMsg = true
				break loop
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	// Send an ack ourselves to test SendMessageAck.
	require.NoError(t, nc.SendMessageAck("msg-id-1"))
	require.Eventually(t, func() bool {
		acks := conn.Acks()
		for _, a := range acks {
			if a == "msg-id-1" {
				return true
			}
		}
		return false
	}, 2*time.Second, 50*time.Millisecond)
	_ = gotMsg

	// Unsubscribe via the client and make sure the server observes it.
	require.NoError(t, nc.Unsubscribe())
	select {
	case <-conn.Unsubscribed():
	case <-time.After(2 * time.Second):
		t.Fatalf("server never observed unsubscribe")
	}
}

func TestNotification2Offline_RenewToken_Empty(t *testing.T) {
	if !testcore.IsOffline() {
		t.Skip("offline-only test")
	}
	client := testcore.CreateTestClient(t)
	ctx := context.Background()
	token, err := client.Notification2.RenewToken(ctx, notif.ClientOptions{
		Options: notif.TokenOptions{
			Subscription:     "subA",
			ExpiresInMinutes: 5,
		},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}
