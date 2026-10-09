package remoteaccess

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/multiformats/go-multistream"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/wsconnadapter"
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

// DefaultMultiplexNegotiationTimeout is the maximum time to wait for the device to acknowledge
// multiplexing, unless RemoteAccessOptions.MultiplexNegotiationTimeout is set. It includes the time
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
func (c *RemoteAccessClient) OpenMultiplexedSession() (*yamux.Session, error) {
	wsConn, remoteURL, err := c.createRemoteAccessConnection()
	if err != nil {
		return nil, err
	}
	conn := &countingConn{Conn: wsconnadapter.New(wsConn)}

	timeout := c.ctx.MultiplexNegotiationTimeout
	if timeout <= 0 {
		timeout = DefaultMultiplexNegotiationTimeout
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if err := multistream.SelectProtoOrFail(MultiplexProtocol, conn); err != nil {
		conn.Close()
		if conn.received.Load() > 0 {
			return nil, fmt.Errorf("%w: %v", ErrMultiplexNotSupported, err)
		}
		return nil, fmt.Errorf("%w: %v", errMultiplexNoAnswer, err)
	}
	_ = conn.SetDeadline(time.Time{})

	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard
	session, err := yamux.Client(conn.Conn, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	c8y.Logger.Infof("Multiplexing connections to %v", remoteURL)
	return session, nil
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
func (m *multiplexer) getSession(c *RemoteAccessClient) (*yamux.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.unsupported {
		return nil, ErrMultiplexNotSupported
	}
	if m.session != nil && !m.session.IsClosed() {
		return m.session, nil
	}
	session, err := c.OpenMultiplexedSession()
	if errors.Is(err, errMultiplexNoAnswer) {
		m.noAnswers++
		if m.noAnswers >= multiplexMaxNoAnswers {
			err = fmt.Errorf("%w: no answer to %d requests: %v", ErrMultiplexNotSupported, m.noAnswers, err)
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
func (c *RemoteAccessClient) serveMultiplexed(localConn net.Conn) bool {
	session, err := c.mux.getSession(c)
	if err != nil {
		if errors.Is(err, ErrMultiplexNotSupported) {
			c8y.Logger.Infof("Multiplexing not available, using one websocket per connection. %v", err)
		} else {
			c8y.Logger.Warnf("Could not open multiplexed session, using one websocket per connection. %v", err)
		}
		return false
	}
	stream, err := session.OpenStream()
	if err != nil {
		c8y.Logger.Warnf("Could not open multiplexed stream, using one websocket per connection. %v", err)
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
