package remoteaccess

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/logger"
	"github.com/reubenmiller/go-c8y/pkg/proxy"
)

type RemoteAccessOptions struct {
	ManagedObjectID string
	RemoteAccessID  string
}

func parseListenerAddress(v string) (network string, addr string, err error) {
	network = "tcp"

	networkTypeAddress := strings.SplitN(v, "://", 2)
	switch len(networkTypeAddress) {
	case 0:
		err = fmt.Errorf("invalid local address")
	case 1:
		addr = networkTypeAddress[0]
	case 2:
		network = networkTypeAddress[0]
		addr = networkTypeAddress[1]
	}

	return network, addr, err
}

type RemoteAccessClient struct {
	client   *c8y.Client
	ctx      RemoteAccessOptions
	listener net.Listener
	log      logger.Logger
}

// Create new Remote Access client to allow local clients
// to connect to a device via the Cloud Remote Access feature
func NewRemoteAccessClient(client *c8y.Client, opt RemoteAccessOptions, log logger.Logger) *RemoteAccessClient {
	return &RemoteAccessClient{
		client:   client,
		ctx:      opt,
		listener: nil,
		log:      log,
	}
}

func (c *RemoteAccessClient) createRemoteAccessConnection() (*websocket.Conn, string, error) {
	host := c.client.BaseURL.String()
	wsHost := ""
	if strings.HasPrefix(host, "http://") {
		wsHost = "ws://" + host[7:]
	} else if strings.HasPrefix(host, "https://") {
		wsHost = "wss://" + host[8:]
	}
	remoteURL := fmt.Sprintf("%s/service/remoteaccess/client/%s/configurations/%s", strings.TrimRight(wsHost, "/"), c.ctx.ManagedObjectID, c.ctx.RemoteAccessID)

	requestHeader := http.Header{}
	requestHeader.Add("Content-Type", "application/json")

	if c.client.Token != "" {
		c.log.Debug("Using bearer token")
		requestHeader.Add("Authorization", "Bearer "+c.client.Token)
	} else {
		c.log.Debug("Using basic auth")
		requestHeader.Add("Authorization", c8y.NewBasicAuthString(c.client.GetTenantName(context.Background()), c.client.Username, c.client.Password))
	}
	c.log.Infof("Connection to Cumulocity IoT CRA: remote=%s, headers=%v", remoteURL, requestHeader)

	wsConn, _, err := websocket.DefaultDialer.Dial(remoteURL, requestHeader)
	return wsConn, remoteURL, err
}

// Get the listener address. Useful when using the "free port" option, and need
// to know which port the listener chose
func (c *RemoteAccessClient) GetListenerAddress() string {
	if c.listener != nil {
		return c.listener.Addr().String()
	}
	return ""
}

// Listen and serve a single connection. It bridges between the websocket and the given reader/writer
// Typically it can be used to setup proxying to stdin/stdout
func (c *RemoteAccessClient) ListenServe(r io.ReadCloser, w io.Writer) error {
	clientWsConn, remoteURL, err := c.createRemoteAccessConnection()
	if err != nil {
		c.log.Errorf("DIALER: %v", err.Error())
		return err
	}
	c.log.Infof("Proxying traffic to %v via %v for %v", remoteURL, clientWsConn.RemoteAddr(), "stdio")

	// block until finished as stdio mode can not launch multiple instances
	proxy.CopyReadWriter(clientWsConn, r, w)
	return nil
}

// Start a client using which listens to either incoming requests via a TCP or Unix socket
// Set local stream address to listen to
// Example: :8080, 127.0.0.1:8080, 127.0.0.1:0 (first free port)
func (c *RemoteAccessClient) Listen(addr string) error {
	network, localAddress, err := parseListenerAddress(addr)
	if err != nil {
		return err
	}

	c.log.Infof("Creating listener. network=%s, address=%s", network, localAddress)

	l, err := net.Listen(network, localAddress)
	if err != nil {
		c.log.Errorf("%s LISTENER: %v", strings.ToUpper(network), err.Error())
		return err
	}

	c.listener = l
	return nil
}

// Serve requests to the local TCP server or Unix socket
// The Listen must be called prior to trying to serve
func (c *RemoteAccessClient) Serve() error {
	if c.listener == nil {
		return fmt.Errorf("listen must be called before serve")
	}

	// Close the listener when the application closes.
	defer c.listener.Close()
	for {
		// Listen for an incoming connection.
		tcpConn, err := c.listener.Accept()
		if err != nil {
			c.log.Errorf("ACCEPT: %v", err.Error())
		}

		clientWsConn, remoteURL, err := c.createRemoteAccessConnection()
		if err != nil {
			c.log.Errorf("DIALER: %v", err.Error())
			return err
		}
		// Handle connections in a new goroutine.
		c.log.Infof("Proxying traffic to %v via %v for %v", remoteURL, clientWsConn.RemoteAddr(), tcpConn.RemoteAddr())
		go proxy.Copy(clientWsConn, tcpConn)
	}
}
