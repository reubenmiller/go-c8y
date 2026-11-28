package notification2

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/notification2"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	MinTokenMinutes int64 = 1
)

var apiSubscriptions = "/notification2/subscriptions"
var apiSubscription = "/notification2/subscriptions/{id}"
var apiToken = "/notification2/token"
var apiUnsubscribe = "/notification2/unsubscribe"

var paramID = "id"

const resultProperty = "subscriptions"

// NewService creates a new notification2 service with device resolution
// capabilities, so the subscription source (a managed object / device) can be
// given as an id, name, external id or query and resolved against the inventory.
func NewService(common *httpcore.Service, moService *managedobjects.Service) *Service {
	return &Service{
		svc:            *common,
		DeviceResolver: managedobjects.NewDeviceResolver(moService),
	}
}

type Service struct {
	svc            httpcore.Service
	DeviceResolver *managedobjects.DeviceResolver
}

type TokenOptions struct {
	ExpiresInMinutes  int64  `json:"expiresInMinutes,omitempty"`
	Subscriber        string `json:"subscriber,omitempty"`
	DefaultSubscriber string `json:"-"`
	Subscription      string `json:"subscription,omitempty"`
	Shared            bool   `json:"shared,omitempty"`
}

func (nt *TokenOptions) GetDefaultSubscriber() string {
	if nt.DefaultSubscriber != "" {
		return nt.DefaultSubscriber
	}
	return "goc8y"
}

// ListOptions is generated from the OpenAPI spec — see zz_generated_options.go

type DeleteBySourceOptions struct {
	Context string `url:"context,omitempty"`
	Source  string `url:"source,omitempty"`
}

type CreateOptions struct {
	Context            string             `json:"context,omitempty"`
	FragmentsToCopy    []string           `json:"fragmentsToCopy,omitempty"`
	Source             any                `json:"source,omitempty"`
	Subscription       string             `json:"subscription,omitempty"`
	SubscriptionFilter SubscriptionFilter `json:"subscriptionFilter,omitempty"`
}

type SubscriptionFilter struct {
	Apis       []string `json:"apis,omitempty"`
	TypeFilter string   `json:"typeFilter,omitempty"`
}

type UnsubscribeResponse struct {
	Result string `json:"result,omitempty"`
}

type TokenClaim struct {
	Subscriber string `json:"sub,omitempty"`
	Topic      string `json:"topic,omitempty"`
	Shared     string `json:"shared,omitempty"`
	jwt.RegisteredClaims
}

func (c *TokenClaim) IsShared() bool {
	return strings.EqualFold(c.Shared, "true")
}

func (c *TokenClaim) Tenant() string {
	index := strings.Index(c.Topic, "/")
	if index == -1 {
		return ""
	}
	return c.Topic[0:index]
}

func (c *TokenClaim) Subscription() string {
	index := strings.LastIndex(c.Topic, "/")
	if index == -1 {
		return ""
	}
	return c.Topic[index+1:]
}

func (c *TokenClaim) HasExpired() bool {
	var v = jwt.NewValidator(jwt.WithLeeway(5 * time.Second))
	err := v.Validate(c)
	return err != nil
}

type ClientOptions struct {
	Token             string
	Consumer          string
	Options           TokenOptions
	ConnectionOptions notification2.ConnectionOptions
}

type SubscriptionIterator = pagination.Iterator[jsonmodels.Notification2Subscription]

func (s *Service) Get(ctx context.Context, id string) op.Result[jsonmodels.Notification2Subscription] {
	return httpcore.Execute(ctx, s.getB(id), jsonmodels.NewNotification2Subscription)
}

func (s *Service) getB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, id).
		SetURL(apiSubscription)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// resolveSource turns a subscription source reference (id, "name:", "ext:" or
// "query:") into a managed object id, hitting the inventory only for non-id
// refs. Resolution runs in a context that ignores dry-run so the resolved id is
// reflected even when the caller is in --dry mode.
func (s *Service) resolveSource(ctx context.Context, source string) (string, error) {
	if source == "" || s.DeviceResolver == nil {
		return source, nil
	}
	return s.DeviceResolver.ResolveID(ctxhelpers.ResolutionContext(ctx), managedobjects.DeviceRef(source), nil)
}

func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Notification2Subscription] {
	resolved, err := s.resolveSource(ctx, opt.Source)
	if err != nil {
		return op.Failed[jsonmodels.Notification2Subscription](err, true)
	}
	opt.Source = resolved
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewNotification2Subscription)
}

