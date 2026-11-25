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
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/wsdial"
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

	// Logger receives the client's messages. nil uses the logger of the
	// Cumulocity client (see api.Client.Logger).
	Logger *slog.Logger

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
	log      *slog.Logger

	dryRunMu      sync.Mutex
	dryRunRequest *http.Request
}

// Create new Remote Access client to allow local clients
// to connect to a device via the Cloud Remote Access feature
func NewRemoteAccessClient(client *api.Client, opt RemoteAccessOptions) *RemoteAccessClient {
	log := opt.Logger
	if log == nil {
		log = client.Logger()
	}
	return &RemoteAccessClient{
		client:   client,
		ctx:      opt,
		listener: nil,
		log:      log,
	}
}

// remoteAccessURL is the websocket URL of the remote access configuration.
func (c *RemoteAccessClient) remoteAccessURL() string {
	host := c.client.BaseURL.String()
	wsHost := ""
	if strings.HasPrefix(host, "http://") {
		wsHost = "ws://" + host[7:]
	} else if strings.HasPrefix(host, "https://") {
		wsHost = "wss://" + host[8:]
	}
	return fmt.Sprintf("%s/service/remoteaccess/client/%s/configurations/%s", strings.TrimRight(wsHost, "/"), c.ctx.ManagedObjectID, c.ctx.RemoteAccessID)
}

// DialContext opens one websocket to the remote access configuration, using the
// connection settings of the Cumulocity client: its TLS and proxy settings, the
// current authorization (see api.Client.WebsocketHeader), request middleware and
// the context options of ctx. Cancelling ctx aborts connecting.
//
// Dry run: when ctx has dry run enabled (api.WithDryRun) nothing is connected. The
// upgrade request is handed to the dry-run handler (api.WithDryRunHandler) or
// logged, like an HTTP request, and the error returned matches
// wsdial.ErrDryRun; wsdial.DryRunRequest(err) is the request, and
// [RemoteAccessClient.DryRunRequest] returns it too.
func (c *RemoteAccessClient) DialContext(ctx context.Context) (*websocket.Conn, error) {
	conn, _, err := c.createRemoteAccessConnection(ctx)
	return conn, err
}

// DryRunRequest returns the upgrade request which the last dry-run connection
// would have sent, or nil if there was none.
func (c *RemoteAccessClient) DryRunRequest() *http.Request {
	c.dryRunMu.Lock()
	defer c.dryRunMu.Unlock()
	return c.dryRunRequest
}

func (c *RemoteAccessClient) createRemoteAccessConnection(ctx context.Context) (*websocket.Conn, string, error) {
	remoteURL := c.remoteAccessURL()

	requestHeader := http.Header{}
	requestHeader.Add("Content-Type", "application/json")

	c.log.Info("Connecting to Cumulocity IoT", "url", remoteURL)

	wsConn, _, err := c.client.DialWebsocket(ctx, remoteURL, requestHeader)
	if err != nil {
		if req := wsdial.DryRunRequest(err); req != nil {
			c.dryRunMu.Lock()
			c.dryRunRequest = req
			c.dryRunMu.Unlock()
		}
		return nil, remoteURL, err
	}
	return wsConn, remoteURL, nil
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
	return c.ListenServeContext(context.Background(), r, w)
}

// ListenServeContext is like ListenServe. ctx is used to connect (see
// DialContext); under dry run nothing is connected and it returns nil, the
// request is available from DryRunRequest.
func (c *RemoteAccessClient) ListenServeContext(ctx context.Context, r io.ReadCloser, w io.Writer) error {
	clientWsConn, remoteURL, err := c.createRemoteAccessConnection(ctx)
	if wsdial.IsDryRun(err) {
		return nil
	}
	if err != nil {
		c.log.Error("Could not create remote access connection", "err", err.Error())
		return err
	}
	c.log.Info(fmt.Sprintf("Proxying traffic to %v via %v for %v", remoteURL, clientWsConn.RemoteAddr(), "stdio"))

	// block until finished as stdio mode can not launch multiple instances
	proxy.CopyReadWriterWithLogger(c.log, clientWsConn, r, w)
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

	c.log.Info("Creating listener", "network", network, "address", localAddress)

	l, err := net.Listen(network, localAddress)
	if err != nil {
		c.log.Error("Could not create listener", "network", strings.ToUpper(network), "err", err.Error())
		return err
	}

	c.listener = l
	return nil
}

// Serve requests to the local TCP server or Unix socket
// The Listen must be called prior to trying to serve
func (c *RemoteAccessClient) Serve() error {
	return c.ServeContext(context.Background())
}

// ServeContext is like Serve. ctx is used to connect for every local connection
// (see DialContext), and cancelling it stops serving: the listener is closed
// and ServeContext returns ctx.Err().
//
// Under dry run (api.WithDryRun) it does not accept connections: it reports the
// upgrade request once (see DialContext), makes it available from DryRunRequest
// and returns nil.
func (c *RemoteAccessClient) ServeContext(ctx context.Context) error {
	if c.listener == nil {
		return fmt.Errorf("listen must be called before serve")
	}

	// Close the listener when the application closes.
	defer c.listener.Close()
	defer c.mux.close()

	if api.IsDryRun(ctx) {
		_, _, err := c.createRemoteAccessConnection(ctx)
		if err != nil && !wsdial.IsDryRun(err) {
			return err
		}
		return nil
	}

	stop := context.AfterFunc(ctx, func() { _ = c.listener.Close() })
	defer stop()
	for {
		// Listen for an incoming connection.
		tcpConn, err := c.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			c.log.Error("Failed to accept incoming connection (ACCEPT)", "err", err.Error())
			// avoid a busy loop on persistent errors (e.g. too many open files)
			time.Sleep(100 * time.Millisecond)
			continue
		}

		if c.ctx.Multiplex && c.serveMultiplexed(ctx, tcpConn) {
			continue
		}

		clientWsConn, remoteURL, err := c.createRemoteAccessConnection(ctx)
		if err != nil {
			c.log.Error("Could not create remove access connection", "err", err.Error())
			tcpConn.Close()
			return err
		}
		// Handle connections in a new goroutine.
		c.log.Info(fmt.Sprintf("Proxying traffic to %v via %v for %v", remoteURL, clientWsConn.RemoteAddr(), tcpConn.RemoteAddr()))
		go proxy.CopyWithLogger(c.log, clientWsConn, tcpConn)
	}
}
