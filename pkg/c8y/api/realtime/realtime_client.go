package realtime

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/obeattie/ohmyglob"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/wsurl"
	"golang.org/x/net/publicsuffix"
)

// Package-level logger with identifying attributes.
// Users can filter logs by checking for "component"="realtime" attribute.
// To configure logging, use SetLogger() on the Client.
var logger = slog.Default().With("component", "realtime")

const (
	// VERSION preferred Bayeux version
	VERSION = "1.0"

	// MINIMUM_VERSION supported Bayeux version
	MINIMUM_VERSION = "1.0"

	// MinimumRetryDelay is the minimum retry delay in milliseconds to wait before sending another /meta/connect message
	MinimumRetryDelay int64 = 500
)

const (
	// MaximumRetryInterval is the maximum interval (in seconds) between reconnection attempts
	MaximumRetryInterval int64 = 30

	// MinimumRetryInterval is the minimum interval (in seconds) between reconnection attempts
	MinimumRetryInterval int64 = 5

	// RetryBackoffFactor is the backoff factor applied to the retry interval for every unsuccessful reconnection attempt.
	// i.e. the next retry interval is calculated as follows
	// interval = MinimumRetryInterval
	// interval = Min(MaximumRetryInterval, interval * RetryBackoffFactor)
	RetryBackoffFactor float64 = 1.5
)

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	pingPeriod = (pongWait * 9) / 10
)

// Client allows connecting to a Bayeux server and subscribing to channels.
type Client struct {
	mtx           sync.RWMutex
	url           *url.URL
	c8yURL        *url.URL
	clientID      string
	ctx           context.Context
	cancel        context.CancelFunc
	workerDone    chan struct{}
	messages      chan *Message
	connected     bool
	dialer        *websocket.Dialer
	ws            *websocket.Conn
	extension     any
	tenant        string
	username      string
	password      string
	requestID     uint64
	requestHeader http.Header

	send chan *request

	hub *Hub

	pendingRequests sync.Map

	// logger is the logger used by this client instance
	// If not set, uses the package-level logger
	logger *slog.Logger
}

// Message is the type delivered to subscribers.
type Message struct {
	Channel      string       `json:"channel"`
	Payload      RealtimeData `json:"data,omitempty"`
	ID           string       `json:"id,omitempty"`
	ClientID     string       `json:"clientId,omitempty"`
	Extension    any          `json:"ext,omitempty"`
	Advice       *advice      `json:"advice,omitempty"`
	Successful   bool         `json:"successful,omitempty"`
	Subscription string       `json:"subscription,omitempty"`
}

// RealtimeData contains the websocket frame data
type RealtimeData struct {
	RealtimeAction string          `json:"realtimeAction,omitempty"`
	Data           jsondoc.JSONDoc `json:"data,omitempty"`
}

type subscription struct {
	glob       ohmyglob.Glob
	out        chan<- *Message
	isWildcard bool
	disabled   bool
}

