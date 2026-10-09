package remoteaccess

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"github.com/multiformats/go-multistream"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/wsconnadapter"
)

// fakeRemoteAccess emulates the Cumulocity remote access endpoint and the device side
// (thin-edge.io remote access plugin) forwarding to an echo server
type fakeRemoteAccess struct {
	server        *httptest.Server
	websockets    atomic.Int32
	supportsMux   bool
	echoServerURL string

	// dropNegotiations is the number of multiplexing requests to drop without answering
	dropNegotiations atomic.Int32
	// silent never answers multiplexing requests (e.g. a target which waits for more data)
	silent bool
}

func newFakeRemoteAccess(t *testing.T, supportsMux bool) *fakeRemoteAccess {
	t.Helper()
	echo := echoServer(t)
	f := &fakeRemoteAccess{supportsMux: supportsMux, echoServerURL: echo}
	upgrader := websocket.Upgrader{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		f.websockets.Add(1)
		go f.device(wsConn)
	}))
	t.Cleanup(f.server.Close)
	return f
}

// multistreamHeader is the first data of a client requesting multiplexing
var multistreamHeader = []byte("\x13/multistream/1.0.0\n")

// device behaves like the remote access plugin: in-band negotiation (multistream-select), then
// either a yamux session (multiplexing) or a plain passthrough to the target
func (f *fakeRemoteAccess) device(wsConn *websocket.Conn) {
	conn := wsconnadapter.New(wsConn)
	defer conn.Close()

	first := make([]byte, 1024)
	n, err := conn.Read(first)
	if err != nil {
		return
	}
	first = first[:n]

	if bytes.HasPrefix(first, multistreamHeader) {
		if f.dropNegotiations.Add(-1) >= 0 {
			// e.g. the websocket failing before the device answered
			return
		}
		if f.silent {
			_, _ = io.Copy(io.Discard, conn)
			return
		}
		if !f.supportsMux {
			// an older plugin forwards the negotiation to the (HTTP) target, which rejects it
			_, _ = conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
			return
		}
		mux := multistream.NewMultistreamMuxer[string]()
		mux.AddHandler(MultiplexProtocol, nil)
		if _, _, err := mux.Negotiate(&replayConn{Reader: io.MultiReader(bytes.NewReader(first), conn), Conn: conn}); err != nil {
			return
		}
		session, err := yamux.Server(conn, yamuxTestConfig())
		if err != nil {
			return
		}
		for {
			stream, err := session.AcceptStream()
			if err != nil {
				return
			}
			target, err := net.Dial("tcp", f.echoServerURL)
			if err != nil {
				stream.Close()
				continue
			}
			go pipe(target, stream, multiplexHalfCloseTimeout)
		}
	}

	// passthrough
	target, err := net.Dial("tcp", f.echoServerURL)
	if err != nil {
		return
	}
	defer target.Close()
	_, _ = target.Write(first)
	go func() { _, _ = io.Copy(target, conn) }()
	_, _ = io.Copy(conn, target)
}

// replayConn replays data that has already been read from the connection
type replayConn struct {
	io.Reader
	net.Conn
}

func (c *replayConn) Read(p []byte) (int, error) { return c.Reader.Read(p) }

func yamuxTestConfig() *yamux.Config {
	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard
	return config
}

func echoServer(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				_, _ = io.Copy(conn, conn)
			}()
		}
	}()
	return listener.Addr().String()
}

func newClient(t *testing.T, fake *fakeRemoteAccess, opts RemoteAccessOptions) *RemoteAccessClient {
	t.Helper()
	client := c8y.NewClient(nil, fake.server.URL, "t12345", "user", "password", true)
	opts.ManagedObjectID = "1234"
	opts.RemoteAccessID = "1"
	ra := NewRemoteAccessClient(client, opts)
	if err := ra.Listen("127.0.0.1:0"); err != nil {
		t.Fatal(err)
	}
	return ra
}

func startClient(t *testing.T, fake *fakeRemoteAccess, multiplex bool) string {
	t.Helper()
	ra := newClient(t, fake, RemoteAccessOptions{Multiplex: multiplex})
	go func() { _ = ra.Serve() }()
	return ra.GetListenerAddress()
}

// exchange opens a new local connection, sends a message and reads the echo. With halfClose, the
// local side closes its writing direction first and the reply must still arrive (multiplexing only:
// the per-connection mode closes both directions when the local side is done).
func exchange(t *testing.T, address string, message string, halfClose bool) {
	t.Helper()
	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write([]byte(message)); err != nil {
		t.Fatal(err)
	}
	reply := make([]byte, len(message))
	if halfClose {
		_ = conn.(*net.TCPConn).CloseWrite()
		all, err := io.ReadAll(conn)
		if err != nil {
			t.Fatalf("reading reply: %v", err)
		}
		reply = all
	} else if _, err := io.ReadFull(conn, reply); err != nil {
		t.Fatalf("reading reply: %v", err)
	}
	if string(reply) != message {
		t.Fatalf("unexpected reply %q, want %q", reply, message)
	}
}

