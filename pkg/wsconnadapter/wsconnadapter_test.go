package wsconnadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

// Reads must return data or an error, never (0, nil), also at the boundary between websocket
// messages: many readers (e.g. io.ReadFull with byte readers) treat that as a lack of progress.
func TestReadContinuesWithTheNextMessage(t *testing.T) {
	messages := []string{"first", "second", "third"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for _, message := range messages {
			_ = conn.WriteMessage(websocket.BinaryMessage, []byte(message))
		}
		_, _, _ = conn.ReadMessage()
	}))
	defer server.Close()

	wsConn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	conn := New(wsConn)
	defer conn.Close()

	want := strings.Join(messages, "")
	var got []byte
	buf := make([]byte, 1)
	for len(got) < len(want) {
		n, err := conn.Read(buf)
		if err != nil {
			t.Fatalf("read failed after %q: %v", got, err)
		}
		if n == 0 {
			t.Fatalf("read returned no data and no error after %q", got)
		}
		got = append(got, buf[:n]...)
	}
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
