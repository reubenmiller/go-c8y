package contexthelpers

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

// ErrRequestNotApproved is returned for a request whose approver declined it.
var ErrRequestNotApproved = errors.New("request not approved")

// RequestApprover decides whether a request that changes data may be sent. It
// receives the final request (resolved ids, headers and query parameters
// applied) before any byte is sent. Returning false declines the request;
// returning an error declines it with that error.
type RequestApprover func(req *http.Request) (approve bool, err error)

type requestApproverKey struct{}
type approvalScopeKey struct{}

// WithRequestApprover returns a context whose requests that change data (any
// method other than GET, HEAD and OPTIONS) are passed to fn before they are
// sent. A nil fn removes the approver.
func WithRequestApprover(ctx context.Context, fn RequestApprover) context.Context {
	return context.WithValue(ctx, requestApproverKey{}, fn)
}

// GetRequestApprover returns the approver set with WithRequestApprover, or nil.
func GetRequestApprover(ctx context.Context) RequestApprover {
	fn, _ := ctx.Value(requestApproverKey{}).(RequestApprover)
	return fn
}

// approvalScope records the first approval decision made under it, so an
// operation that sends several requests (upload then update, get-or-create) is
// approved once.
type approvalScope struct {
	mu      sync.Mutex
	decided bool
	err     error
}

// WithApprovalScope returns a context that starts a new approval scope. The
// approver is asked once per scope: later requests in the same scope reuse the
// first decision.
func WithApprovalScope(ctx context.Context) context.Context {
	return context.WithValue(ctx, approvalScopeKey{}, &approvalScope{})
}

// ApproveRequest asks the context's approver whether req may be sent. It
// returns nil when no approver is set, for reads, or when the request is
// approved, and ErrRequestNotApproved (or the approver's error) otherwise.
func ApproveRequest(ctx context.Context, req *http.Request) error {
	fn := GetRequestApprover(ctx)
	if fn == nil || req == nil {
		return nil
	}
	switch req.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return nil
	}
	scope, _ := ctx.Value(approvalScopeKey{}).(*approvalScope)
	if scope == nil {
		return decide(fn, req)
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	if !scope.decided {
		scope.err = decide(fn, req)
		scope.decided = true
	}
	return scope.err
}

func decide(fn RequestApprover, req *http.Request) error {
	ok, err := fn(req)
	if err != nil {
		return err
	}
	if !ok {
		return ErrRequestNotApproved
	}
	return nil
}