func TestMultiplexingUsesASingleRemoteAccessSession(t *testing.T) {
	fake := newFakeRemoteAccess(t, true)
	address := startClient(t, fake, true)

	for i := 0; i < 5; i++ {
		exchange(t, address, fmt.Sprintf("hello %d", i), i%2 == 0)
	}

	if got := fake.websockets.Load(); got != 1 {
		t.Fatalf("expected 1 remote access websocket, got %d", got)
	}
}

func TestMultiplexingFallsBackWhenNotSupported(t *testing.T) {
	fake := newFakeRemoteAccess(t, false)
	address := startClient(t, fake, true)

	for i := 0; i < 3; i++ {
		exchange(t, address, fmt.Sprintf("hello %d", i), false)
	}

	// one websocket for the (rejected) negotiation, then one per connection, and no further probing
	if got := fake.websockets.Load(); got != 4 {
		t.Fatalf("expected 4 remote access websockets, got %d", got)
	}
}

func TestWithoutMultiplexingEachConnectionUsesItsOwnSession(t *testing.T) {
	fake := newFakeRemoteAccess(t, true)
	address := startClient(t, fake, false)

	for i := 0; i < 3; i++ {
		exchange(t, address, fmt.Sprintf("hello %d", i), false)
	}

	if got := fake.websockets.Load(); got != 3 {
		t.Fatalf("expected 3 remote access websockets, got %d", got)
	}
}

func TestMultiplexingIsRetriedAfterAFailedNegotiation(t *testing.T) {
	fake := newFakeRemoteAccess(t, true)
	fake.dropNegotiations.Store(1)
	address := startClient(t, fake, true)

	for i := 0; i < 3; i++ {
		exchange(t, address, fmt.Sprintf("hello %d", i), false)
	}

	// dropped negotiation, then one websocket for the first connection, then a single multiplexed session
	if got := fake.websockets.Load(); got != 3 {
		t.Fatalf("expected 3 remote access websockets, got %d", got)
	}
}

func TestMultiplexingIsDisabledAfterRepeatedNegotiationsWithoutAnswer(t *testing.T) {
	fake := newFakeRemoteAccess(t, false)
	fake.dropNegotiations.Store(100)
	address := startClient(t, fake, true)

	for i := 0; i < multiplexMaxNoAnswers+2; i++ {
		exchange(t, address, fmt.Sprintf("hello %d", i), false)
	}

	// one websocket per connection, plus one per negotiation until the device is considered not to support it
	if got, want := fake.websockets.Load(), int32(2*multiplexMaxNoAnswers+2); got != want {
		t.Fatalf("expected %d remote access websockets, got %d", want, got)
	}
}

func TestMultiplexingNegotiationTimeoutIsConfigurable(t *testing.T) {
	fake := newFakeRemoteAccess(t, true)
	fake.silent = true
	ra := newClient(t, fake, RemoteAccessOptions{Multiplex: true, MultiplexNegotiationTimeout: 100 * time.Millisecond})
	go func() { _ = ra.Serve() }()

	for i := 0; i < multiplexMaxNoAnswers+1; i++ {
		started := time.Now()
		exchange(t, ra.GetListenerAddress(), fmt.Sprintf("hello %d", i), false)
		if elapsed := time.Since(started); elapsed > 2*time.Second {
			t.Fatalf("exchange took %v, expected the negotiation to time out after 100ms", elapsed)
		}
	}

	// one websocket per connection, plus one per negotiation until the device is considered not to support it
	if got, want := fake.websockets.Load(), int32(2*multiplexMaxNoAnswers+1); got != want {
		t.Fatalf("expected %d remote access websockets, got %d", want, got)
	}
}

func TestServeReturnsWhenListenerIsClosed(t *testing.T) {
	fake := newFakeRemoteAccess(t, true)
	ra := newClient(t, fake, RemoteAccessOptions{Multiplex: true})
	done := make(chan error, 1)
	go func() { done <- ra.Serve() }()

	ra.listener.Close()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serve did not return after the listener was closed")
	}
}

func TestPipeClosesLocalConnectionWhichIgnoresTheHalfClose(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	client, err := yamux.Client(clientSide, yamuxTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	server, err := yamux.Server(serverSide, yamuxTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	// the local client connects and then stays idle, keeping its side open
	idleClient, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer idleClient.Close()
	localConn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}

	stream, err := client.OpenStream()
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		pipe(localConn, stream, 100*time.Millisecond)
		close(done)
	}()

	// the remote side finishes
	remote, err := server.AcceptStream()
	if err != nil {
		t.Fatal(err)
	}
	remote.Close()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("pipe did not return after the remote side finished")
	}
}
