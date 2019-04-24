package c8y

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
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
}

// Message is the type delivered to subscribers.
type Message struct {
	Channel   string       `json:"channel"`
	Payload   RealtimeData `json:"data,omitempty"`
	ID        string       `json:"id,omitempty"`
	ClientID  string       `json:"clientId,omitempty"`
	Extension interface{}  `json:"ext,omitempty"`
}

// RealtimeData contains the websocket frame data
type RealtimeData struct {
	RealtimeAction string          `json:"realtimeAction,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`

	Item gjson.Result `json:"-"`
}

type subscription struct {
	glob ohmyglob.Glob
	out  chan<- *Message
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
	Advice                   advice          `json:"advice"`
}

type advice struct {
	Reconnect string `json:"reconnect,omitempty"`
	Timeout   int64  `json:"timeout,omitempty"`
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
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.tomb.Killf("Close")
	c.connected = false
	return c.disconnect()
}

func (c *RealtimeClient) disconnect() error {
	message := &request{
		Channel:  "/meta/disconnect",
		ClientID: c.clientID,
	}

	err := c.ws.WriteJSON(message)

	if err != nil {
		return err
	}

	return nil
}

// WaitForConnection wait for the connection to be estabilished before returning
func (c *RealtimeClient) WaitForConnection() error {
	for c.connected == false {
		// TODO: only return once the connection has been made instead of fixed delay
		time.Sleep(250 * time.Millisecond)
	}
	return nil
}

func (c *RealtimeClient) createWebsocket() error {
	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
	}
	log.Printf("Establishing connection to %s", c.url.String())
	ws, _, err := dialer.Dial(c.url.String(), nil)

	if err != nil {
		return err
	}
	c.mtx.Lock()
	c.ws = ws
	c.mtx.Unlock()
	return nil
}

func (c *RealtimeClient) reconnect() error {
	c.ws.Close()
	err := c.createWebsocket()

	if err != nil {
		c.mtx.Unlock()
		c.connected = false
		c.mtx.Lock()
		return err
	}
	c.sendMetaConnect()
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *RealtimeClient) connect() error {
	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
	}
	log.Printf("Establishing connection to %s", c.url.String())
	ws, _, err := dialer.Dial(c.url.String(), nil)

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
				if !c.IsConnected() {
					log.Println("Connection has been closed by the client")
					return
				}
				log.Println("read:", err, messages)
				log.Println("Handling connection error. You need to reconnect")

				// Try to reconnect with a new websocket
				c.reconnect()
				// return
			}

			for _, message := range messages {
				switch channelType := message.Channel; channelType {
				case "/meta/handshake":
					log.Printf("ws: Handshake\n")

					if message.ClientID != "" {
						c.mtx.Lock()
						c.clientID = message.ClientID
						c.connected = true
						c.mtx.Unlock()
						log.Printf("Detected clientID: %s\n", message.ClientID)
					} else {
						log.Panicf("No clientID present in handshake. Check that the tenant, usename and password is correct. Raw Message: %s\n", message)
					}

				case "/meta/subscribe":
					log.Printf("ws: Subscribe\n")
					if !c.IsConnected() {
						c.sendMetaConnect()
					}

				case "/meta/connect":
					c.mtx.Lock()
					c.connected = true
					c.mtx.Unlock()
					log.Printf("ws: Connect\n")

				case "/meta/disconnect":
					log.Printf("ws: disconnect\n")

				default:
					// log.Printf("ws: data: %s\n", message)
					// Data package received
					message.Payload.Item = gjson.ParseBytes(message.Payload.Data)
					c.mtx.RLock()
					for _, s := range c.subscriptions {
						if s.glob.MatchString(message.Channel) {
							s.out <- &message
						}
					}
					c.mtx.RUnlock()
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
			err := c.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return err
			}

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
		Advice: advice{
			Interval: 0,
			Timeout:  60000,
		},
	}

	return c.writeJSON(handshakeMessage)
}

func (c *RealtimeClient) sendMetaConnect() error {
	if c.ws == nil {
		return fmt.Errorf("Websocket is nil")
	}
	log.Print("Sending meta/connect")
	message := &request{
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	return c.writeJSON(message)
}

func getRealtimeID(id ...string) string {
	if len(id) > 0 {
		return id[0]
	}
	return "*"
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

// Subscribe setup a subscription to
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

	err = c.writeJSON(message)

	if err == nil {
		c.mtx.Lock()
		c.subscriptions = append(c.subscriptions, subscription{
			glob: glob,
			out:  out,
		})
		c.mtx.Unlock()
	}
	return err
}

// UnsubscribeAll unsubscribes to all of the subscribed channels
func (c *RealtimeClient) UnsubscribeAll() (errs []error) {

	for _, pattern := range c.subscriptions {
		err := c.Unsubscribe(pattern.glob.String())
		if err != nil {
			errs = append(errs, err)
		}
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

	err = c.writeJSON(message)

	if err != nil {
		log.Printf("Failed to unsubscribe to subscription. %s", err)
	}

	return err
}

func (c *RealtimeClient) writeJSON(r *request) error {
	// Add a unique request id
	r.ID = strconv.FormatUint(atomic.AddUint64(&c.requestID, 1), 10)
	return c.ws.WriteJSON([]request{*r})
}
