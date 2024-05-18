package proxy

import (
	"io"
	"net"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/wsconnadapter"
)

func chanFromConn(conn io.Reader) chan []byte {
	c := make(chan []byte)

	go func() {
		b := make([]byte, 1024)

		for {
			n, err := conn.Read(b)
			if n > 0 {
				res := make([]byte, n)
				// Copy the buffer so it doesn't get changed while read by the recipient.
				copy(res, b[:n])
				c <- res
			}
			if err != nil {
				c <- nil
				break
			}
		}
	}()

	return c
}

// Copy accepts a websocket connection and TCP connection and copies data between them
func Copy(gwsConn *websocket.Conn, tcpConn net.Conn) {
	wsConn := wsconnadapter.New(gwsConn)
	wsChan := chanFromConn(wsConn)
	tcpChan := chanFromConn(tcpConn)

	defer wsConn.Close()
	defer tcpConn.Close()
	for {
		select {
		case wsData := <-wsChan:
			if wsData == nil {
				c8y.Logger.Infof("Connection closed: D: %v, S: %v", tcpConn.LocalAddr(), wsConn.RemoteAddr())
				return
			} else {
				tcpConn.Write(wsData)
			}
		case tcpData := <-tcpChan:
			if tcpData == nil {
				c8y.Logger.Infof("Connection closed: D: %v, S: %v", tcpConn.LocalAddr(), wsConn.LocalAddr())
				return
			} else {
				wsConn.Write(tcpData)
			}
		}
	}

}

// Copy accepts a websocket connection and read/writer and copies data between them
func CopyReadWriter(gwsConn *websocket.Conn, r io.ReadCloser, w io.Writer) {
	wsConn := wsconnadapter.New(gwsConn)
	wsChan := chanFromConn(wsConn)
	stdioChan := chanFromConn(r)

	defer wsConn.Close()
	defer r.Close()
	for {
		select {
		case wsData := <-wsChan:
			if wsData == nil {
				c8y.Logger.Infof("STDIO connection closed: D: %v, S: %v", "stdio", wsConn.RemoteAddr())
				return
			} else {
				w.Write(wsData)
			}
		case tcpData := <-stdioChan:
			if tcpData == nil {
				c8y.Logger.Infof("STDIO connection closed: D: %v, S: %v", "stdio", wsConn.LocalAddr())
				return
			} else {
				wsConn.Write(tcpData)
			}
		}
	}
}
