package realtime

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/wsdial"
)

// ErrDryRun is matched (errors.Is) by the error ConnectContext returns instead
// of a connection when the connector is in dry run. It is wsdial.ErrDryRun.
var ErrDryRun = wsdial.ErrDryRun

// IsDryRun reports whether err stands for a connection which was not made
// because of dry run.
func IsDryRun(err error) bool {
	return errors.Is(err, wsdial.ErrDryRun)
}

// DryRunMetaChannels is the key of op.Result.Meta which lists the channels a
// dry run would have subscribed to.
const DryRunMetaChannels = "channels"

// DryRunStream turns the error of Connect/ConnectContext into the result of a
// dry-run SubscribeStream. When err is a dry-run error it returns ok=true and a
// result with:
//
//   - Status Skipped, no error, Meta["reason"] and Meta[DryRunMetaChannels]
//     (the channels that would have been subscribed to)
//   - Request: the websocket upgrade request which would have been sent
//   - Data: a stream which ends immediately, without error
//
// The channels are also logged (see logDryRunChannels). For any other err it
// returns ok=false and the caller handles err as before.
func DryRunStream[T any](ctx context.Context, err error, channels ...string) (result op.Result[*Stream[T]], ok bool) {
	if !IsDryRun(err) {
		return result, false
	}
	messages := make(chan *Message)
	close(messages)
	stream := NewStream(ctx, messages, nil, func(*Message) T {
		var zero T
		return zero
	}, nil)
	result = op.Skipped(stream, "dry run: websocket not connected")
	fillDryRun(ctx, &result.Request, result.Meta, err, channels)
	return result, true
}

// DryRunSubscribe is like DryRunStream for the Subscribe methods of the
// services (op.Result[chan error]). It also closes out, so a reader of out ends.
func DryRunSubscribe(ctx context.Context, err error, out chan<- *Message, channels ...string) (result op.Result[chan error], ok bool) {
	if !IsDryRun(err) {
		return result, false
	}
	if out != nil {
		close(out)
	}
	acks := make(chan error)
	close(acks)
	result = op.Skipped(acks, "dry run: websocket not connected")
	fillDryRun(ctx, &result.Request, result.Meta, err, channels)
	return result, true
}

func fillDryRun(ctx context.Context, req **http.Request, meta op.Meta, err error, channels []string) {
	*req = wsdial.DryRunRequest(err)
	meta[DryRunMetaChannels] = append([]string(nil), channels...)
	logDryRunChannels(ctx, channels)
}

// logDryRunChannels reports the channels a dry run would subscribe to. With a
// dry-run handler registered the caller owns the output, so it is only logged
// at debug level.
func logDryRunChannels(ctx context.Context, channels []string) {
	if len(channels) == 0 {
		return
	}
	level := slog.LevelInfo
	if ctxhelpers.DryRunHandler(ctx) != nil {
		level = slog.LevelDebug
	}
	slog.Log(ctx, level, "DRY RUN subscribe", "channels", channels)
}
