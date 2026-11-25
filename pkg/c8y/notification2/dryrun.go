package notification2

import (
	"context"
	"errors"
	"log/slog"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/wsdial"
)

// IsDryRun reports whether err stands for a connection which was not made
// because of dry run (see Notification2Client.ConnectContext).
func IsDryRun(err error) bool {
	return errors.Is(err, wsdial.ErrDryRun)
}

// Keys of op.Result.Meta set by DryRunStream.
const (
	// DryRunMetaConsumer is the consumer name of the subscription.
	DryRunMetaConsumer = "consumer"
	// DryRunMetaURL is the websocket URL which would have been opened, with the
	// token redacted.
	DryRunMetaURL = "url"
)

// DryRunStream turns the error of ConnectContext into the result of a dry-run
// SubscribeStream. When err is a dry-run error it closes client and returns
// ok=true and a result with Status Skipped, no error, the upgrade request in
// Request, the consumer and the redacted URL in Meta, and a stream which ends
// immediately without error. For any other err it returns ok=false.
func DryRunStream[T any](ctx context.Context, client *Notification2Client, err error) (result op.Result[*Stream[T]], ok bool) {
	if !IsDryRun(err) {
		return result, false
	}
	_ = client.Close()

	messages := make(chan *Message)
	close(messages)
	stream := NewStream(ctx, client, messages, func(*Message) T {
		var zero T
		return zero
	}, nil)
	result = op.Skipped(stream, "dry run: websocket not connected")
	result.Request = wsdial.DryRunRequest(err)
	result.Meta[DryRunMetaConsumer] = client.Subscription.Consumer
	result.Meta[DryRunMetaURL] = client.URL(true)

	level := slog.LevelInfo
	if ctxhelpers.DryRunHandler(ctx) != nil {
		level = slog.LevelDebug
	}
	slog.Log(ctx, level, "DRY RUN subscribe", "consumer", client.Subscription.Consumer, "url", client.URL(true))
	return result, true
}
