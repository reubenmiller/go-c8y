package notification2

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/pkg/logger"
	"github.com/tidwall/gjson"
	tomb "gopkg.in/tomb.v2"
)

var Logger logger.Logger

func init() {
	Logger = logger.NewLogger("notifications2")
}

const (
	// MaximumRetryInterval is the maximum interval (in seconds) between reconnection attempts
	MaximumRetryInterval int64 = 300

	// MinimumRetryInterval is the minimum interval (in seconds) between reconnection attempts
	MinimumRetryInterval int64 = 5

	// RetryBackoffFactor is the backoff factor applied to the retry interval for every unsuccessful reconnection attempt.
	// i.e. the next retry interval is calculated as followins
	// interval = MinimumRetryInterval
	// interval = Min(MaximumRetryInterval, interval * RetryBackoffFactor)
	RetryBackoffFactor float64 = 2
)

const (
	writeWait = 10 * time.Second

	pongWait = 30 * time.Second

	pingPeriod = (pongWait * 9) / 10
)

func SetLogger(log logger.Logger) {
	if log == nil {
		Logger = logger.NewDummyLogger("notification2")
	} else {
		Logger = log
	}
}

// Notification2Client is a client used for the notification2 interface
type Notification2Client struct {
	mtx          sync.RWMutex
	host         string
	url          *url.URL
	tomb         *tomb.Tomb
	messages     chan *Message
	connected    bool
	dialer       *websocket.Dialer
	ws           *websocket.Conn
	Subscription Subscription

	hub  *Hub
	send chan []byte
}

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
	Identifier  []byte `json:"identifier"`
	Description []byte `json:"description"`
	Action      []byte `json:"action"`
	Payload     []byte `json:"data,omitempty"`
}

type ActionType string

var ActionTypeCreate ActionType = "CREATE"
var ActionTypeUpdate ActionType = "UPDATE"
var ActionTypeDelete ActionType = "DELETE"

func (m *Message) JSON() gjson.Result {
	return gjson.ParseBytes(m.Payload)
}

func getEndpoint(host string, subscription Subscription) *url.URL {
	fullHost := "wss://" + host
	if index := strings.Index(host, "://"); index > -1 {
		fullHost = "wss" + host[index:]
	}
	c8yhost, _ := url.Parse(fullHost)
	c8yhost.Path = "/notification2/consumer/"
	c8yhost.RawQuery = "token=" + subscription.Token

	if subscription.Consumer != "" {
		c8yhost.RawQuery += "&consumer=" + subscription.Consumer
	}

	return c8yhost
}