type request struct {
	Channel                  string          `json:"channel"`
	Data                     json.RawMessage `json:"data,omitempty"`
	ID                       string          `json:"id,omitempty"`
	ClientID                 string          `json:"clientId,omitempty"`
	Extension                any             `json:"ext,omitempty"`
	Version                  string          `json:"version,omitempty"`
	MinimumVersion           string          `json:"minimumVersion,omitempty"`
	SupportedConnectionTypes []string        `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string          `json:"connectionType,omitempty"`
	Subscription             string          `json:"subscription,omitempty"`
	Advice                   *advice         `json:"advice,omitempty"`
}

type advice struct {
	Reconnect string `json:"reconnect,omitempty"`
	Timeout   int64  `json:"timeout"` // don't use omitempty, otherwise timeout: 0 will be removed
	Interval  int64  `json:"interval,omitempty"`
}

// MetaMessage Bayeux message
type MetaMessage struct {
	Message
	Version                  string   `json:"version,omitempty"`
	MinimumVersion           string   `json:"minimumVersion,omitempty"`
	SupportedConnectionTypes []string `json:"supportedConnectionTypes,omitempty"`
	ConnectionType           string   `json:"connectionType,omitempty"`
	Timestamp                string   `json:"timestamp,omitempty"`
	Successful               bool     `json:"successful"`
	Subscription             string   `json:"subscription,omitempty"`
	Error                    string   `json:"error,omitempty"`
	Advice                   *advice  `json:"advice,omitempty"`
}

type c8yExtensionMessage struct {
	ComCumulocityAuthn comCumulocityAuthn `json:"com.cumulocity.authn"`
}

type comCumulocityAuthn struct {
	Token     string `json:"token,omitempty"`
	XSRFToken string `json:"xsrfToken,omitempty"`
}

func getC8yExtension(tenant, username, password string) c8yExtensionMessage {
	return c8yExtensionMessage{
		ComCumulocityAuthn: comCumulocityAuthn{
			// Always use the tenant name as prefix in the c8y username!!! This ensures you connect to the correct tenant!
			Token: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s/%s:%s", tenant, username, password))),
		},
	}
}

func getC8yExtensionFromToken(token string) c8yExtensionMessage {
	return c8yExtensionMessage{
		ComCumulocityAuthn: comCumulocityAuthn{
			Token: token,
		},
	}
}

func getC8yExtensionFromXSRFToken(token string) c8yExtensionMessage {
	return c8yExtensionMessage{
		ComCumulocityAuthn: comCumulocityAuthn{
			// Always use the tenant name as prefix in the c8y username!!! This ensures you connect to the correct tenant!
			XSRFToken: token,
		},
	}
}

func getRealtimeURL(host string) *url.URL {
	c8yHost, err := wsurl.GetWebsocketURL(host, "cep/realtime")
	if err != nil {
		logger.Error("Invalid websocket url", "err", err)
		os.Exit(1)
	}
	return c8yHost
}

type ClientOptions struct {
	Host     string
	Tenant   string
	Username string
	Password string
	Token    string

	ChannelSize        int
	InsecureSkipVerify bool
}

// NewClient initializes a new Bayeux client. By default `http.DefaultClient`
// is used for HTTP connections.
func NewClient(wsDialer *websocket.Dialer, opt ClientOptions) *Client {
	if wsDialer == nil {
		// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
		wsDialer = &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  10 * time.Second,
			EnableCompression: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opt.InsecureSkipVerify,
			},
		}
	}

	if opt.ChannelSize < 0 {
		opt.ChannelSize = 100
	}

	// Convert url to a websocket
	websocketURL := getRealtimeURL(opt.Host)
	c8yURL, _ := url.Parse(opt.Host)

	client := &Client{
		url:      websocketURL,
		dialer:   wsDialer,
		messages: make(chan *Message, opt.ChannelSize),

		c8yURL: c8yURL,

		send: make(chan *request),

		hub: NewHub(),
	}
	if opt.Token != "" {
		client.extension = getC8yExtensionFromToken(opt.Token)
	} else {
		client.extension = getC8yExtension(opt.Tenant, opt.Username, opt.Password)
	}

	go client.hub.Run()
	go client.writeHandler()
	return client
}

// SetRequestHeader sets the header to use when establishing the realtime connection.
func (c *Client) SetRequestHeader(header http.Header) {
	c.requestHeader = header
}

// SetCookies sets the cookies used for outgoing requests
func (c *Client) SetCookies(cookies []*http.Cookie) error {
	if c.dialer == nil {
		return fmt.Errorf("dialer is nil")
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return fmt.Errorf("failed to create cookie jar: %w", err)
	}
	jar.SetCookies(c.c8yURL, cookies)
	c.dialer.Jar = jar
	return nil
}

// SetXSRFToken set the token required for authentication via OAUTH
func (c *Client) SetXSRFToken(token string) {
	c.extension = getC8yExtensionFromXSRFToken(token)
}

// SetBearerToken set the token required for authentication via OAUTH
func (c *Client) SetBearerToken(token string) {
	c.extension = getC8yExtensionFromToken(token)
}

// SetLogger configures a custom logger for this client instance.
// This allows users to:
// - Use a custom slog.Handler (e.g., to write to a file or change format)
// - Add additional context attributes to all realtime logs
// - Disable logging by passing slog.New(slog.NewTextHandler(io.Discard, nil))
//
// Example - Add custom attributes:
//
//	customLogger := slog.Default().With("service", "my-app", "client_id", "abc123")
//	client.SetLogger(customLogger)
//
// Example - Disable logging:
//
//	client.SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
func (c *Client) SetLogger(l *slog.Logger) {
	c.logger = l
}

// log returns the logger for this client instance.
// Falls back to package-level logger if not configured.
func (c *Client) log() *slog.Logger {
	if c.logger != nil {
		return c.logger
	}
	return logger
}

// Connect performs a handshake with the server and will repeatedly initiate a
// websocket connection until `Close` is called on the client.
func (c *Client) Connect() error {
	if !c.IsConnected() {

		err := <-c.connect()
		if err != nil {
			return err
		}

		err = <-c.getAdvice()
		if err != nil {
			return err
		}
	}
	return nil
}

// IsConnected returns true if the websocket is connected
func (c *Client) IsConnected() bool {
	c.mtx.RLock()
	isConnected := c.connected
	c.mtx.RUnlock()
	return isConnected
}

// Close notifies the Bayeux server of the intent to disconnect and terminates
// the background polling loop.
func (c *Client) Close() error {
	if err := c.disconnect(); err != nil {
		c.log().Info("Failed to disconnect. %s", "err", err)
	}

	c.mtx.Lock()
	if c.cancel != nil {
		c.log().Info("Stopping worker")
		c.cancel()
		c.cancel = nil
	}
	c.mtx.Unlock()

	// Wait for worker to finish if it was started
	if c.workerDone != nil {
		<-c.workerDone
	}

	return nil
}

// Disconnect sends a disconnect signal to the server and closes the websocket
func (c *Client) Disconnect() error {
	return c.disconnect()
}

func (c *Client) disconnect() error {
	message := &request{
		ID:       c.nextMessageID(),
		Channel:  "/meta/disconnect",
		ClientID: c.clientID,
	}

	// Change to disconnected state, as the server will not send a reply upon receiving the /meta/disconnect command
	c.mtx.Lock()
	c.connected = false
	c.mtx.Unlock()
	c.send <- message

	return nil
}

func (c *Client) createWebsocket() error {
	c.log().Info("Establishing connection", "to", c.url.String())
	ws, _, err := c.dialer.Dial(c.url.String(), c.requestHeader)

	if err != nil {
		return err
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.ws = ws
	return nil
}

func (c *Client) reconnect() error {
	connected := false

	c.mtx.Lock()
	if c.cancel != nil {
		c.cancel()
	}
	// Wait for worker to finish
	if c.workerDone != nil {
		<-c.workerDone
		c.workerDone = nil
	}
	c.cancel = nil
	c.connected = false
	c.mtx.Unlock()

	// Remove all pending requests
	c.pendingRequests.Range(func(key, value any) bool {
		c.pendingRequests.Delete(key)
		return true
	})

	interval := MinimumRetryInterval

	for !connected {
		c.log().Info(fmt.Sprintf("Retrying in %ds", interval))
		<-time.After(time.Duration(interval) * time.Second)
		c.ws.Close()
		err := c.createWebsocket()

		if err != nil {
			interval = int64(math.Min(float64(MaximumRetryInterval), RetryBackoffFactor*float64(interval)))
			continue
		}

		if err := c.Connect(); err != nil {
			c.log().Info("Failed to get advice from server", "err", err)
		} else {
			connected = true
		}
	}

	c.log().Info("Established connection, any subscriptions will be also be resubmitted")

	c.reactivateSubscriptions()
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *Client) connect() chan error {
	if c.dialer == nil {
		panic("Missing dialer for realtime client")
	}
	c.log().Info(fmt.Sprintf("Establishing connection to %s", c.url.String()))
	ws, _, err := c.dialer.Dial(c.url.String(), c.requestHeader)

	if err != nil {
		panic(err)
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	c.ws = ws

	if c.cancel == nil {
		c.ctx, c.cancel = context.WithCancel(context.Background())
		c.workerDone = make(chan struct{})
		go c.worker()
	}

	return c.handshake()
}

func (c *Client) worker() {
	defer close(c.workerDone)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})

	c.ws.SetPongHandler(func(string) error {
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	go func() {
		defer close(done)
		for {
			messages := []Message{}

			err := c.ws.ReadJSON(&messages)

			if err != nil {
				c.log().Info("ws ReadJSON", "err", err, "message", messages)

				if !c.IsConnected() {
					c.log().Info("Connection has been closed by the client")
					return
				}
				c.log().Info("Handling connection error. You need to reconnect")

				go c.reconnect()
				return
			}

			for _, message := range messages {
				if strings.HasPrefix(message.Channel, "/meta") {
					if messageText, err := json.Marshal(message); err == nil {
						c.log().Info("ws (recv)", "channel", message.Channel, "text", messageText)
					}
				}

				switch channelType := message.Channel; channelType {
				case "/meta/handshake":
					if message.Successful {
						c.mtx.Lock()
						c.clientID = message.ClientID
						c.connected = true
						c.mtx.Unlock()
					} else {
						c.log().Error("No clientID present in handshake. Check that the tenant, username and password is correct", "message", message)
						os.Exit(1)
					}

				case "/meta/subscribe":
					if message.Successful {
						c.log().Info("Successfully subscribed to channel", "value", message.Subscription)
					} else {
						c.log().Info("Failed to subscribe to channel", "value", message.Subscription)
					}

				case "/meta/unsubscribe":
					if message.Successful {
						c.log().Info("Successfully unsubscribed to channel", "value", message.Subscription)
					}

				case "/meta/connect":
					// https://docs.cometd.org/current/reference/
					wasConnected := c.IsConnected()
					connected := message.Successful

					if message.Advice != nil {
						retryDelay := message.Advice.Interval
						if retryDelay <= 0 {
							// Minimum retry delay
							retryDelay = MinimumRetryDelay
						}
						switch message.Advice.Reconnect {
						case "handshake":
							c.log().Info("Scheduling sending of new handshake to server with a small delay", "delay_ms", retryDelay)
							time.AfterFunc(time.Duration(retryDelay)*time.Millisecond, func() {
								c.handshake()
							})
						case "retry":
							c.log().Info("Resending /meta/connect heartbeat after a small delay", "delay_ms", retryDelay)
							time.AfterFunc(time.Duration(retryDelay)*time.Millisecond, func() {
								c.sendMeta()
							})
						case "none":
							// Do not attempt to retry or send a handshake as it must respect the servers response
							panic("Server indicated that no retry or handshake should be done")
						}
						// Server indicated that a handshake should be sent again
						break
					}

					if !wasConnected && connected {
						// Reconnected
					} else if wasConnected && !connected {
						// Disconnected
						c.disconnect()
					} else if connected {
						// New connection
						c.mtx.Lock()
						c.connected = true
						c.mtx.Unlock()

						go c.sendMeta()
					}

				case "/meta/disconnect":
					if message.Successful {
						c.log().Info("Successfully disconnected with server")
					}

				default:
					// Data package received
					message.Payload.Data = jsondoc.New(message.Payload.Data.Bytes())
					c.hub.broadcast <- &message
				}

				// remove the message from the queue
				if message.ID != "" {
					c.log().Info("Removing message from pending requests", "id", message.ID)
					c.pendingRequests.Delete(message.ID)
					c.logRemainingResponses()
				}
			}
		}
	}()

	for {
		defer c.ws.Close()
		select {
		case <-c.ctx.Done():
			return

		case <-interrupt:
			c.log().Info("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			if err := c.Disconnect(); err != nil {
				c.log().Info("Failed to send disconnect to server", "err", err)
				return
			}

			return
		}
	}
}

func (c *Client) handshake() chan error {
	message := &request{
		ID:                       c.nextMessageID(),
		Channel:                  "/meta/handshake",
		Version:                  VERSION,
		MinimumVersion:           MINIMUM_VERSION,
		SupportedConnectionTypes: []string{"websocket", "long-polling"},
		Extension:                c.extension,
		Advice: &advice{
			Interval:  0,
			Timeout:   60000,
			Reconnect: "retry",
		},
	}

	c.send <- message
	return c.WaitForMessage(message.ID)
}

func (c *Client) sendMeta() error {
	if c.ws == nil {
		return fmt.Errorf("websocket is nil")
	}
	message := &request{
		ID:             c.nextMessageID(),
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	c.send <- message
	return nil
}

func (c *Client) getAdvice() chan error {
	clientID := c.clientID
	message := &request{
		ID:             c.nextMessageID(),
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       clientID,
		Advice: &advice{
			Timeout: 0,
		},
	}

	c.send <- message
	return c.WaitForMessage(message.ID)
}

func getRealtimeID(id ...string) string {
	if len(id) > 0 {
		return id[0]
	}
	return "*"
}

// Alarms subscribes to events on alarms objects from the CEP realtime engine
func Alarms(id ...string) string {
	return "/alarms/" + getRealtimeID(id...)
}

// AlarmsWithChildren subscribes to events on alarms (including children) objects from the CEP realtime engine
func AlarmsWithChildren(id ...string) string {
	return "/alarmsWithChildren/" + getRealtimeID(id...)
}

// Events subscribes to events on event objects from the CEP realtime engine
func Events(id ...string) string {
	return "/events/" + getRealtimeID(id...)
}

// ManagedObjects subscribes to events on managed objects from the CEP realtime engine
func ManagedObjects(id ...string) string {
	return "/managedobjects/" + getRealtimeID(id...)
}

// Measurements subscribes to events on measurement objects from the CEP realtime engine
func Measurements(id ...string) string {
	return "/measurements/" + getRealtimeID(id...)
}

// Operations subscribes to events on operations objects from the CEP realtime engine
func Operations(id ...string) string {
	return "/operations/" + getRealtimeID(id...)
}

// Subscribe setup a subscription to the given element
// The subscription will be automatically cancelled when the context is cancelled or times out
func (c *Client) Subscribe(ctx context.Context, pattern string, out chan<- *Message) chan error {
	c.log().Info("Subscribing to pattern", "value", pattern)

	glob, err := ohmyglob.Compile(pattern, nil)
	if err != nil {
		errCh := make(chan error, 1)
		errCh <- fmt.Errorf("invalid pattern: %s", err)
		close(errCh)
		close(out)
		return errCh
	}

	message := &request{
		ID:             c.nextMessageID(),
		Channel:        "/meta/subscribe",
		Subscription:   pattern,
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	c.hub.register <- &subscription{
		glob:       glob,
		out:        out,
		isWildcard: strings.HasSuffix(glob.String(), "*"),
		disabled:   false,
	}

	c.send <- message

	errCh := c.WaitForMessage(message.ID)

	// Monitor context for cancellation/timeout throughout the subscription lifetime
	go func() {
		// First, wait for subscription acknowledgment or context cancellation
		select {
		case <-ctx.Done():
			// Context cancelled before subscription completes
			c.log().Info("Context cancelled before subscription completed", "pattern", pattern, "err", ctx.Err())
			unsubErrCh := c.Unsubscribe(pattern)
			select {
			case <-unsubErrCh:
				c.log().Info("Successfully unsubscribed", "pattern", pattern)
			case <-time.After(5 * time.Second):
				c.log().Warn("Unsubscribe timed out", "pattern", pattern)
			}
			return

		case err, ok := <-errCh:
			if !ok || err != nil {
				// Subscription failed or channel closed with error
				if err != nil {
					c.log().Warn("Subscription failed", "pattern", pattern, "err", err)
				}
				return
			}
			// Subscription succeeded (received nil) - continue to monitor context below
			c.log().Debug("Subscription acknowledged successfully", "pattern", pattern)
		}

		// Subscription is now active - monitor context for the lifetime of the subscription
		<-ctx.Done()
		c.log().Info("Context cancelled, unsubscribing", "pattern", pattern, "err", ctx.Err())
		unsubErrCh := c.Unsubscribe(pattern)
		select {
		case <-unsubErrCh:
			c.log().Info("Successfully unsubscribed", "pattern", pattern)
		case <-time.After(5 * time.Second):
			c.log().Warn("Unsubscribe timed out", "pattern", pattern)
		}
	}()

	return errCh
}

// reactivateSubscriptions sends subscription messages to the Bayeux server for each active subscription currently set in the client
func (c *Client) reactivateSubscriptions() {
	ids := []string{}

	for _, subscription := range c.hub.GetActiveChannels() {
		message := &request{
			ID:             c.nextMessageID(),
			Channel:        "/meta/subscribe",
			Subscription:   subscription,
			ConnectionType: "websocket",
			ClientID:       c.clientID,
		}

		ids = append(ids, message.ID)
		c.send <- message
	}

	c.WaitForMessages(ids...)
}

// UnsubscribeAll unsubscribes to all of the subscribed channels.
// The channel related to the subscription is left open, and will be
// reused if another call with the same pattern is made to Subscribe()
func (c *Client) UnsubscribeAll() chan error {
	ids := []string{}

	subs := c.hub.GetActiveChannels()
	for _, pattern := range subs {
		message := &request{
			ID:           c.nextMessageID(),
			Channel:      "/meta/unsubscribe",
			Subscription: pattern,
			ClientID:     c.clientID,
		}

		ids = append(ids, message.ID)
		c.send <- message
	}

	// Wait for the server to response to the unsubscribe messages
	// only when all of them have been received (or a timeout has occurred) then return
	return c.WaitForMessages(ids...)
}

// Unsubscribe unsubscribe to a given pattern
func (c *Client) Unsubscribe(pattern string) chan error {
	c.log().Info("Unsubscribing to pattern", "value", pattern)

	message := &request{
		ID:           c.nextMessageID(),
		Channel:      "/meta/unsubscribe",
		Subscription: pattern,
		ClientID:     c.clientID,
	}

	c.hub.unregister <- pattern
	c.send <- message
	return c.WaitForMessage(message.ID)
}

func (c *Client) nextMessageID() string {
	return strconv.FormatUint(atomic.AddUint64(&c.requestID, 1), 10)
}

func (c *Client) logMessage(r *request) {
	if text, err := json.Marshal(r); err == nil {
		c.log().Info("ws (send)", "channel", r.Channel, "text", text)
	} else {
		c.log().Info("Could not marshal message for sending", "err", err)
	}
}

func (c *Client) logRemainingResponses() {
	ids := []string{}
	c.pendingRequests.Range(func(key, value interface{}) bool {
		ids = append(ids, key.(string))
		return true
	})
	c.log().Info("Pending messages", "id", ids)
}

// WaitForMessages waits for a server response related to the list of message ids
func (c *Client) WaitForMessages(ids ...string) chan error {
	wg := new(sync.WaitGroup)
	wg.Add(len(ids))

	errorChannel := make(chan error)
	defer close(errorChannel)

	for _, id := range ids {
		go func(id string) {
			err := <-c.WaitForMessage(id)
			if err != nil {
				errorChannel <- err
			}
			wg.Done()
		}(id)
	}
	wg.Wait()
	return errorChannel
}

// WaitForMessage waits for a message with the corresponding id to be sent by the server
func (c *Client) WaitForMessage(ID string) chan error {
	out := make(chan error)

	waitInterval := 10 * time.Second

	c.log().Info("Waiting for message", "id", ID)

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer func() {
			ticker.Stop()
			close(out)
		}()
		timeout := time.After(waitInterval)

		for {
			select {
			case <-ticker.C:
				// c.log().Info("Checking if ID has been removed")
				if _, exists := c.pendingRequests.Load(ID); !exists {
					c.log().Info("Received message", "id", ID)
					out <- nil
					return
				}
			case <-timeout:
				out <- errors.New("Timeout")
				return
			}
		}
	}()

	return out
}

func (c *Client) writeHandler() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()

		// Close
		close(c.send)
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The send channel has been closed
				c.log().Info("Channel has been closed")
				return
			}

			if message.ID == "" {
				message.ID = c.nextMessageID()
			}

			// Store message id
			c.pendingRequests.Store(message.ID, message)

			c.logMessage(message)
			c.logRemainingResponses()

			if c.ws != nil {
				if err := c.ws.WriteJSON([]request{*message}); err != nil {
					c.log().Info("Failed to send JSON message", "err", err)
				}
			}

		case <-ticker.C:
			// Regularly check if the Websocket is alive by sending a PingMessage to the server
			if c.ws != nil {
				// A websocket ping should initiate a websocket pong response from the server
				// If the pong is not received in the minimum time, then the connection will be reset
				c.ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.log().Info("Failed to send ping message to server")
					go c.reconnect()
					break
				}
				c.log().Info("Sent ping successfully")
			}
		}
	}
}
