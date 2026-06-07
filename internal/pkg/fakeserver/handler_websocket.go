package fakeserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

// Reuse a single upgrader.  We accept connections from any origin so that
// tests don't need to provide an Origin header.
var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// realtimeBayeuxMessage represents a single Bayeux protocol frame as used by
// the Cumulocity CEP realtime client.
type realtimeBayeuxMessage struct {
	ID                       string         `json:"id,omitempty"`
	Channel                  string         `json:"channel"`
	ClientID                 string         `json:"clientId,omitempty"`
	Data                     any            `json:"data,omitempty"`
	Extension                any            `json:"ext,omitempty"`
	Version                  string         `json:"version,omitempty"`
	MinimumVersion           string         `json:"minimumVersion,omitempty"`
	SupportedConnectionTypes []string       `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string         `json:"connectionType,omitempty"`
	Subscription             string         `json:"subscription,omitempty"`
	Advice                   map[string]any `json:"advice,omitempty"`
	Successful               *bool          `json:"successful,omitempty"`
	Error                    string         `json:"error,omitempty"`
}

// RealtimeConnection is a handle for an active realtime connection on the
// fake server.  Tests may use it to push messages back to the client.
type RealtimeConnection struct {
	mu       sync.Mutex
	ws       *websocket.Conn
	clientID string
	closed   bool

	subsMu sync.Mutex
	subs   map[string]struct{}
}

// PushData broadcasts a data frame on the given subscription channel to the
// connected client.  Returns an error if the write fails.
func (rc *RealtimeConnection) PushData(channel string, action string, payload any) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.closed {
		return fmt.Errorf("connection closed")
	}
	msg := map[string]any{
		"channel":  channel,
		"clientId": rc.clientID,
		"data": map[string]any{
			"realtimeAction": action,
			"data":           payload,
		},
	}
	return rc.ws.WriteJSON([]any{msg})
}

// Subscriptions returns the currently active subscription patterns.
func (rc *RealtimeConnection) Subscriptions() []string {
	rc.subsMu.Lock()
	defer rc.subsMu.Unlock()
	out := make([]string, 0, len(rc.subs))
	for k := range rc.subs {
		out = append(out, k)
	}
	return out
}

// Close terminates the fake realtime connection.
func (rc *RealtimeConnection) Close() {
	rc.mu.Lock()
	rc.closed = true
	rc.ws.Close()
	rc.mu.Unlock()
}

// realtimeState tracks active realtime connections by clientId so tests can
// push messages without needing direct access to the websocket connection.
type realtimeState struct {
	mu          sync.Mutex
	connections map[string]*RealtimeConnection
	idSeq       uint64
}

func newRealtimeState() *realtimeState {
	return &realtimeState{connections: make(map[string]*RealtimeConnection)}
}

func (r *realtimeState) nextID() string {
	return strconv.FormatUint(atomic.AddUint64(&r.idSeq, 1), 10)
}

func (r *realtimeState) register(conn *RealtimeConnection) {
	r.mu.Lock()
	r.connections[conn.clientID] = conn
	r.mu.Unlock()
}

func (r *realtimeState) unregister(clientID string) {
	r.mu.Lock()
	delete(r.connections, clientID)
	r.mu.Unlock()
}

// Connections returns a snapshot of the active connections.
func (r *realtimeState) Connections() []*RealtimeConnection {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*RealtimeConnection, 0, len(r.connections))
	for _, c := range r.connections {
		out = append(out, c)
	}
	return out
}

// handleRealtime handles WebSocket upgrades on /cep/realtime and implements
// just enough of the Bayeux protocol used by Cumulocity for client tests to
// connect, subscribe, receive pushed frames and disconnect.
func (fs *FakeServer) handleRealtime(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn := &RealtimeConnection{
		ws:   ws,
		subs: make(map[string]struct{}),
	}
	defer func() {
		conn.Close()
		if conn.clientID != "" {
			fs.Realtime.unregister(conn.clientID)
		}
	}()

	for {
		var batch []realtimeBayeuxMessage
		if err := ws.ReadJSON(&batch); err != nil {
			return
		}

		responses := make([]realtimeBayeuxMessage, 0, len(batch))
		for _, msg := range batch {
			resp := realtimeBayeuxMessage{
				ID:       msg.ID,
				Channel:  msg.Channel,
				ClientID: msg.ClientID,
			}
			ok := true
			switch msg.Channel {
			case "/meta/handshake":
				conn.clientID = "fake-client-" + fs.Realtime.nextID()
				resp.ClientID = conn.clientID
				resp.Version = "1.0"
				resp.SupportedConnectionTypes = []string{"websocket"}
				fs.Realtime.register(conn)
			case "/meta/connect":
				resp.Advice = map[string]any{
					"interval":  0,
					"timeout":   60000,
					"reconnect": "retry",
				}
			case "/meta/subscribe":
				conn.subsMu.Lock()
				conn.subs[msg.Subscription] = struct{}{}
				conn.subsMu.Unlock()
				resp.Subscription = msg.Subscription
			case "/meta/unsubscribe":
				conn.subsMu.Lock()
				delete(conn.subs, msg.Subscription)
				conn.subsMu.Unlock()
				resp.Subscription = msg.Subscription
			case "/meta/disconnect":
				resp.ID = msg.ID
			default:
				// data publication from client -- not used in our tests
				ok = true
			}
			resp.Successful = &ok
			responses = append(responses, resp)
		}

		if len(responses) > 0 {
			conn.mu.Lock()
			err := ws.WriteJSON(responses)
			conn.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// Notification2Connection represents an active notification2 websocket session
// on the fake server.  Tests can use it to push notification messages to the
// client.
type Notification2Connection struct {
	mu       sync.Mutex
	ws       *websocket.Conn
	consumer string
	token    string

	acksMu sync.Mutex
	acks   []string
	closed bool

	unsubscribed chan struct{}
}

// PushMessage sends a notification2 frame with the given headers and JSON
// payload to the connected client.
func (nc *Notification2Connection) PushMessage(identifier, description, action string, payload []byte) error {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if nc.closed {
		return fmt.Errorf("connection closed")
	}
	frame := fmt.Sprintf("%s\n%s\n%s\n\n%s", identifier, description, action, string(payload))
	return nc.ws.WriteMessage(websocket.TextMessage, []byte(frame))
}

// Acks returns the message acknowledgements received from the client.
func (nc *Notification2Connection) Acks() []string {
	nc.acksMu.Lock()
	defer nc.acksMu.Unlock()
	out := make([]string, len(nc.acks))
	copy(out, nc.acks)
	return out
}

// Token returns the token query parameter the client used to connect.
func (nc *Notification2Connection) Token() string { return nc.token }

// Consumer returns the consumer query parameter the client used to connect.
func (nc *Notification2Connection) Consumer() string { return nc.consumer }

// Unsubscribed returns a channel that is closed once the client sends the
// "unsubscribe_subscriber" frame.
func (nc *Notification2Connection) Unsubscribed() <-chan struct{} { return nc.unsubscribed }

// Close terminates the connection.
func (nc *Notification2Connection) Close() {
	nc.mu.Lock()
	if !nc.closed {
		nc.closed = true
		nc.ws.Close()
	}
	nc.mu.Unlock()
}

// notification2State tracks active notification2 connections.
type notification2State struct {
	mu       sync.Mutex
	connect  chan *Notification2Connection
	connList []*Notification2Connection
}

func newNotification2State() *notification2State {
	return &notification2State{
		connect: make(chan *Notification2Connection, 16),
	}
}

func (s *notification2State) add(conn *Notification2Connection) {
	s.mu.Lock()
	s.connList = append(s.connList, conn)
	s.mu.Unlock()
	// Non-blocking notification.
	select {
	case s.connect <- conn:
	default:
	}
}

// Connections returns all notification2 connections that have been created on
// this server (including closed ones, in connection order).
func (s *notification2State) Connections() []*Notification2Connection {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Notification2Connection, len(s.connList))
	copy(out, s.connList)
	return out
}

// WaitForConnection waits for a new notification2 connection or until the
// timeout elapses.  Returns nil if the timeout is reached.
func (s *notification2State) WaitForConnection(timeout time.Duration) *Notification2Connection {
	select {
	case c := <-s.connect:
		return c
	case <-time.After(timeout):
		return nil
	}
}

// handleNotification2Stream handles WebSocket upgrades on
// /notification2/consumer/ and implements the minimal notification2 protocol
// used by the Cumulocity notification2 client.
func (fs *FakeServer) handleNotification2Stream(w http.ResponseWriter, r *http.Request) {
	ws, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	conn := &Notification2Connection{
		ws:           ws,
		token:        r.URL.Query().Get("token"),
		consumer:     r.URL.Query().Get("consumer"),
		unsubscribed: make(chan struct{}),
	}
	fs.Notification2.add(conn)

	defer conn.Close()

	for {
		_, raw, err := ws.ReadMessage()
		if err != nil {
			return
		}
		text := strings.TrimSpace(string(raw))
		if text == "unsubscribe_subscriber" {
			select {
			case <-conn.unsubscribed:
			default:
				close(conn.unsubscribed)
			}
			continue
		}
		// Treat anything else as an acknowledgement frame.
		conn.acksMu.Lock()
		conn.acks = append(conn.acks, text)
		conn.acksMu.Unlock()
	}
}

// Reset the trivial JSON marshalling helper to avoid unused import warnings
// when the file is built without other helpers.
var _ = json.Unmarshal
