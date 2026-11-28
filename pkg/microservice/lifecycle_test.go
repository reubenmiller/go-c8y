package microservice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// lifecyclePlatform serves the subscriptions and settings endpoints.
type lifecyclePlatform struct {
	mu            sync.Mutex
	users         []model.ServiceUser
	subscriptions atomic.Int32
	settingsCalls atomic.Int32
	headers       []http.Header
	server        *httptest.Server
}

func newLifecyclePlatform(t *testing.T, users ...model.ServiceUser) *lifecyclePlatform {
	t.Helper()
	lp := &lifecyclePlatform{users: users}
	lp.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lp.mu.Lock()
		lp.headers = append(lp.headers, r.Header.Clone())
		lp.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/application/currentApplication/subscriptions":
			lp.subscriptions.Add(1)
			lp.mu.Lock()
			out := make([]map[string]string, 0, len(lp.users))
			for _, u := range lp.users {
				out = append(out, map[string]string{"tenant": u.Tenant, "name": u.Username, "password": u.Password})
			}
			lp.mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"users": out})
		case "/application/currentApplication/settings":
			lp.settingsCalls.Add(1)
			user, _, _ := r.BasicAuth()
			interval := "60"
			if strings.HasPrefix(user, "t1/") {
				interval = "30" // tenant override
			}
			_, _ = fmt.Fprintf(w, `{"interval":%q}`, interval)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(lp.server.Close)
	return lp
}

func (lp *lifecyclePlatform) setUsers(users ...model.ServiceUser) {
	lp.mu.Lock()
	lp.users = users
	lp.mu.Unlock()
}

func newLifecycleMicroservice(lp *lifecyclePlatform) *Microservice {
	ms, err := NewMicroservice(lp.server.URL, func(o *api.ClientOptions) {
		o.SkipTenantLookup = true
		o.Auth = authentication.AuthOptions{Tenant: "management", Username: "bootstrap", Password: "x",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic}}
	})
	if err != nil {
		panic(err)
	}
	return ms
}

var (
	userT1 = model.ServiceUser{Tenant: "t1", Username: "service_app", Password: "p1"}
	userT2 = model.ServiceUser{Tenant: "t2", Username: "service_app", Password: "p2"}
)

// recorder collects watch callbacks.
type recorder struct {
	mu     sync.Mutex
	events []string
	errs   []error
}

func (r *recorder) add(e string) {
	r.mu.Lock()
	r.events = append(r.events, e)
	r.mu.Unlock()
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

func (r *recorder) options(interval time.Duration) WatchOptions {
	return WatchOptions{
		Interval: interval,
		OnTenantAdded: func(ctx context.Context, u model.ServiceUser) error {
			r.add("added:" + u.Tenant)
			return nil
		},
		OnTenantRemoved: func(ctx context.Context, tenant string) error {
			r.add("removed:" + tenant)
			return nil
		},
		OnCredentialsChanged: func(ctx context.Context, u model.ServiceUser) error {
			r.add("changed:" + u.Tenant)
			return nil
		},
		OnError: func(err error) {
			r.mu.Lock()
			r.errs = append(r.errs, err)
			r.mu.Unlock()
		},
	}
}

func Test_WatchSubscriptions_Lifecycle(t *testing.T) {
	lp := newLifecyclePlatform(t, userT2, userT1)
	ms := newLifecycleMicroservice(lp)
	rec := &recorder{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- ms.WatchSubscriptions(ctx, rec.options(20*time.Millisecond)) }()

	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 2 }, 2*time.Second, 5*time.Millisecond)
	assert.Equal(t, []string{"added:t1", "added:t2"}, rec.snapshot()[:2], "startup reports all tenants, sorted")

	// t2 unsubscribes; t1's service user is recreated with a new password.
	t1New := userT1
	t1New.Password = "p1-new"
	lp.setUsers(t1New)
	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 4 }, 2*time.Second, 5*time.Millisecond)
	assert.ElementsMatch(t, []string{"changed:t1", "removed:t2"}, rec.snapshot()[2:4])

	got, ok := ms.GetServiceUser("t1")
	require.True(t, ok)
	assert.Equal(t, "p1-new", got.Password)

	// The unsubscribed tenant is forgotten. Whether a miss refreshes is checked
	// by Test_WatchSubscriptions_NoRefreshOnMiss: the watcher polls every 20ms
	// here, so counting the requests around the lookup would be racy.
	_, ok = ms.GetServiceUser("t2")
	assert.False(t, ok)

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("WatchSubscriptions did not return after cancellation")
	}
	after := lp.subscriptions.Load()
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, after, lp.subscriptions.Load(), "no requests after cancellation")
}