func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SubscriptionIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Notification2Subscription] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewNotification2Subscription,
	)
}

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiSubscriptions)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Notification2Subscription] {
	return httpcore.Execute(ctx, s.createB(opt), jsonmodels.NewNotification2Subscription).IgnoreConflict()
}

// CreateRaw creates a subscription from a raw body (map / json.RawMessage),
// preserving the CLI's --data/--template body fidelity. A device reference at
// "source.id" that is not an id (e.g. "dev01", "name:dev01") is resolved to
// the device id first; this is the equivalent of Create for an untyped body.
func (s *Service) CreateRaw(ctx context.Context, body any) op.Result[jsonmodels.Notification2Subscription] {
	body, err := s.DeviceResolver.ResolveBodyRefs(ctx, body, "source.id")
	if err != nil {
		return op.Failed[jsonmodels.Notification2Subscription](err, false)
	}
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewNotification2Subscription).IgnoreConflict()
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiSubscriptions)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) Delete(ctx context.Context, id string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(id)).IgnoreNotFound()
}

func (s *Service) deleteB(id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramID, id).
		SetURL(apiSubscription)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) DeleteBySource(ctx context.Context, opt DeleteBySourceOptions) op.Result[core.NoContent] {
	resolved, err := s.resolveSource(ctx, opt.Source)
	if err != nil {
		return op.Failed[core.NoContent](err, true)
	}
	opt.Source = resolved
	return httpcore.ExecuteNoContent(ctx, s.deleteBySourceB(opt))
}

func (s *Service) deleteBySourceB(opt DeleteBySourceOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiSubscriptions)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) CreateToken(ctx context.Context, opt TokenOptions) op.Result[jsonmodels.Notification2Token] {
	if opt.Subscriber == "" {
		opt.Subscriber = opt.GetDefaultSubscriber()
	}
	return httpcore.Execute(ctx, s.createTokenB(opt), jsonmodels.NewNotification2Token)
}

// CreateTokenRaw creates a token from a raw body (map / json.RawMessage),
// preserving the CLI's --data/--template body fidelity and any extra fields the
// typed TokenOptions does not model (e.g. type/signed/nonPersistent). The
// default subscriber must already be applied by the caller.
func (s *Service) CreateTokenRaw(ctx context.Context, body any) op.Result[jsonmodels.Notification2Token] {
	return httpcore.Execute(ctx, s.createTokenB(body), jsonmodels.NewNotification2Token)
}

func (s *Service) createTokenB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiToken)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) UnsubscribeSubscriber(ctx context.Context, token string) op.Result[UnsubscribeResponse] {
	return httpcore.Execute(ctx, s.unsubscribeSubscriberB(token), func(data []byte) UnsubscribeResponse {
		var result UnsubscribeResponse
		json.Unmarshal(data, &result)
		return result
	})
}

func (s *Service) unsubscribeSubscriberB(token string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParam("token", token).
		SetURL(apiUnsubscribe)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func (s *Service) ParseToken(tokenString string) (*TokenClaim, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token. expected 3 fields")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, err
	}

	claim := &TokenClaim{}
	err = json.Unmarshal(raw, claim)
	return claim, err
}

