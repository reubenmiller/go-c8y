package remoteaccess

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hashicorp/yamux"
	"github.com/multiformats/go-multistream"
	"github.com/reubenmiller/go-c8y/v2/pkg/wsconnadapter"
)

// Multiplexing (e.g. thin-edge.io remote access plugin): many local connections are carried over
// a single remote access websocket using yamux (https://github.com/hashicorp/yamux/blob/master/spec.md),
// instead of opening a new websocket (and remote access operation) per connection.
//
// The mode is negotiated in-band with multistream-select
// (https://github.com/multiformats/multistream-select): the client proposes MultiplexProtocol and
// the yamux session starts once the device confirms it. Devices without multiplexing support
// forward the negotiation to the target instead, so any other answer means "not supported".
const MultiplexProtocol = "/yamux/1.0.0"

// ErrMultiplexNotSupported is returned when the device did not acknowledge the multiplexing request
var ErrMultiplexNotSupported = errors.New("remote access multiplexing is not supported by the device")

// errMultiplexNoAnswer is returned when the device did not answer the multiplexing request, which
// does not tell whether the device supports it (e.g. a slow device, or a target which waits for more data)
var errMultiplexNoAnswer = errors.New("no answer to the remote access multiplexing request")

// errMultiplexDisabled is returned once multiplexing was disabled for the client, so the reason is
// only reported once
var errMultiplexDisabled = fmt.Errorf("%w: multiplexing was disabled", ErrMultiplexNotSupported)

// DefaultMultiplexNegotiationTimeout is the maximum time to wait for the device to acknowledge
// multiplexing, unless ClientOptions.MultiplexNegotiationTimeout is set. It includes the time
// for the device to connect to the remote access session.
var DefaultMultiplexNegotiationTimeout = 15 * time.Second

// multiplexMaxNoAnswers is the number of consecutive multiplexing requests without an answer after
// which the device is considered not to support multiplexing
const multiplexMaxNoAnswers = 3

// multiplexHalfCloseTimeout is how long the local side of a stream may keep sending after the
// remote side has finished, before the local connection is closed
const multiplexHalfCloseTimeout = 30 * time.Second

// multiplexer manages the multiplexed session of a client. The session is (re)opened on demand.
type multiplexer struct {
	mu          sync.Mutex
	session     *yamux.Session
	unsupported bool
	noAnswers   int
}

// OpenMultiplexedSession opens a remote access websocket and negotiates multiplexing with the device.
// It returns ErrMultiplexNotSupported if the device answered with anything other than an
// acknowledgement. If the device did not answer (in time), e.g. the websocket was closed before,
// the error wraps errMultiplexNoAnswer.
func (c *Client) OpenMultiplexedSession() (*yamux.Session, error) {
	return c.OpenMultiplexedSessionContext(context.Background())
}

// OpenMultiplexedSessionContext is like OpenMultiplexedSession; ctx is used to
// connect (see DialContext).
func (c *Client) OpenMultiplexedSessionContext(ctx context.Context) (*yamux.Session, error) {
	wsConn, remoteURL, err := c.createRemoteAccessConnection(ctx)
	if err != nil {
		return nil, err
	}
	conn := &countingConn{Conn: wsconnadapter.New(wsConn)}

	timeout := c.ctx.MultiplexNegotiationTimeout
	if timeout <= 0 {
		timeout = DefaultMultiplexNegotiationTimeout
	}
	started := time.Now()
	_ = conn.SetDeadline(started.Add(timeout))
	if err := multistream.SelectProtoOrFail(MultiplexProtocol, conn); err != nil {
		conn.Close()
		elapsed := time.Since(started).Round(time.Millisecond)
		if conn.received.Load() > 0 {
			return nil, fmt.Errorf("%w: the device answered with something else after %s: %v", ErrMultiplexNotSupported, elapsed, err)
		}
		return nil, fmt.Errorf("%w: %s", errMultiplexNoAnswer, describeNoAnswer(err, elapsed, timeout))
	}
	_ = conn.SetDeadline(time.Time{})

	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard
	session, err := yamux.Client(conn.Conn, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	c.log.Info("Multiplexing connections", "url", remoteURL)
	return session, nil
}

// describeNoAnswer explains why the device did not answer the multiplexing request
func describeNoAnswer(err error, elapsed, timeout time.Duration) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Sprintf("no answer within %s", timeout)
	}
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) || errors.Is(err, io.EOF) {
		// e.g. a device without multiplexing support forwards the request to the target, which
		// closes the connection, or (thin-edge.io <= 2.1.0) the target dropped its idle connection
		// before the request arrived, so the device selected the passthrough mode
		return fmt.Sprintf("the remote access session was closed after %s without an answer, e.g. the device does not support multiplexing, or the target closed its connection before the request arrived: %v", elapsed, err)
	}
	return fmt.Sprintf("failed after %s: %v", elapsed, err)
}