// NewNotification2Client initialises a new notification2 client used to subscribe to realtime notifications from Cumulocity
func NewNotification2Client(host string, wsDialer *websocket.Dialer, subscription Subscription) *Notification2Client {
	if wsDialer == nil {
		// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
		wsDialer = &websocket.Dialer{
			Proxy:             http.ProxyFromEnvironment,
			HandshakeTimeout:  45 * time.Second,
			EnableCompression: false,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	client := &Notification2Client{
		host:         host,
		url:          getEndpoint(host, subscription),
		dialer:       wsDialer,
		messages:     make(chan *Message, 100),
		Subscription: subscription,

		send: make(chan []byte),

		hub: NewHub(),
	}

	go client.hub.Run()
	go client.writeHandler()
	return client
}

// Connect performs a handshake with the server and will repeatedly initiate a
// websocket connection until `Close` is called on the client.
func (c *Notification2Client) Connect() error {
	if !c.IsConnected() {
		err := c.connect()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Notification2Client) Endpoint() string {
	return fmt.Sprintf("%s%s", c.url.Host, c.url.Path)
}

func (c *Notification2Client) URL() string {
	return getEndpoint(c.url.Host, c.Subscription).String()
}

// IsConnected returns true if the websocket is connected
func (c *Notification2Client) IsConnected() bool {
	c.mtx.RLock()
	isConnected := c.connected
	c.mtx.RUnlock()
	return isConnected
}

// Close the connection
func (c *Notification2Client) Close() error {
	if err := c.disconnect(); err != nil {
		Logger.Warnf("Failed to disconnect. %s", err)
	}
	c.mtx.Lock()
	defer c.mtx.Unlock()
	if c.tomb != nil {
		Logger.Debugf("Stopping worker")
		c.tomb.Killf("Close")
		c.tomb = nil
	}
	return nil
}

func (c *Notification2Client) disconnect() error {
	// Change to disconnected state, as the server will not send a reply upon receiving the /meta/disconnect command
	c.mtx.Lock()
	defer c.mtx.Unlock()
	c.connected = false

	// TODO: Add option to unsubscribe on disconnect (e.g. no offline messages required?)
	// Note: If you unsubscribe, then notifications will be ignored when the client is offline
	// if c.ws != nil {
	// 	if err := c.Unsubscribe(); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

func (c *Notification2Client) createWebsocket() (*websocket.Conn, error) {

	if c.Subscription.TokenRenewal != nil {
		token, err := c.Subscription.TokenRenewal(c.Subscription.Token)
		if err != nil {
			return nil, err
		}
		c.Subscription.Token = token
	}

	Logger.Debugf("Establishing connection to %s", c.Endpoint())
	ws, _, err := c.dialer.Dial(c.URL(), nil)

	if err != nil {
		Logger.Warnf("Failed to establish connection. %s", err)
		return ws, err
	}
	Logger.Debugf("Established websocket connection. %s", err)
	return ws, nil
}

func (c *Notification2Client) reconnect() error {
	c.Close()

	connected := false
	interval := MinimumRetryInterval

	for !connected {
		Logger.Warnf("Retrying in %ds", interval)
		<-time.After(time.Duration(interval) * time.Second)
		err := c.connect()

		if err != nil {
			Logger.Warnf("Failed to connect. %s", err)
			interval = int64(math.Min(float64(MaximumRetryInterval), RetryBackoffFactor*float64(interval)))
			continue
		}

		connected = true
	}

	Logger.Warn("Reestablished connection")
	return nil
}

// StartWebsocket opens a websocket to cumulocity
func (c *Notification2Client) connect() error {
	if c.dialer == nil {
		panic("Missing dialer for realtime client")
	}
	ws, err := c.createWebsocket()

	if err != nil {
		return err
	}

	c.mtx.Lock()
	defer c.mtx.Unlock()

	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.ws = ws
	if c.tomb == nil {
		c.tomb = &tomb.Tomb{}
		c.tomb.Go(c.worker)
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
		line := scanner.Bytes()
		if len(line) == 0 {
			inHeader = false
		}
		if inHeader {
			if i == 0 {
				message.Identifier = line
			} else if i == 1 {
				message.Description = line
			} else if i == 2 {
				message.Action = line
			}
			// Ignore unknown header indexes
		} else {
			message.Payload = line
		}
		i++
	}
	return message
}

func (c *Notification2Client) writeHandler() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		close(c.send)
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				// The send channel has been closed
				Logger.Info("Channel has been closed")
				return
			}

			if c.ws != nil {
				if err := c.ws.WriteMessage(websocket.TextMessage, message); err != nil {
					Logger.Warnf("Failed to send message. %s", err)
				}
			}

		case <-ticker.C:
			// Regularly check if the Websocket is alive by sending a PingMessage to the server
			if c.ws != nil {
				// A websocket ping should initiate a websocket pong response from the server
				// If the pong is not received in the minimum time, then the connection will be reset
				c.ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					Logger.Warnf("Failed to send ping message to server")
					// go c.reconnect()
					return
				}
				Logger.Debug("Sent ping successfully")
			}
		}
	}
}

func (c *Notification2Client) Register(pattern string, out chan<- Message) {
	Logger.Debugf("Subscribing to %s", pattern)

	c.hub.register <- &ClientSubscription{
		Pattern:  pattern,
		Out:      out,
		Disabled: false,
	}
}

func (c *Notification2Client) SendMessageAck(messageIdentifier []byte) error {
	Logger.Debugf("Sending message ack: %s", messageIdentifier)
	return c.ws.WriteMessage(websocket.TextMessage, messageIdentifier)
}

func (c *Notification2Client) worker() error {
	done := make(chan struct{})

	c.ws.SetPongHandler(func(appData string) error {
		Logger.Debugf("Pong handler. %v", appData)
		c.ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	c.ws.SetPingHandler(func(appData string) error {
		Logger.Debugf("Ping handler. %v", appData)
		return nil
	})

	go func() {
		defer close(done)
		for {
			messageType, rawMessage, err := c.ws.ReadMessage()

			Logger.Debugf("Received message: %s", rawMessage)

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
					Logger.Infof("error %v", err)
				}

				Logger.Warnf("err: %s", err)
				go c.reconnect()
				break
			}

			if messageType == websocket.TextMessage {
				message := parseMessage(rawMessage)

				c.hub.broadcast <- *message

				Logger.Debugf("message id: %s", message.Identifier)
				Logger.Debugf("message description: %s", message.Description)
				Logger.Debugf("message action: %s", message.Action)
				Logger.Debugf("message payload: %s", message.Payload)
			}
		}
	}()

	defer c.ws.Close()
	<-c.tomb.Dying()
	Logger.Info("Worker is shutting down")
	return nil
}

// Unsubscribe unsubscribe to a given pattern
func (c *Notification2Client) Unsubscribe() error {
	Logger.Info("unsubscribing")
	return c.ws.WriteMessage(websocket.TextMessage, []byte("unsubscribe_subscriber"))
}
