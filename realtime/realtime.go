package realtime

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/obeattie/ohmyglob"
	tomb "gopkg.in/tomb.v2"
)

const (
	// VERSION prefered Bayeux version
	VERSION = "1.0"

	// MINIMUM_VERSION supported Bayeux version
	MINIMUM_VERSION = "1.0"
)

// Client allows connecting to a Bayeux server and subscribing to channels.
type Client struct {
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
}

// Message is the type delivered to subscribers.
type Message struct {
	Channel   string          `json:"channel"`
	Data      json.RawMessage `json:"data,omitempty"`
	ID        string          `json:"id,omitempty"`
	ClientID  string          `json:"clientId,omitempty"`
	Extension interface{}     `json:"ext,omitempty"`
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

// NewClient initialises a new Bayeux client. By default `http.DefaultClient`
// is used for HTTP connections.
func NewClient(host string, wsDialer *websocket.Dialer, tenant, username, password string) *Client {
	if wsDialer == nil {
		wsDialer = &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  10 * time.Second,
			EnableCompression: false,
		}
	}

	// Convert url to a websocket
	websocketURL := getRealtimURL(host)

	return &Client{
		url:       websocketURL,
		dialer:    wsDialer,
		messages:  make(chan *Message, 100),
		extension: getC8yExtension(tenant, username, password),

		tenant:   tenant,
		username: username,
		password: password,
	}
}

// Connect performs a handshake with the server and will repeatedly initiate a
// websocket connection until `Close` is called on the client.
func (c *Client) Connect() error {
	return c.connect()
}

// Close notifies the Bayeux server of the intent to disconnect and terminates
// the background polling loop.
func (c *Client) Close() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.tomb.Killf("Close")
	c.connected = false
	return c.disconnect()
}

func (c *Client) disconnect() error {
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
func (c *Client) WaitForConnection() error {
	for c.connected == false {
		time.Sleep(100 * time.Millisecond)
	}
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *Client) connect() error {
	dialer := websocket.Dialer{
		Proxy:             http.ProxyFromEnvironment,
		HandshakeTimeout:  10 * time.Second,
		EnableCompression: false,
	}
	ws, _, err := dialer.Dial("wss://nordex.nifqa.nordex-online.com/cep/realtime/", nil)

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

func (c *Client) worker() error {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			messages := []Message{}

			err := c.ws.ReadJSON(&messages)

			if err != nil {
				log.Println("read:", err, messages)
				return
			}

			for _, message := range messages {
				switch channelType := message.Channel; channelType {
				case "/meta/handshake":
					fmt.Printf("ws: Handshake\n")

					if message.ClientID != "" {
						c.mtx.Lock()
						c.clientID = message.ClientID
						c.connected = true
						c.mtx.Unlock()
						fmt.Printf("Detected clientID: %s\n", message.ClientID)
					}

				case "/meta/subscribe":
					fmt.Printf("ws: Subscribe\n")
					c.sendMetaConnect()

				case "/meta/connect":
					fmt.Printf("ws: Connect\n")

				case "/meta/disconnect":
					fmt.Printf("ws: disconnect\n")

				default:
					// Data package received
					for _, s := range c.subscriptions {
						if s.glob.MatchString(message.Channel) {
							s.out <- &message
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
			err := c.ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return err
			}

			return fmt.Errorf("Stopping websocket")
		}
	}
}

func (c *Client) handshake() error {
	handshakeMessage := &request{
		Channel:                  "/meta/handshake",
		Version:                  VERSION,
		MinimumVersion:           MINIMUM_VERSION,
		SupportedConnectionTypes: []string{"websocket", "long-polling"},
		Extension:                c.extension,
		Advice: advice{
			Interval: 0,
			Timeout:  60000,
		},
	}

	return c.writeJSON(handshakeMessage)
}

func (c *Client) sendMetaConnect() error {
	if c.ws == nil {
		return fmt.Errorf("Websocket is nil")
	}
	fmt.Println("Sending meta/connect")
	message := &request{
		ID:             "2",
		Channel:        "/meta/connect",
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	return c.writeJSON(message)
}

const (
	// MEASUREMENTS cep realtime
	MEASUREMENTS = "measurements"

	// ALARMS CEP realtime
	ALARMS = "alarms"

	// OPERATIONS CEP realtime
	OPERATIONS = "operations"
)

// Subscribe setup a subscription to
func (c *Client) Subscribe(pattern string, out chan<- *Message) error {
	fmt.Println("Subscribing to ", pattern)

	glob, err := ohmyglob.Compile(pattern, nil)
	if err != nil {
		return fmt.Errorf("Invalid pattern: %s", err)
	}

	message := &request{
		ID:             "2",
		Channel:        "/meta/subscribe",
		Subscription:   pattern,
		ConnectionType: "websocket",
		ClientID:       c.clientID,
	}

	err = c.writeJSON(message)

	if err == nil {
		c.subscriptions = append(c.subscriptions, subscription{
			glob: glob,
			out:  out,
		})
	}

	return err
}

func (c *Client) writeJSON(r *request) error {
	return c.ws.WriteJSON([]request{*r})
}
