package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"strings"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// GetJSONDoc returns the message payload as a JSON document, so StreamData is
// a jsondoc.Unwrapper like the other models.
func (d StreamData[T]) GetJSONDoc() jsondoc.JSONDoc {
	if v, ok := any(d.Data).(jsondoc.Unwrapper); ok {
		return v.GetJSONDoc() // includes jsondoc.JSONDoc itself
	}
	b, err := json.Marshal(d.Data)
	if err != nil {
		return jsondoc.Empty()
	}
	return jsondoc.New(b)
}

// RealtimeAction returns the action that triggered the message (e.g. CREATE).
func (d StreamData[T]) RealtimeAction() string { return d.Action }

// StreamOptions limit how long a stream is read by Documents.
type StreamOptions struct {
	// Actions keeps only messages with one of these realtime actions
	// (case-insensitive, e.g. "CREATE", "UPDATE", "DELETE"). Empty keeps all.
	Actions []string
	// MaxMessages stops the stream after this many messages. 0 is unlimited.
	MaxMessages int64
	// Timeout stops the stream after this duration. 0 is unlimited.
	Timeout time.Duration
}

// WithOptions sets the limits Documents applies and returns the stream.
func (s *Stream[T]) WithOptions(opts StreamOptions) *Stream[T] {
	s.opts = opts
	return s
}

type actioner interface{ RealtimeAction() string }

// Documents reads the stream as JSON documents (see op.DocumentSource) until
// the subscription ends, MaxMessages documents were read or Timeout passed,
// then closes the stream. Reaching the timeout or cancelling the context ends
// the stream normally, without an error.
func (s *Stream[T]) Documents() iter.Seq2[jsondoc.JSONDoc, error] {
	return func(yield func(jsondoc.JSONDoc, error) bool) {
		defer s.Close()
		if s.opts.Timeout > 0 {
			timer := time.AfterFunc(s.opts.Timeout, func() {
				if s.cancel != nil {
					s.cancel()
				}
			})
			defer timer.Stop()
		}
		actions := map[string]bool{}
		for _, a := range s.opts.Actions {
			if a = strings.TrimSpace(a); a != "" {
				actions[strings.ToUpper(a)] = true
			}
		}
		var count int64
		for item, err := range s.Items() {
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return
				}
				yield(jsondoc.Empty(), err)
				return
			}
			if len(actions) > 0 {
				if a, ok := any(item).(actioner); ok && !actions[strings.ToUpper(a.RealtimeAction())] {
					continue
				}
			}
			u, ok := any(item).(jsondoc.Unwrapper)
			if !ok {
				continue
			}
			if !yield(u.GetJSONDoc(), nil) {
				return
			}
			count++
			if s.opts.MaxMessages > 0 && count >= s.opts.MaxMessages {
				return
			}
		}
	}
}

// SubscribeStream subscribes to one or more channel patterns (e.g.
// Alarms("*"), Measurements("12345")) and returns one stream of their
// messages, with the payload as a JSON document. Use it for channels that no
// typed service covers, or to listen to several kinds of data at once.
func (c *Client) SubscribeStream(ctx context.Context, patterns ...string) op.Result[*Stream[StreamData[jsondoc.JSONDoc]]] {
	if err := c.Connect(); err != nil {
		return op.Failed[*Stream[StreamData[jsondoc.JSONDoc]]](err, false)
	}

	merged := make(chan *Message, 10)
	acks := make(chan error, 1)
	var wg sync.WaitGroup
	var ackErrs []error
	var ackMu sync.Mutex
	var ackWG sync.WaitGroup
	for _, pattern := range patterns {
		in := make(chan *Message, 10)
		errCh := c.Subscribe(ctx, pattern, in)
		ackWG.Add(1)
		go func() {
			defer ackWG.Done()
			select {
			case err, ok := <-errCh:
				if ok && err != nil {
					ackMu.Lock()
					ackErrs = append(ackErrs, err)
					ackMu.Unlock()
				}
			case <-ctx.Done():
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case msg, ok := <-in:
					if !ok {
						return
					}
					select {
					case merged <- msg:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	go func() {
		ackWG.Wait()
		acks <- errors.Join(ackErrs...)
		close(acks)
	}()
	go func() {
		wg.Wait()
		close(merged)
	}()

	stream := NewStream(ctx, merged, acks, func(msg *Message) StreamData[jsondoc.JSONDoc] {
		return StreamData[jsondoc.JSONDoc]{
			Action:  msg.Payload.RealtimeAction,
			Channel: msg.Channel,
			Data:    jsondoc.New(msg.Payload.Data.Bytes()),
		}
	}, func() {
		for _, pattern := range patterns {
			c.Unsubscribe(pattern)
		}
	})
	return op.OK(stream)
}
