package notification2

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/wsdial"
	"github.com/reubenmiller/go-c8y/v2/pkg/wsurl"
	"github.com/tidwall/gjson"
)

const (
	// MaximumRetryInterval is the maximum interval (in seconds) between reconnection attempts
	MaximumRetryInterval int64 = 300

	// MinimumRetryInterval is the minimum interval (in seconds) between reconnection attempts
	MinimumRetryInterval int64 = 5

	// RetryBackoffFactor is the backoff factor applied to the retry interval for every unsuccessful reconnection attempt.
	// i.e. the next retry interval is calculated as follows
	// interval = MinimumRetryInterval
	// interval = Min(MaximumRetryInterval, interval * RetryBackoffFactor)
	RetryBackoffFactor float64 = 2
)

type ConnectionOptions struct {
	// Send pings to client with this interval. Must be less than pongWait.
	PingInterval time.Duration
	PongWait     time.Duration
	WriteWait    time.Duration
	Insecure     bool
}

func (o *ConnectionOptions) GetWriteDuration() time.Duration {
	if o.WriteWait == 0 {
		return 60 * time.Second
	}
	return o.WriteWait
}

func (o *ConnectionOptions) GetPongDuration() time.Duration {
	if o.PongWait == 0 {
		return 120 * time.Second
	}
	return o.PongWait
}

func (o *ConnectionOptions) GetPingDuration() time.Duration {
	if o.PingInterval == 0 {
		return 60 * time.Second
	}
	return o.PingInterval
}

// Client is a client used for the notification2 interface
type Client struct {
	mtx               sync.RWMutex
	host              string
	url               *url.URL
	ctx               context.Context
	cancel            context.CancelFunc
	workerDone        chan struct{}
	messages          chan *Message
	connected         bool
	dialer            *websocket.Dialer
	connector         wsdial.Connector
	reconnectCtx      context.Context
	ws                *websocket.Conn
	Subscription      Subscription
	ConnectionOptions ConnectionOptions

	hub  *Hub
	send chan []byte

	// writeMu serialises the writes to the websocket (the write handler and Unsubscribe)
	writeMu sync.Mutex

	// done is closed by Close and stops the hub, the write handler, the
	// worker and any reconnect loop.
	done      chan struct{}
	closeOnce sync.Once
}

// ErrClientClosed is returned when sending on a closed client.
// ErrMissingDialer is returned by Connect when the client has no websocket dialer.
var ErrMissingDialer = errors.New("notification2: missing websocket dialer")

var ErrClientClosed = errors.New("notification2: client is closed")

type Subscription struct {
	Consumer string `json:"consumer,omitempty"`
	Token    string `json:"token,omitempty"`

	TokenRenewal func(string) (string, error)
}

type ClientSubscription struct {
	Pattern  string
	Action   string
	Out      chan<- Message
	Disabled bool
}