func (s *Service) RenewToken(ctx context.Context, opt ClientOptions) (string, error) {
	isValid := true
	claimMatch := true

	subscription := opt.Options.Subscription
	subscriber := s.NormalizedConsumer(opt.Options.Subscriber)
	expiresInMinutes := opt.Options.ExpiresInMinutes
	shared := opt.Options.Shared

	if opt.Token != "" {
		claims := TokenClaim{}
		parser := jwt.NewParser()
		token, _, err := parser.ParseUnverified(opt.Token, &claims)

		if err != nil {
			slog.Info("Token is invalid", "err", err)
			isValid = false
		} else if err := jwt.NewValidator(jwt.WithLeeway(5 * time.Second)).Validate(token.Claims); err != nil {
			slog.Info("Token is invalid", "err", err)
			isValid = false
		}

		alg := ""
		if token != nil && token.Method != nil {
			// token is nil when the token could not be parsed at all
			alg = token.Method.Alg()
		}
		slog.Info("Existing token", "alg", alg, "valid", isValid, "expired", claims.HasExpired(), "issuedAt", claims.IssuedAt, "expiresAt", claims.ExpiresAt, "subscription", claims.Subscription(), "subscriber", claims.Subscriber, "shared", claims.IsShared(), "tenant", claims.Tenant())

		if opt.Options.Subscription != "" {
			if claims.Subscription() != opt.Options.Subscription {
				claimMatch = false
			}
		} else {
			subscription = claims.Subscription()
		}

		if opt.Options.Subscriber != "" {
			if claims.Subscriber != opt.Options.Subscriber {
				claimMatch = false
			}
		} else {
			subscriber = claims.Subscriber
		}

		shared = claims.IsShared()

		if claimMatch && expiresInMinutes == 0 {
			if claims.ExpiresAt != nil && claims.IssuedAt != nil {
				expiresInMinutes = claims.ExpiresAt.Unix() - claims.IssuedAt.Unix()
			}
		}

		if isValid && claimMatch {
			slog.Info("Using existing valid token")
			return opt.Token, nil
		}
		slog.Info("Token does not match claim. Invalid information will be ignored in the token")
	}

	if expiresInMinutes < MinTokenMinutes {
		expiresInMinutes = MinTokenMinutes
	}

	slog.Info("Creating new notification2 token")
	result := s.CreateToken(ctx, TokenOptions{
		ExpiresInMinutes:  expiresInMinutes,
		Subscription:      subscription,
		Subscriber:        subscriber,
		DefaultSubscriber: opt.Options.DefaultSubscriber,
		Shared:            shared,
	})
	if result.Err != nil {
		return "", result.Err
	}
	return result.Data.Token(), nil
}

// CreateClient renews or creates the notification2 token and returns a client
// for it. The client connects through the Cumulocity client (TLS, proxy,
// request middleware and extra headers, dry run), see
// [notification2.Notification2Client.SetConnector]. ctx supplies the context
// values of the token renewal done on every reconnect, but not its cancellation.
func (s *Service) CreateClient(ctx context.Context, opt ClientOptions) (*notification2.Notification2Client, error) {
	token, err := s.RenewToken(ctx, opt)
	if err != nil {
		return nil, err
	}

	renewCtx := context.WithoutCancel(ctx)
	client, err := notification2.NewNotification2ClientE(s.svc.Client.BaseURL(), nil, notification2.Subscription{
		TokenRenewal: func(v string) (string, error) {
			return s.RenewToken(renewCtx, ClientOptions{
				Token: v,
			})
		},
		Consumer: s.NormalizedConsumer(opt.Consumer),
		Token:    token,
	}, opt.ConnectionOptions)
	if err != nil {
		return nil, err
	}
	if s.svc.Connector != nil {
		client.SetConnector(s.svc.Connector)
	}
	return client, nil
}

func (s *Service) NormalizedConsumer(v string) string {
	result := make([]rune, 0, len(v))
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result = append(result, r)
		}
	}
	return string(result)
}

type MessageStream = notification2.Stream[notification2.StreamData[jsondoc.JSONDoc]]

func (s *Service) SubscribeStream(ctx context.Context, opt ClientOptions) op.Result[*MessageStream] {
	client, err := s.CreateClient(ctx, opt)
	if err != nil {
		return op.Failed[*MessageStream](err, false)
	}

	if err := client.ConnectContext(ctx); err != nil {
		if dry, ok := notification2.DryRunStream[notification2.StreamData[jsondoc.JSONDoc]](ctx, client, err); ok {
			return dry
		}
		_ = client.Close()
		return op.Failed[*MessageStream](err, false)
	}

	// The client delivers messages to ch; forward them to the stream until
	// the stream is closed or ctx is done.
	streamCtx, stopForwarding := context.WithCancel(ctx)
	messages := make(chan *notification2.Message, 10)
	ch := make(chan notification2.Message)
	client.Register("*", ch)
	go func() {
		defer close(messages)
		for {
			select {
			case <-streamCtx.Done():
				return
			case msg := <-ch:
				select {
				case messages <- &msg:
				case <-streamCtx.Done():
					return
				}
			}
		}
	}()

	stream := notification2.NewStream(streamCtx, client, messages, func(msg *notification2.Message) notification2.StreamData[jsondoc.JSONDoc] {
		return notification2.StreamData[jsondoc.JSONDoc]{
			Action:      msg.Action,
			Identifier:  msg.Identifier,
			Description: msg.Description,
			Data:        jsondoc.New(msg.Payload),
		}
	}, func() {
		stopForwarding()
		// Unsubscribe from the realtime channel, then stop the client's
		// goroutines.
		if err := client.Unsubscribe(); err != nil {
			slog.Warn("Failed to unsubscribe", "err", err)
		}
		_ = client.Close()
	})

	return op.OK(stream)
}
