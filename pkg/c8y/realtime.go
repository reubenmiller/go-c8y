package c8y

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	"github.com/tidwall/gjson"
	tomb "gopkg.in/tomb.v2"
)

const (
	// VERSION preferred Bayeux version
	VERSION = "1.0"

	// MINIMUMVERSION supported Bayeux version
	MINIMUMVERSION = "1.0"
)

// RealtimeClient allows connecting to a Bayeux server and subscribing to channels.
type RealtimeClient struct {
	mtx           sync.RWMutex
	url           *url.URL
	clientID      string
	tomb          *tomb.Tomb
	subscriptions []subscription
	messages      chan *Message
	connected     bool
	http          *http.Client
	dialer        *websocket.Dialer
	ws            *websocket.Conn
	interval      time.Duration
	extension     interface{}
	tenant        string
	username      string
	password      string
	requestID     uint64

	connectionStatus   chan bool
	subscribeCommand   chan struct{}
	unsubscribeCommand chan struct{}
}

// Message is the type delivered to subscribers.
type Message struct {
	Channel      string       `json:"channel"`
	Payload      RealtimeData `json:"data,omitempty"`
	ID           string       `json:"id,omitempty"`
	ClientID     string       `json:"clientId,omitempty"`
	Extension    interface{}  `json:"ext,omitempty"`
	Advice       *advice      `json:"advice,omitempty"`
	Successful   bool         `json:"successful,omitempty"`
	Subscription string       `json:"subscription,omitempty"`
}

// RealtimeData contains the websocket frame data
type RealtimeData struct {
	RealtimeAction string          `json:"realtimeAction,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`

	Item gjson.Result `json:"-"`
}

type subscription struct {
	glob     ohmyglob.Glob
	out      chan<- *Message
	disabled bool
}

type request struct {
	Channel                  string          `json:"channel"`
	Data                     json.RawMessage `json:"data,omitempty"`
	ID                       string          `json:"id,omitempty"`
	ClientID                 string          `json:"clientId,omitempty"`
	Extension                interface{}     `json:"ext,omitempty"`
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
	Interval  int    `json:"interval,omitempty"`
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
	Token string `json:"token"`
}

func getC8yExtension(tenant, username, password string) c8yExtensionMessage {
	return c8yExtensionMessage{
		ComCumulocityAuthn: comCumulocityAuthn{
			// Always use the tenant name as prefix in the c8y username!!! This ensures you connect to the correct tenant!
			Token: base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s/%s:%s", tenant, username, password))),
		},
	}
}

func getRealtimURL(host string) *url.URL {
	c8yhost, _ := url.Parse(host)

	if c8yhost.Scheme == "http" {
		c8yhost.Scheme = "ws"
	} else {
		c8yhost.Scheme = "wss"
	}

	c8yhost.Path = "/cep/realtime"

	return c8yhost
}

