package contexthelpers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRequestHooksCompose(t *testing.T) {
	var calls []string
	ctx := WithRequestHook(context.Background(), func(*http.Request) error { calls = append(calls, "first"); return nil })
	ctx = WithRequestHook(ctx, func(*http.Request) error { calls = append(calls, "second"); return nil })
	ctx = WithRequestHook(ctx, nil)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err := RequestHook(ctx)(req); err != nil {
		t.Fatal(err)
	}
	if strings.Join(calls, ",") != "first,second" {
		t.Fatalf("hooks ran as %v", calls)
	}

	failing := WithRequestHook(context.Background(), func(*http.Request) error { return errors.New("stop") })
	failing = WithRequestHook(failing, func(*http.Request) error { t.Fatal("must not run after an error"); return nil })
	if err := RequestHook(failing)(req); err == nil {
		t.Fatal("expected the first hook's error")
	}
}

type tagged struct {
	io.Reader
	tag string
}

func (tagged) Close() error { return nil }

func TestResponseBodyWrappersCompose(t *testing.T) {
	ctx := WithResponseBodyWrapper(context.Background(), func(r *http.Response) io.ReadCloser { return tagged{r.Body, "inner"} })
	ctx = WithResponseBodyWrapper(ctx, func(r *http.Response) io.ReadCloser {
		inner, ok := r.Body.(tagged)
		if !ok || inner.tag != "inner" {
			t.Fatalf("second wrapper got %T", r.Body)
		}
		return tagged{r.Body, "outer"}
	})
	resp := &http.Response{Body: io.NopCloser(strings.NewReader("x"))}
	body := ResponseBodyWrapper(ctx)(resp)
	if b, ok := body.(tagged); !ok || b.tag != "outer" {
		t.Fatalf("got %T", body)
	}
}
