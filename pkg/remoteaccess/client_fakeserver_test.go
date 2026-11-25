package remoteaccess

import (
	"bytes"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"log/slog"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
)

// A local TCP connection is proxied over the remote access websocket (the
// fake server's device side echoes every message). The client logs to the
// logger of the Cumulocity client.
func TestRemoteAccessClientProxiesTCP(t *testing.T) {
	fs := fakeserver.New(t)
	configID, _ := fs.RemoteAccessConfigs.Create([]byte(`{"name":"ssh","protocol":"PASSTHROUGH","hostname":"127.0.0.1","port":22}`), "")

	var logs syncBuffer
	client := api.NewClient(api.ClientOptions{
		Logger:  slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
		BaseURL: fs.URL(),
		Auth:    authentication.AuthOptions{Tenant: "t12345", Username: "admin", Password: "admin-pass"},
	})
	ra := NewRemoteAccessClient(client, RemoteAccessOptions{ManagedObjectID: "12345", RemoteAccessID: configID})
	if err := ra.Listen("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	go func() { _ = ra.Serve() }()

	conn, err := net.DialTimeout("tcp", ra.GetListenerAddress(), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := conn.Write([]byte("hello device")); err != nil {
		t.Fatal(err)
	}
	got := make([]byte, len("hello device"))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello device" {
		t.Errorf("echo = %q", got)
	}
	if !strings.Contains(logs.String(), "Connecting to Cumulocity IoT") {
		t.Errorf("the client's logger got no remote access messages: %q", logs.String())
	}
}

// syncBuffer is a bytes.Buffer safe for the concurrent writes of the proxy
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