func Test_WatchSubscriptions_NoRefreshOnMiss(t *testing.T) {
	lp := newLifecyclePlatform(t, userT1)
	ms := newLifecycleMicroservice(lp)
	rec := &recorder{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	// The interval is so long that only the initial poll happens, so the
	// request count can only change if the lookup itself refreshes
	go func() { done <- ms.WatchSubscriptions(ctx, rec.options(time.Hour)) }()

	require.Eventually(t, func() bool { return len(rec.snapshot()) >= 1 }, 2*time.Second, 5*time.Millisecond)
	before := lp.subscriptions.Load()
	_, ok := ms.GetServiceUser("t2")
	assert.False(t, ok)
	assert.Equal(t, before, lp.subscriptions.Load(), "no refresh on a miss while watching")

	cancel()
	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("WatchSubscriptions did not return after cancellation")
	}
}

func Test_WatchSubscriptions_CallbackErrorDoesNotStop(t *testing.T) {
	lp := newLifecyclePlatform(t, userT1)
	ms := newLifecycleMicroservice(lp)
	rec := &recorder{}
	opts := rec.options(20 * time.Millisecond)
	opts.OnTenantAdded = func(ctx context.Context, u model.ServiceUser) error {
		return errors.New("setup failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = ms.WatchSubscriptions(ctx, opts) }()

	require.Eventually(t, func() bool {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		return len(rec.errs) == 1
	}, 2*time.Second, 5*time.Millisecond)
	_, ok := ms.GetServiceUser("t1")
	assert.True(t, ok, "the cache is updated even when a callback fails")
	polls := lp.subscriptions.Load()
	require.Eventually(t, func() bool { return lp.subscriptions.Load() > polls }, 2*time.Second, 5*time.Millisecond, "watching continues")
}

func Test_ForEachTenant_DefaultStopsAtFirstError(t *testing.T) {
	ms := &Microservice{}
	ms.SetServiceUsers([]model.ServiceUser{userT1, userT2, {Tenant: "t3"}})
	var calls atomic.Int32
	err := ms.ForEachTenant(context.Background(), func(ctx context.Context, u model.ServiceUser) error {
		calls.Add(1)
		if u.Tenant == "t2" {
			return errors.New("boom")
		}
		return nil
	})
	require.EqualError(t, err, "boom")
	assert.EqualValues(t, 2, calls.Load(), "the third tenant is not processed")
}

func Test_ForEachTenant_ConcurrentWithErrorTolerance(t *testing.T) {
	ms := &Microservice{}
	ms.SetServiceUsers([]model.ServiceUser{userT1, userT2, {Tenant: "t3"}})
	var calls atomic.Int32
	_ = ms.ForEachTenant(context.Background(), func(ctx context.Context, u model.ServiceUser) error {
		calls.Add(1)
		auth, ok := authentication.AuthFromContext(ctx)
		require.True(t, ok)
		assert.Equal(t, u.Tenant, auth.Tenant)
		if u.Tenant == "t2" {
			return errors.New("boom")
		}
		return nil
	}, pipeline.Options{Workers: 3})
	assert.EqualValues(t, 3, calls.Load())
}

func Test_Settings_PerTenantAndCached(t *testing.T) {
	lp := newLifecyclePlatform(t, userT1, userT2)
	ms := newLifecycleMicroservice(lp)
	require.NoError(t, ms.RefreshServiceUsers(context.Background()))

	s1, err := ms.Settings(ms.WithServiceUser(context.Background(), "t1"))
	require.NoError(t, err)
	assert.Equal(t, "30", s1.Get("interval").String(), "t1 overrides the editable setting")
	s2, err := ms.Settings(ms.WithServiceUser(context.Background(), "t2"))
	require.NoError(t, err)
	assert.Equal(t, "60", s2.Get("interval").String())

	_, err = ms.Settings(ms.WithServiceUser(context.Background(), "t1"))
	require.NoError(t, err)
	assert.EqualValues(t, 2, lp.settingsCalls.Load(), "the second t1 read is cached")
}

func Test_Bootstrap_PerTenantIsolation(t *testing.T) {
	lp := newLifecyclePlatform(t, userT1)
	t.Setenv("C8Y_MICROSERVICE_ISOLATION", "PER_TENANT")
	t.Setenv("C8Y_TENANT", "t5")
	t.Setenv("C8Y_USER", "service_app")
	t.Setenv("C8Y_PASSWORD", "p5")
	ms := newLifecycleMicroservice(lp)

	require.NoError(t, ms.Bootstrap(context.Background()))
	u, ok := ms.GetServiceUser("")
	require.True(t, ok)
	assert.Equal(t, "t5", u.Tenant)
	assert.EqualValues(t, 0, lp.subscriptions.Load(), "no subscriptions request in PER_TENANT mode")
}

func Test_ApplicationKeyHeader(t *testing.T) {
	lp := newLifecyclePlatform(t, userT1)
	t.Setenv("APPLICATION_KEY", "abc")
	ms := newLifecycleMicroservice(lp)

	require.NoError(t, ms.RefreshServiceUsers(context.Background()))
	lp.mu.Lock()
	defer lp.mu.Unlock()
	require.NotEmpty(t, lp.headers)
	assert.Equal(t, "abc", lp.headers[len(lp.headers)-1].Get(HeaderApplicationKey))
}