// NewRealtimeClient initialises a new Bayeux client. By default `http.DefaultClient`
// is used for HTTP connections.
func NewRealtimeClient(host string, wsDialer *websocket.Dialer, tenant, username, password string) *RealtimeClient {
	if wsDialer == nil {
		// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
		wsDialer = &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  10 * time.Second,
			EnableCompression: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	// Convert url to a websocket
	websocketURL := getRealtimURL(host)

	return &RealtimeClient{
		url:       websocketURL,
		dialer:    wsDialer,
		messages:  make(chan *Message, 100),
		extension: getC8yExtension(tenant, username, password),

		tenant:   tenant,
		username: username,
		password: password,

		connectionStatus:   make(chan bool),
		subscribeCommand:   make(chan struct{}),
		unsubscribeCommand: make(chan struct{}),
	}
}

// TenantName returns the tenant name used in the client
func (c *RealtimeClient) TenantName() string {
	return c.tenant
}

// Connect performs a handshake with the server and will repeatedly initiate a
// websocket connection until `Close` is called on the client.
func (c *RealtimeClient) Connect() error {
	if !c.IsConnected() {
		return c.connect()
	}
	return nil
}

// IsConnected returns true if the websocket is connected
func (c *RealtimeClient) IsConnected() bool {
	c.mtx.RLock()
	isConnected := c.connected
	c.mtx.RUnlock()
	return isConnected
}

// Close notifies the Bayeux server of the intent to disconnect and terminates
// the background polling loop.
func (c *RealtimeClient) Close() error {
	if err := c.disconnect(); err != nil {
		log.Printf("Failed to disconnect. %s", err)
	}
	c.WaitForDisconnection()
	log.Printf("Killing go routine")
	c.tomb.Killf("Close")
	return nil
}

// Disconnect sends a disconnect signal to the server and closes the websocket
func (c *RealtimeClient) Disconnect() error {
	return c.disconnect()
}

func (c *RealtimeClient) disconnect() error {
	message := &request{
		Channel:  "/meta/disconnect",
		ClientID: c.clientID,
	}

	// Change to disconnected state, as the server will not send a reply upon receiving the /meta/disconnect command
	c.mtx.Lock()
	c.connected = false
	c.mtx.Unlock()
	// c.connectionStatus <- false

	err := c.sendJSON(message)

	if err != nil {
		return err
	}
	return nil
}

// WaitForConnection waits for the connection to be estabilished before returning
func (c *RealtimeClient) WaitForConnection(timeout ...time.Duration) error {
	d := getDurationWithDefault(timeout...)
	timeoutCh := time.After(d)

	for {
		select {
		case isConnected := <-c.connectionStatus:
			if isConnected {
				return nil
			}
		case <-timeoutCh:
			err := errors.New("Timed out whilst waiting for pending responses to be received")
			log.Printf("Waiting error: %s", err)
			return err
		}
	}
}

// WaitForPendingSubscribe waits for a message to be received on the Subscribe channel.
// It should be called after using Subscribe() if you want to serially wait for response from
// the server regarding the last sent Subscribe command
func (c *RealtimeClient) WaitForPendingSubscribe(timeout ...time.Duration) error {
	d := getDurationWithDefault(timeout...)
	timeoutCh := time.After(d)

	select {
	case <-c.subscribeCommand:
		return nil
	case <-timeoutCh:
		err := errors.New("Timed out whilst waiting for pending responses to be received")
		log.Printf("Waiting error: %s", err)
		return err
	}
}

// WaitForPendingUnsubscribe waits for a message to be received on the Unsubscribe channel.
// It should be called after using Unsubscribe() if you want to serially wait for response from
// the server regarding the last sent Unsubscribe command
func (c *RealtimeClient) WaitForPendingUnsubscribe(timeout ...time.Duration) {
	d := getDurationWithDefault(timeout...)
	timeoutCh := time.After(d)

	select {
	case <-c.unsubscribeCommand:
		return
	case <-timeoutCh:
		err := errors.New("Timed out whilst waiting for pending responses to be received")
		log.Printf("Waiting error: %s", err)
		return
	}
}

// WaitForDisconnection waits for the disconnection of the realtime client to the server
func (c *RealtimeClient) WaitForDisconnection() error {
	ticker := time.NewTicker(200 * time.Millisecond)
	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-ticker.C:
			if !c.IsConnected() {
				log.Printf("Disconnected from server")
				return nil
			}
		case <-timeout:
			err := errors.New("Timed out whilst waiting for /meta/disconnect from server")
			log.Printf("Disconnect error: %s", err)
			return err
		}
	}
}

