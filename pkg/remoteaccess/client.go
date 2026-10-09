package remoteaccess

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/currenttenant"
	"github.com/reubenmiller/go-c8y/v2/pkg/proxy"
)

type RemoteAccessOptions struct {
	ManagedObjectID string
	RemoteAccessID  string

	// Multiplex carries all local connections over a single remote access session (yamux),
	// if the device supports it (e.g. thin-edge.io remote access plugin). Falls back to one
	// remote access session per connection otherwise.
	//
	// Multiplexing is negotiated in-band, so a device without multiplexing support forwards the
	// negotiation request (a few bytes of text) to the target once, before falling back.
	Multiplex bool

	// MultiplexNegotiationTimeout is the maximum time to wait for the device to acknowledge
	// multiplexing. Defaults to DefaultMultiplexNegotiationTimeout
	MultiplexNegotiationTimeout time.Duration
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
	client   *api.Client
	ctx      RemoteAccessOptions
	listener net.Listener
	mux      multiplexer
}

// Create new Remote Access client to allow local clients
// to connect to a device via the Cloud Remote Access feature
func NewRemoteAccessClient(client *api.Client, opt RemoteAccessOptions) *RemoteAccessClient {
	return &RemoteAccessClient{
		client:   client,
		ctx:      opt,
		listener: nil,
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

	if c.client.Auth.Token != "" {
		slog.Debug("Using bearer token")
		requestHeader.Add("Authorization", "Bearer "+c.client.Auth.Token)
	} else {
		slog.Debug("Using basic auth")
		currentTenant := c.client.Tenants.Current.Get(context.Background(), currenttenant.GetOptions{})
		tenantID := ""
		if currentTenant.Err != nil {
			tenantID = currentTenant.Data.ID()
		}
		requestHeader.Add("Authorization", api.NewBasicAuthString(tenantID, c.client.Auth.Username, c.client.Auth.Password))
	}
	slog.Info("Connecting to Cumulocity IoT", "url", remoteURL, "headers", c.client.HideSensitiveInformationIfActive(fmt.Sprintf("%v", requestHeader)))

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
		slog.Error("Could not create remote access connection", "err", err.Error())
		return err
	}
	slog.Info(fmt.Sprintf("Proxying traffic to %v via %v for %v", remoteURL, clientWsConn.RemoteAddr(), "stdio"))

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

	slog.Info("Creating listener", "network", network, "address", localAddress)

	l, err := net.Listen(network, localAddress)
	if err != nil {
		slog.Error("Could not create listener", "network", strings.ToUpper(network), "err", err.Error())
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
	defer c.mux.close()
	for {
		// Listen for an incoming connection.
		tcpConn, err := c.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			slog.Error("Failed to accept incoming connection (ACCEPT)", "err", err.Error())
			// avoid a busy loop on persistent errors (e.g. too many open files)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if c.ctx.Multiplex && c.serveMultiplexed(tcpConn) {
			continue
		}

		clientWsConn, remoteURL, err := c.createRemoteAccessConnection()
		if err != nil {
			slog.Error("Could not create remove access connection", "err", err.Error())
			tcpConn.Close()
			return err
		}
		// Handle connections in a new goroutine.
		slog.Info(fmt.Sprintf("Proxying traffic to %v via %v for %v", remoteURL, clientWsConn.RemoteAddr(), tcpConn.RemoteAddr()))
		go proxy.Copy(clientWsConn, tcpConn)
	}
}
