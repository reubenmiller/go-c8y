package realtime

import (
	"context"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

func testStream(msgs ...[2]string) *Stream[StreamData[jsondoc.JSONDoc]] {
	ch := make(chan *Message, len(msgs))
	for _, m := range msgs {
		msg := &Message{Channel: "/alarms/1"}
		msg.Payload.RealtimeAction = m[0]
		msg.Payload.Data = jsondoc.New([]byte(m[1]))
		ch <- msg
	}
	return NewStream(context.Background(), ch, nil, func(msg *Message) StreamData[jsondoc.JSONDoc] {
		return StreamData[jsondoc.JSONDoc]{Action: msg.Payload.RealtimeAction, Channel: msg.Channel, Data: msg.Payload.Data}
	}, nil)
}

func read(s *Stream[StreamData[jsondoc.JSONDoc]]) (docs []string, errs []error) {
	for d, err := range s.Documents() {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		docs = append(docs, string(d.Raw()))
	}
	return
}

func TestStreamDocuments(t *testing.T) {
	msgs := [][2]string{{"CREATE", `{"id":"1"}`}, {"UPDATE", `{"id":"2"}`}, {"CREATE", `{"id":"3"}`}}

	docs, errs := read(testStream(msgs...).WithOptions(StreamOptions{MaxMessages: 2, Timeout: time.Second}))
	if len(errs) != 0 || len(docs) != 2 || docs[1] != `{"id":"2"}` {
		t.Errorf("max messages: docs=%v errs=%v", docs, errs)
	}

	docs, errs = read(testStream(msgs...).WithOptions(StreamOptions{Actions: []string{"create"}, Timeout: 200 * time.Millisecond}))
	if len(errs) != 0 || len(docs) != 2 || docs[0] != `{"id":"1"}` || docs[1] != `{"id":"3"}` {
		t.Errorf("actions: docs=%v errs=%v", docs, errs)
	}

	start := time.Now()
	docs, errs = read(testStream().WithOptions(StreamOptions{Timeout: 100 * time.Millisecond}))
	if len(errs) != 0 || len(docs) != 0 || time.Since(start) > 2*time.Second {
		t.Errorf("timeout must end the stream without an error: docs=%v errs=%v", docs, errs)
	}
}