func (c *RealtimeClient) createWebsocket() error {
	log.Printf("Establishing connection to %s", c.url.String())
	ws, _, err := c.dialer.Dial(c.url.String(), nil)

	if err != nil {
		return err
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.ws = ws
	return nil
}

func (c *RealtimeClient) reconnect() error {
	c.ws.Close()
	err := c.createWebsocket()

	if err != nil {
		c.mtx.Lock()
		c.connected = false
		c.mtx.Unlock()
		// c.connectionStatus <- false
		return err
	}
	c.getAdvice()
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *RealtimeClient) connect() error {
	if c.dialer == nil {
		panic("Missing dialer for realtime client")
	}
	log.Printf("Establishing connection to %s", c.url.String())
	ws, _, err := c.dialer.Dial(c.url.String(), nil)

	if err != nil {
		return err
	}

	c.ws = ws

	if err := c.handshake(); err != nil {
		return err
	}

	c.tomb = &tomb.Tomb{}
	c.tomb.Go(c.worker)

	return nil
}

func (c *RealtimeClient) worker() error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			messages := []Message{}
			err := c.ws.ReadJSON(&messages)

			if err != nil {
				log.Printf("wc ReadJSON: error=%s, message=%v", err, messages)

				/* if strings.Contains(err.Error(), "(normal)") {
					log.Println("Connection has been closed by the client")
					return
				} */

				if !c.IsConnected() {
					log.Println("Connection has been closed by the client")
					return
				}
				log.Println("Handling connection error. You need to reconnect")
				if err := c.connect(); err != nil {
					log.Printf("Failed to send connect. %s", err)
				}
				// c.reconnect()
			}

			for _, message := range messages {
				if strings.HasPrefix(message.Channel, "/meta") {
					if messageText, err := json.Marshal(message); err == nil {
						log.Printf("ws (recv): %s : %s", message.Channel, messageText)
					}
				}
				switch channelType := message.Channel; channelType {
				case "/meta/handshake":
					if message.Successful && message.ClientID != "" {
						c.mtx.Lock()
						c.clientID = message.ClientID
						c.connected = true
						c.mtx.Unlock()
						c.connectionStatus <- true

						// Get /meta/connect information about the connection
						if err := c.getAdvice(); err != nil {
							log.Printf("Failed to send init /meta/connect. %s", err)
						}
					} else {
						log.Panicf("No clientID present in handshake. Check that the tenant, usename and password is correct. Raw Message: %v", message)
					}

				case "/meta/subscribe":
					if message.Successful {
						log.Printf("Successfully subscribed to channel %s", message.Subscription)
					} else {
						log.Printf("Failed to subscribe to channel %s", message.Subscription)
					}
					c.subscribeCommand <- struct{}{}
					log.Println("Posted to subscribeCommand channel")

				case "/meta/unsubscribe":
					if message.Successful {
						// TODO: Unsubscribe to channel
						log.Printf("Successfully unsubscribed to channel %s", message.Subscription)
					}
					c.unsubscribeCommand <- struct{}{}
					log.Println("Posted to unsubscribeCommand channel")

				case "/meta/connect":
					// https://docs.cometd.org/current/reference/
					wasConnected := c.IsConnected()
					connected := message.Successful

					if message.Advice != nil {
						if message.Advice.Reconnect == "handshake" {
							log.Printf("Received advice from server to reconnect using handshake")
							if err := c.handshake(); err != nil {
								log.Printf("Failed to send handshake")
							}
						}
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
						// c.connectionStatus <- true

						if err := c.sendMeta(); err != nil {
							log.Printf("Failed to send /meta/connect reponse to server")
						}
					}

				case "/meta/disconnect":
					if message.Successful {
						log.Printf("Successfully disconnected with server")
					}

				default:
					// Data package received
					if !c.IsConnected() {
						return
					}
					message.Payload.Item = gjson.ParseBytes(message.Payload.Data)
					subscriptions := c.subscriptions

					for _, s := range subscriptions {
						if s.glob.MatchString(message.Channel) && !s.disabled {
							if c.connected {
								s.out <- &message
							}
						}
					}
				}
			}
		}
	}()

	for {
		defer c.ws.Close()
		select {
		case <-c.tomb.Dying():
			return nil

		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			if err := c.Disconnect(); err != nil {
				log.Println("Failed to send disconnect to server:", err)
				return err
			}
			/* err := c.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return err
			} */

			return fmt.Errorf("Stopping websocket")
		}
	}
}

func (c *RealtimeClient) handshake() error {
	handshakeMessage := &request{
		Channel:                  "/meta/handshake",
		Version:                  VERSION,
		MinimumVersion:           MINIMUMVERSION,
		SupportedConnectionTypes: []string{"websocket", "long-polling"},
		Extension:                c.extension,
		Advice: &advice{
			Interval:  0,
			Timeout:   60000,
			Reconnect: "retry",
		},
	}

	err := c.sendJSON(handshakeMessage)
	return err
}