// Message is the type delivered to subscribers.
type Message struct {
	Identifier  string `json:"identifier"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Payload     []byte `json:"data,omitempty"`
}

type ActionType string

var ActionTypeCreate ActionType = "CREATE"
var ActionTypeUpdate ActionType = "UPDATE"
var ActionTypeDelete ActionType = "DELETE"

func (m *Message) JSON() gjson.Result {
	return gjson.ParseBytes(m.Payload)
}

func getEndpoint(host string, subscription Subscription) (*url.URL, error) {
	c8yHost, err := wsurl.GetWebsocketURL(host, "notification2/consumer/")
	if err != nil {
		return nil, fmt.Errorf("invalid notification2 host %q: %w", host, err)
	}
	c8yHost.RawQuery = "token=" + subscription.Token

	if subscription.Consumer != "" {
		c8yHost.RawQuery += "&consumer=" + subscription.Consumer
	}
	return c8yHost, nil
}

// NewClient initialises a new notification2 client used to subscribe to
// realtime notifications from Cumulocity. It returns an error when host is invalid.
func NewClient(host string, wsDialer *websocket.Dialer, subscription Subscription, options ConnectionOptions) (*Client, error) {
	endpoint, err := getEndpoint(host, subscription)
	if err != nil {
		return nil, err
	}
	if wsDialer == nil {
		// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
		wsDialer = &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  45 * time.Second,
			EnableCompression: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: options.Insecure,
			},
		}
	}

	client := &Client{
		host:              host,
		url:               endpoint,
		dialer:            wsDialer,
		messages:          make(chan *Message, 100),
		Subscription:      subscription,
		ConnectionOptions: options,

		send: make(chan []byte),
		done: make(chan struct{}),

		hub: NewHub(),
	}

	go client.hub.Run(client.done)
	go client.writeHandler()
	return client, nil
}

// SetConnector sets the connector used to open the websocket, for example an
// *api.Client: the connection then uses the client's TLS and proxy settings,
// request middleware and extra headers, and respects dry run. The notification2
// token in the URL authenticates the connection, so the connector's
// Authorization header is not sent. nil goes back to the client's own dialer.
func (c *Client) SetConnector(connector wsdial.Connector) {
	c.mtx.Lock()
	c.connector = connector
	c.mtx.Unlock()
}

// Connect opens the websocket and will repeatedly initiate a websocket
// connection until `Close` is called on the client.
//
// Connect is ConnectContext with context.Background().
func (c *Client) Connect() error {
	return c.ConnectContext(context.Background())
}

// ConnectContext is like Connect, and cancelling ctx (or its deadline passing)
// aborts connecting. Automatic reconnects use the values of ctx but not its
// cancellation, so a short-lived ctx can be used to connect.
//
// Dry run: when the connector is in dry run (api.WithDryRun) nothing is connected
// and ConnectContext returns an error for which errors.Is(err,
// wsdial.ErrDryRun) is true; wsdial.DryRunRequest(err) is the upgrade request.
func (c *Client) ConnectContext(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if !c.IsConnected() {
		c.mtx.Lock()
		c.reconnectCtx = context.WithoutCancel(ctx)
		c.mtx.Unlock()
		err := c.connect(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) connectionContext() context.Context {
	c.mtx.RLock()
	defer c.mtx.RUnlock()
	if c.reconnectCtx != nil {
		return c.reconnectCtx
	}
	return context.Background()
}

func (c *Client) Endpoint() string {
	return c.url.String()
}

func (c *Client) URL(hideSensitive bool) string {
	endpoint, err := getEndpoint(c.url.String(), c.Subscription)
	if err != nil {
		return c.url.String()
	}
	wsURL := endpoint.String()
	if hideSensitive {
		if u, err := url.Parse(wsURL); err == nil {
			q := u.Query()
			if q.Has("token") {
				q.Set("token", "redacted")
				u.RawQuery = q.Encode()
				return u.String()
			}
		}
		return wsURL
	}
	return wsURL
}

// IsConnected returns true if the websocket is connected
func (c *Client) IsConnected() bool {
	c.mtx.RLock()
	isConnected := c.connected
	c.mtx.RUnlock()
	return isConnected
}

// Close closes the connection and stops all of the client's goroutines,
// including reconnect attempts. It is safe to call more than once.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		close(c.done)
	})
	c.stopWorker()
	return nil
}

// isClosed reports whether Close was called.
func (c *Client) isClosed() bool {
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

// stopWorker closes the websocket and stops the read worker, keeping the
// client usable for a reconnect.
func (c *Client) stopWorker() {
	if err := c.disconnect(); err != nil {
		slog.Warn("Failed to disconnect", "err", err)
	}
	c.mtx.Lock()
	if c.cancel != nil {
		slog.Debug("Stopping worker")
		c.cancel()
		c.mtx.Unlock()
		<-c.workerDone
		c.mtx.Lock()
		c.cancel = nil
		c.ctx = nil
		c.workerDone = nil
	}
	c.mtx.Unlock()
}

func (c *Client) disconnect() error {
	// Change to disconnected state, as the server will not send a reply upon receiving the /meta/disconnect command
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.connected = false
	if c.ws != nil {
		return c.ws.Close()
	}

	return nil
}

func (c *Client) createWebsocket(ctx context.Context) (*websocket.Conn, error) {

	if c.Subscription.TokenRenewal != nil {
		token, err := c.Subscription.TokenRenewal(c.Subscription.Token)
		if err != nil {
			return nil, err
		}
		c.Subscription.Token = token
	}

	slog.Debug(fmt.Sprintf("Establishing connection to %s", c.URL(true)))
	ws, err := c.dial(ctx)

	if err != nil {
		if !errors.Is(err, wsdial.ErrDryRun) {
			slog.Warn("Failed to establish connection", "err", err)
		}
		return ws, err
	}
	slog.Debug("Established websocket connection")
	return ws, nil
}

// dial opens the websocket with the connector, or with the client's dialer.
func (c *Client) dial(ctx context.Context) (*websocket.Conn, error) {
	c.mtx.RLock()
	connector := c.connector
	dialer := c.dialer
	c.mtx.RUnlock()

	if connector != nil {
		// The token in the URL is the credential of this endpoint
		ws, _, err := connector.DialWebsocket(ctxhelpers.WithNoAuth(ctx), c.URL(false), nil)
		return ws, err
	}
	if dialer == nil {
		return nil, ErrMissingDialer
	}
	ws, _, err := dialer.DialContext(ctx, c.URL(false), nil)
	return ws, err
}

func (c *Client) reconnect() error {
	c.stopWorker()

	connected := false
	interval := MinimumRetryInterval

	for !connected {
		slog.Warn(fmt.Sprintf("Retrying in %ds", interval))
		select {
		case <-c.done:
			return ErrClientClosed
		case <-time.After(time.Duration(interval) * time.Second):
		}
		err := c.connect(c.connectionContext())

		if err != nil {
			slog.Warn("Failed to connect", "err", err)
			interval = int64(math.Min(float64(MaximumRetryInterval), RetryBackoffFactor*float64(interval)))
			continue
		}

		connected = true
	}

	slog.Warn("Reestablished connection")
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *Client) connect(ctx context.Context) error {
	c.mtx.RLock()
	hasDialer := c.dialer != nil || c.connector != nil
	c.mtx.RUnlock()
	if !hasDialer {
		return ErrMissingDialer
	}

	// This may be overkill, but is in place to prevent unexpected errors
	if c.ws != nil {
		if err := c.ws.Close(); err != nil {
			slog.Info("Error whilst closing connection before connecting", "err", err)
		}
	}
	ws, err := c.createWebsocket(ctx)

	if err != nil {
		return err
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.ws = ws
	if c.ctx == nil {
		c.ctx, c.cancel = context.WithCancel(context.Background())
		c.workerDone = make(chan struct{})
		go c.worker()
	}
	c.connected = true

	return nil
}

func parseMessage(raw []byte) *Message {
	inHeader := true
	message := &Message{}

	scanner := bufio.NewScanner(bytes.NewReader(raw))

	i := 0
	for scanner.Scan() {
		// Note: .Bytes() does not allocate memory, so you need to allocate the data
		// and copy the data to another variable if you want it to persist
		line := scanner.Bytes()
		if len(line) == 0 {
			inHeader = false
			// empty line is the border between the header and body
			continue
		}
		if inHeader {
			switch i {
			case 0:
				message.Identifier = string(line)
			case 1:
				message.Description = string(line)
			case 2:
				message.Action = string(line)
			}
			// Ignore unknown header indexes
		} else {
			// Copy payload
			message.Payload = make([]byte, len(line))
			copy(message.Payload, line)
			// TODO: Check if a single websocket message can contain multiple messages
			// Stop processing further messages
			break
		}
		i++
	}
	return message
}

func (c *Client) writeHandler() {
	ticker := time.NewTicker(c.ConnectionOptions.GetPingDuration())

	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case message, ok := <-c.send:
			if !ok {
				// The send channel has been closed
				slog.Info("Channel has been closed")
				return
			}

			if c.ws != nil {
				c.writeMu.Lock()
				c.ws.SetWriteDeadline(time.Now().Add(c.ConnectionOptions.GetWriteDuration()))
				err := c.ws.WriteMessage(websocket.TextMessage, message)
				c.writeMu.Unlock()
				if err != nil {
					if c.isClosed() || errors.Is(err, net.ErrClosed) {
						// the client was closed while the message was waiting: nobody is left to read it
						slog.Debug("Dropped message as the connection is closed", "err", err)
					} else {
						slog.Warn("Failed to send message", "err", err)
					}
				}
			}

		case <-ticker.C:
			// Regularly check if the Websocket is alive by sending a PingMessage to the server
			if c.ws != nil {
				// A websocket ping should initiate a websocket pong response from the server
				// If the pong is not received in the minimum time, then the connection will be reset
				c.writeMu.Lock()
				c.ws.SetWriteDeadline(time.Now().Add(c.ConnectionOptions.GetWriteDuration()))
				err := c.ws.WriteMessage(websocket.PingMessage, nil)
				c.writeMu.Unlock()
				if err != nil {
					slog.Warn("Failed to send ping message to server", "err", err)
					// go c.reconnect()
					continue
				}
				slog.Debug("Sent ping successfully")
			}
		}
	}
}

func (c *Client) Register(pattern string, out chan<- Message) {
	slog.Debug(fmt.Sprintf("Subscribing to %s", pattern))

	select {
	case c.hub.register <- &ClientSubscription{
		Pattern:  pattern,
		Out:      out,
		Disabled: false,
	}:
	case <-c.done:
	}
}

func (c *Client) SendMessageAck(messageIdentifier string) error {
	slog.Debug("Sending message ack", "identifier", messageIdentifier)
	select {
	case c.send <- []byte(messageIdentifier):
		return nil
	case <-c.done:
		return ErrClientClosed
	}
}

func (c *Client) worker() {
	defer close(c.workerDone)
	done := make(chan struct{})

	c.ws.SetReadDeadline(time.Now().Add(c.ConnectionOptions.GetPongDuration()))
	c.ws.SetPongHandler(func(string) error {
		slog.Debug("Received pong message")
		c.ws.SetReadDeadline(time.Now().Add(c.ConnectionOptions.GetPongDuration()))
		return nil
	})

	go func() {
		defer close(done)
		for {
			messageType, rawMessage, err := c.ws.ReadMessage()

			if err == nil {
				slog.Debug("Received websocket message", "type", messageType, "len", len(rawMessage))
			}

			if err != nil {
				// Taken from https://github.com/gorilla/websocket/blob/main/examples/chat/client.go
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					slog.Warn("unexpected websocket error", "err", err)
				} else {
					slog.Info("websocket error", "err", err)
				}
				if !c.isClosed() {
					go c.reconnect()
				}
				break
			}

			switch messageType {
			case websocket.TextMessage:
				slog.Debug("Raw notification2 message", "len", len(rawMessage), "message", rawMessage)
				message := parseMessage(rawMessage)
				slog.Debug("message", "id", message.Identifier)
				slog.Debug("message", "description", message.Description)
				slog.Debug("message", "action", message.Action)
				slog.Debug("message", "payload", message.Payload)
				select {
				case c.hub.broadcast <- *message:
				case <-c.done:
					return
				}

			case websocket.CloseMessage:
				slog.Warn("Received close", "message", rawMessage)
			case websocket.PingMessage:
				slog.Debug("Received ping", "message", rawMessage)

			case websocket.PongMessage:
				slog.Debug("Received pong", "message", rawMessage)
			}
		}
	}()

	defer c.ws.Close()
	<-c.ctx.Done()
	slog.Info("Worker is shutting down")
}

// Unsubscribe tells the server that the subscriber is done. The message is
// written before Unsubscribe returns, so that closing the client afterwards
// does not drop it (a message queued for the write handler could be written to
// the connection which Close has closed).
func (c *Client) Unsubscribe() error {
	slog.Info("unsubscribing")
	if c.isClosed() {
		return ErrClientClosed
	}
	c.mtx.RLock()
	ws := c.ws
	c.mtx.RUnlock()
	if ws == nil {
		return ErrClientClosed
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	ws.SetWriteDeadline(time.Now().Add(c.ConnectionOptions.GetWriteDuration()))
	return ws.WriteMessage(websocket.TextMessage, []byte("unsubscribe_subscriber"))
}