// countingConn counts the bytes received, to tell whether the device answered at all
type countingConn struct {
	net.Conn
	received atomic.Int64
}

func (c *countingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.received.Add(int64(n))
	return n, err
}

// getSession returns the open multiplexed session, opening a new one if required
func (m *multiplexer) getSession(ctx context.Context, c *Client) (*yamux.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unsupported {
		return nil, errMultiplexDisabled
	}
	if m.session != nil && !m.session.IsClosed() {
		return m.session, nil
	}
	session, err := c.OpenMultiplexedSessionContext(ctx)
	if errors.Is(err, errMultiplexNoAnswer) {
		m.noAnswers++
		if m.noAnswers >= multiplexMaxNoAnswers {
			err = fmt.Errorf("%w: no answer to %d requests, last error: %v", ErrMultiplexNotSupported, m.noAnswers, err)
		} else {
			err = fmt.Errorf("%w (attempt %d of %d, multiplexing is disabled after %d attempts without an answer)", err, m.noAnswers, multiplexMaxNoAnswers, multiplexMaxNoAnswers)
		}
	}
	if errors.Is(err, ErrMultiplexNotSupported) {
		m.unsupported = true
	}
	if err != nil {
		return nil, err
	}
	m.noAnswers = 0
	m.session = session
	return session, nil
}

func (m *multiplexer) close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.session != nil {
		m.session.Close()
	}
}

// serveMultiplexed carries a local connection as a stream of the multiplexed session.
// It returns false if the connection could not be multiplexed (and was not used).
func (c *Client) serveMultiplexed(ctx context.Context, localConn net.Conn) bool {
	session, err := c.mux.getSession(ctx, c)
	if err != nil {
		switch {
		case errors.Is(err, errMultiplexDisabled):
			c.log.Debug("Multiplexing disabled, using one websocket per connection")
		case errors.Is(err, ErrMultiplexNotSupported):
			c.log.Warn("Multiplexing disabled, using one websocket per connection", "err", err)
		default:
			c.log.Warn("Could not open multiplexed session, using one websocket for this connection", "err", err)
		}
		return false
	}
	stream, err := session.OpenStream()
	if err != nil {
		c.log.Warn("Could not open multiplexed stream, using one websocket per connection", "err", err)
		return false
	}
	go pipe(localConn, stream, multiplexHalfCloseTimeout)
	return true
}

// pipe copies data in both directions until both sides are done, propagating half-closes.
// Once the remote side has finished, the local side has halfCloseTimeout to finish too.
func pipe(localConn net.Conn, stream *yamux.Stream, halfCloseTimeout time.Duration) {
	defer localConn.Close()
	defer stream.Close()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(stream, localConn)
		// no more data from the local side: close the stream for writing (yamux FIN)
		_ = stream.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(localConn, stream)
		if conn, ok := localConn.(interface{ CloseWrite() error }); ok {
			_ = conn.CloseWrite()
			// don't wait forever for a local client which ignores the half-close
			_ = localConn.SetReadDeadline(time.Now().Add(halfCloseTimeout))
		} else {
			_ = localConn.Close()
		}
	}()
	wg.Wait()
}
