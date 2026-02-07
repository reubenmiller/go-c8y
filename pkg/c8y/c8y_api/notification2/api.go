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
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/notification2"
	"resty.dev/v3"
)

var (
	MinTokenMinutes int64 = 1
)

var ApiSubscriptions = "/notification2/subscriptions"
var ApiSubscription = "/notification2/subscriptions/{id}"
var ApiToken = "/notification2/token"
var ApiUnsubscribe = "/notification2/unsubscribe"

var ParamId = "id"

const ResultProperty = "subscriptions"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

type Service struct {
	core.Service
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

type ListOptions struct {
	Context string `url:"context,omitempty"`
	Source  string `url:"source,omitempty"`
	pagination.PaginationOptions
}

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
	return core.Execute(ctx, s.getB(id), jsonmodels.NewNotification2Subscription)
}

func (s *Service) getB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, id).
		SetURL(ApiSubscription)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Notification2Subscription] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewNotification2Subscription)
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

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiSubscriptions)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

func (s *Service) Create(ctx context.Context, opt CreateOptions) op.Result[jsonmodels.Notification2Subscription] {
	return core.Execute(ctx, s.createB(opt), jsonmodels.NewNotification2Subscription)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiSubscriptions)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) Delete(ctx context.Context, id string) op.Result[jsonmodels.Notification2Subscription] {
	return core.Execute(ctx, s.deleteB(id), jsonmodels.NewNotification2Subscription)
}

func (s *Service) deleteB(id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, id).
		SetURL(ApiSubscription)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) DeleteBySource(ctx context.Context, opt DeleteBySourceOptions) op.Result[jsonmodels.Notification2Subscription] {
	return core.Execute(ctx, s.deleteBySourceB(opt), jsonmodels.NewNotification2Subscription)
}

func (s *Service) deleteBySourceB(opt DeleteBySourceOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiSubscriptions)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) CreateToken(ctx context.Context, opt TokenOptions) op.Result[jsonmodels.Notification2Token] {
	if opt.Subscriber == "" {
		opt.Subscriber = opt.GetDefaultSubscriber()
	}
	return core.Execute(ctx, s.createTokenB(opt), jsonmodels.NewNotification2Token)
}

func (s *Service) createTokenB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiToken)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) UnsubscribeSubscriber(ctx context.Context, token string) op.Result[UnsubscribeResponse] {
	return core.Execute(ctx, s.unsubscribeSubscriberB(token), func(data []byte) UnsubscribeResponse {
		var result UnsubscribeResponse
		json.Unmarshal(data, &result)
		return result
	})
}

func (s *Service) unsubscribeSubscriberB(token string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParam("token", token).
		SetURL(ApiUnsubscribe)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) ParseToken(tokenString string) (*TokenClaim, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token. expected 3 fields")
	}
	raw, err := base64.RawStdEncoding.DecodeString(parts[1])
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

		slog.Info("Existing token", "alg", token.Method.Alg(), "valid", isValid, "expired", claims.HasExpired(), "issuedAt", claims.IssuedAt, "expiresAt", claims.ExpiresAt, "subscription", claims.Subscription(), "subscriber", claims.Subscriber, "shared", claims.IsShared(), "tenant", claims.Tenant())

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

func (s *Service) CreateClient(ctx context.Context, opt ClientOptions) (*notification2.Notification2Client, error) {
	token, err := s.RenewToken(ctx, opt)
	if err != nil {
		return nil, err
	}

	client := notification2.NewNotification2Client(s.Client.BaseURL(), nil, notification2.Subscription{
		TokenRenewal: func(v string) (string, error) {
			return s.RenewToken(ctx, ClientOptions{
				Token: v,
			})
		},
		Consumer: s.NormalizedConsumer(opt.Consumer),
		Token:    token,
	}, opt.ConnectionOptions)
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
