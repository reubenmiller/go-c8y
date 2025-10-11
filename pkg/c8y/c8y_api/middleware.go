package c8y_api

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"resty.dev/v3"
)

func MiddlewareAddUserAgent(application string, userAgent string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		r.SetHeader("User-Agent", userAgent)
		r.SetHeader("X-APPLICATION", application)
		return nil
	}
}

func MiddlewareAddHost(domain string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		if domain != "" && r.RawRequest != nil && domain != r.RawRequest.URL.Host {
			// setting the Host header actually does nothing however
			// it makes the setting visible when logging
			r.Header.Set("Host", domain)
			r.RawRequest.Host = domain
		}
		return nil
	}
}

func MiddlewareAddCookies(cookies []*http.Cookie) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		for _, cookie := range cookies {
			if cookie.Name == "XSRF-TOKEN" {
				r.SetHeader("X-"+cookie.Name, cookie.Value)
			} else {
				r.SetCookie(cookie)
			}
		}
		return nil
	}
}

var HeaderAuthorization = "Authorization"

func MiddlewareAuthorization(auth authentication.AuthOptions) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		for _, authType := range auth.GetAuthTypes() {
			switch authType {
			case authentication.AuthTypeBasic:
				user := authentication.JoinTenantUser(auth.Tenant, auth.Username)
				if user != "" && auth.Password != "" {
					r.SetBasicAuth(user, auth.Password)
					return nil
				}
			case authentication.AuthTypeBearer:
				if auth.Token != "" {
					r.Header.Set(HeaderAuthorization, fmt.Sprintf("Bearer %s", auth.Token))
					slog.Info("Auth", "value", r.Header.Get(HeaderAuthorization))

					return nil
				}
			case authentication.AuthTypeUnset:
				if auth.Token != "" {
					r.Header.Set(HeaderAuthorization, fmt.Sprintf("Bearer %s", auth.Token))
					slog.Info("Auth", "value", r.Header.Get(HeaderAuthorization))
					return nil
				}
				user := authentication.JoinTenantUser(auth.Tenant, auth.Username)
				if user != "" && auth.Password != "" {
					r.SetBasicAuth(user, auth.Password)
					return nil
				}
			case authentication.AuthTypeNone:
				return nil
			}
		}

		return nil
	}
}

func SetAuth(c *resty.Client, auth authentication.AuthOptions) {
	if auth.CertificateKey != "" && auth.Certificate != "" {
		if _, err := os.Stat(auth.CertificateKey); err == nil {
			c.SetCertificateFromString(auth.Certificate, auth.CertificateKey)
		} else {
			c.SetCertificateFromFile(auth.Certificate, auth.CertificateKey)
		}
		// TODO: The certificate should only be used to exchange for a token

	} else if auth.Token != "" {
		c.SetAuthToken(auth.Token)
	} else if auth.Username != "" && auth.Password != "" {
		c.SetBasicAuth(authentication.JoinTenantUser(auth.Tenant, auth.Username), auth.Password)
	}
}
