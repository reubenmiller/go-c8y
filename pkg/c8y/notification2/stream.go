package notification2

import (
	"context"
	"iter"
)

// StreamData wraps a realtime message with its action type and typed data payload.
// The Action field indicates what triggered the realtime notification (e.g., "CREATE", "UPDATE", "DELETE").
type StreamData[T any] struct {
	Identifier string

	Description string

	Action string

	// Data contains the typed payload of the realtime message
	Data T
}

// Stream provides an iterator pattern for realtime subscriptions with automatic type conversion.
// It allows consumers to iterate over typed messages from a realtime subscription.
//
// IMPORTANT: Resource Management
// Always call Close() when done with the stream, typically via defer.
// While the subscription will eventually be cleaned up when the context times out,
// defer stream.Close() ensures immediate cleanup when your code exits early.
//
// REQUIRED pattern (consistent with Go best practices):
//
// Using range with Items() (with error handling):
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	streamResult := client.Events.SubscribeStream(ctx, deviceID)
//	if streamResult.Err != nil {
//	    return streamResult.Err
//	}
//	stream := streamResult.Data
//	defer stream.Close() // Required: ensures cleanup
//
//	for event, err := range stream.Items() {
//	    if err != nil {
//	        return err
//	    }
//	    if event.Type() == "targetType" {
//	        break // Safe: defer stream.Close() will cleanup
//	    }
//	}
//
// Using range with Seq() (errors stop iteration):
//
//	for event := range stream.Seq() {
//	    log.Printf("Event: %s", event.Type())
//	}
//	if err := stream.Err(); err != nil {
//	    return err
//	}
//
// Using Next() (traditional iterator):
//
//	for stream.Next() {
//	    event := stream.Value()
//	    log.Printf("Event: %s", event.Type())
//	}
//	if err := stream.Err(); err != nil {
//	    return err
//	}
//
// Cleanup happens via:
// 1. defer stream.Close() - immediate cleanup (recommended)
// 2. defer cancel() - cleanup when context cancelled
// 3. Context timeout - cleanup as safety fallback
type Stream[T any] struct {
	client        *Notification2Client
	messages      <-chan *Message
	ctx           context.Context
	cancel        context.CancelFunc // Cancel function for the stream context
	cleanup       func()             // Optional cleanup function (e.g., unsubscribe)
	current       T
	err           error
	transform     func(*Message) T
	cleanupCalled bool
}

// NewStream creates a new typed stream from a realtime subscription.
// The transform function converts each Message to the desired type T.
// The cleanup function is called when Close() is invoked for explicit resource cleanup.
//
// If the provided context has a cancel function, you can optionally call stream.Close()
// to explicitly trigger cleanup before the context timeout.
func NewStream[T any](
	ctx context.Context,
	client *Notification2Client,
	messages <-chan *Message,
	transform func(*Message) T,
	cleanup func(),
) *Stream[T] {
	// Wrap the context so we can provide an explicit Close() method
	ctx, cancel := context.WithCancel(ctx)
	return &Stream[T]{
		client:        client,
		messages:      messages,
		ctx:           ctx,
		cancel:        cancel,
		cleanup:       cleanup,
		transform:     transform,
		cleanupCalled: false,
	}
}

func (s *Stream[T]) AcknowledgeMessage(messageIdentifier string) error {
	return s.client.SendMessageAck(messageIdentifier)
}

// Next advances the stream to the next message.
// Returns true if a message was received, false if the stream has ended or an error occurred.
// Check Err() to distinguish between normal completion and error conditions.
func (s *Stream[T]) Next() bool {

	// Now wait for messages or context cancellation
	select {
	case msg, ok := <-s.messages:
		if !ok {
			// Channel closed normally
			return false
		}
		s.current = s.transform(msg)
		return true

	case <-s.ctx.Done():
		// Context cancelled or timed out
		s.err = s.ctx.Err()
		return false
	}
}

// Value returns the current message after a successful call to Next().
// Should only be called after Next() returns true.
func (s *Stream[T]) Value() T {
	return s.current
}

// Err returns any error that occurred during iteration.
// Should be called after Next() returns false to check if iteration ended due to an error.
func (s *Stream[T]) Err() error {
	return s.err
}

// Close cancels the subscription and cleans up resources.
// Always call Close() when done with the stream, typically via defer.
// This is required for proper resource cleanup, similar to closing files or database rows.
//
// It's safe to call Close() multiple times.
// After calling Close(), Next() and iteration will return false/stop.
//
// Example:
//
//	streamResult := client.Events.SubscribeStream(ctx, deviceID)
//	if streamResult.Err != nil {
//	    return streamResult.Err
//	}
//	stream := streamResult.Data
//	defer stream.Close() // Required
//
//	for event, err := range stream.Items() {
//	    if shouldStop {
//	        break // Close() called via defer
//	    }
//	}
func (s *Stream[T]) Close() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.cleanup != nil && !s.cleanupCalled {
		s.cleanupCalled = true
		s.cleanup()
	}
	return nil
}

// Items returns an iterator that yields messages with errors.
// This follows the same pattern as pagination.Iterator for consistency.
// Use this when you need to handle errors during iteration.
//
// Example:
//
//	streamResult := client.Events.SubscribeStream(ctx, deviceID)
//	if streamResult.Err != nil {
//	    return streamResult.Err
//	}
//	stream := streamResult.Data
//	defer stream.Close()
//
//	for event, err := range stream.Items() {
//	    if err != nil {
//	        return err
//	    }
//	    log.Printf("Event: %s", event.Type())
//	}
func (s *Stream[T]) Items() iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		// Now iterate over messages or context cancellation
		for {
			select {
			case msg, ok := <-s.messages:
				if !ok {
					// Channel closed normally
					return
				}
				item := s.transform(msg)
				if !yield(item, nil) {
					return
				}

			case <-s.ctx.Done():
				// Context cancelled or timed out
				s.err = s.ctx.Err()
				var zero T
				yield(zero, s.err)
				return
			}
		}
	}
}

// Seq returns an iterator that yields only successful items, discarding errors.
// This follows the same pattern as pagination.Iterator.Seq() for consistency.
// Use this when you want simpler iteration without explicit error handling.
// Note: Errors will cause iteration to stop silently.
//
// Example:
//
//	streamResult := client.Events.SubscribeStream(ctx, deviceID)
//	if streamResult.Err != nil {
//	    return streamResult.Err
//	}
//	stream := streamResult.Data
//	defer stream.Close()
//
//	for event := range stream.Seq() {
//	    log.Printf("Event: %s", event.Type())
//	}
//	// Check for errors after iteration
//	if err := stream.Err(); err != nil {
//	    return err
//	}
func (s *Stream[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for item, err := range s.Items() {
			if err != nil {
				// Stop on error
				return
			}
			if !yield(item) {
				return
			}
		}
	}
}