func (c *RealtimeClient) sendMeta() error {
	if c.ws == nil {
		return fmt.Errorf("Websocket is nil")
	}
	message := &request{
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	return c.sendJSON(message)
}

func (c *RealtimeClient) getAdvice() error {
	if c.ws == nil {
		return fmt.Errorf("Websocket is nil")
	}
	clientID := c.clientID
	message := &request{
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       clientID,
		Advice: &advice{
			Timeout: 0,
		},
	}

	if err := c.sendJSON(message); err != nil {
		return err
	}

	return nil
}

func getRealtimeID(id ...string) string {
	if len(id) > 0 {
		return id[0]
	}
	return "*"
}

func getDurationWithDefault(d ...time.Duration) time.Duration {
	if len(d) == 0 {
		return 30 * time.Second
	}
	return d[0]
}

// RealtimeAlarms subscribes to events on alarms objects from the CEP realtime engine
func RealtimeAlarms(id ...string) string {
	return "/alarms/" + getRealtimeID(id...)
}

// RealtimeAlarmsWithChildren subscribes to events on alarms (including children) objects from the CEP realtime engine
func RealtimeAlarmsWithChildren(id ...string) string {
	return "/alarmsWithChildren/" + getRealtimeID(id...)
}

// RealtimeEvents subscribes to events on event objects from the CEP realtime engine
func RealtimeEvents(id ...string) string {
	return "/events/" + getRealtimeID(id...)
}

// RealtimeManagedObjects subscribes to events on managed objects from the CEP realtime engine
func RealtimeManagedObjects(id ...string) string {
	return "/managedobjects/" + getRealtimeID(id...)
}

// RealtimeMeasurements subscribes to events on measurement objects from the CEP realtime engine
func RealtimeMeasurements(id ...string) string {
	return "/measurements/" + getRealtimeID(id...)
}

// RealtimeOperations subscribes to events on operations objects from the CEP realtime engine
func RealtimeOperations(id ...string) string {
	return "/operations/" + getRealtimeID(id...)
}

// Subscribe setup a subscription to the given element
func (c *RealtimeClient) Subscribe(pattern string, out chan<- *Message) error {
	log.Println("Subscribing to ", pattern)

	glob, err := ohmyglob.Compile(pattern, nil)
	if err != nil {
		return fmt.Errorf("Invalid pattern: %s", err)
	}

	message := &request{
		Channel:        "/meta/subscribe",
		Subscription:   pattern,
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	c.mtx.Lock()
	existingSubscription := false
	for i := range c.subscriptions {
		if c.subscriptions[i].glob.String() == pattern {
			log.Printf("Enabling existing channel %s", pattern)
			existingSubscription = true
			// c.subscriptions[i].disabled = false
		}
	}
	if !existingSubscription {
		c.subscriptions = append(c.subscriptions, subscription{
			glob: glob,
			out:  out,
		})
	}
	c.mtx.Unlock()

	err = c.sendJSON(message)
	return err
}

// UnsubscribeAll unsubscribes to all of the subscribed channels.
// The channel related to the subscriptin is left open, and will be
// reused if another call with the same pattern is made to Subscribe()
func (c *RealtimeClient) UnsubscribeAll() (errs []error) {
	for i, subscription := range c.subscriptions {
		message := &request{
			Channel:      "/meta/unsubscribe",
			Subscription: subscription.glob.String(),
			ClientID:     c.clientID,
		}

		err := c.sendJSON(message)
		if err != nil {
			log.Printf("could not send unsubscribe message. %s", err)
		}
		c.mtx.Lock()
		c.subscriptions[i].disabled = true
		c.mtx.Unlock()
	}
	return
}

// Unsubscribe unsubscribe to a given pattern
func (c *RealtimeClient) Unsubscribe(pattern string) error {
	log.Println("unsubscribing to ", pattern)

	_, err := ohmyglob.Compile(pattern, nil)
	if err != nil {
		return fmt.Errorf("Invalid pattern: %s", err)
	}

	message := &request{
		Channel:      "/meta/unsubscribe",
		Subscription: pattern,
		ClientID:     c.clientID,
	}

	err = c.sendJSON(message)

	if err != nil {
		log.Printf("Failed to unsubscribe to subscription. %s", err)
	}

	return err
}

func (c *RealtimeClient) sendJSON(r *request) error {
	// Add a unique request id
	r.ID = strconv.FormatUint(atomic.AddUint64(&c.requestID, 1), 10)
	if text, err := json.Marshal(r); err == nil {
		log.Printf("ws (send): %s : %s", r.Channel, text)
	} else {
		log.Printf("Could not marshal message for sending. %s", err)
	}
	c.mtx.Lock()
	err := c.ws.WriteJSON([]request{*r})
	c.mtx.Unlock()
	return err
}
