package deviceenrollment

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/reubenmiller/go-c8y/pkg/password"
	"go.mozilla.org/pkcs7"
	"resty.dev/v3"
)

var ApiEnroll = ".well-known/est/simpleenroll"
var ApiReEnroll = ".well-known/est/simplereenroll"
var ApiDeviceAccessToken = "/devicecontrol/deviceAccessToken"

// Service provides device enrollment functionality to enroll new devices and receive device certificates
type Service core.Service

func NewService(s *core.Service) *Service {
	return (*Service)(s)
}

// EnrollOptions options for enrolling a device
type EnrollOptions struct {
	ExternalID      string
	OneTimePassword string
	CSR             *x509.CertificateRequest
}

// Enroll a new device and receive a device certificate
func (s *Service) Enroll(ctx context.Context, opt EnrollOptions) op.Result[X509Certificate] {
	if opt.CSR == nil {
		return op.Failed[X509Certificate](fmt.Errorf("certificate signing request is required"), false)
	}

	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Transfer-Encoding", "base64").
		SetHeader("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(":"+opt.ExternalID+":"+opt.OneTimePassword)))).
		SetContentType("application/pkcs10").
		SetBody(base64.StdEncoding.EncodeToString(opt.CSR.Raw)).
		SetURL(ApiEnroll).
		Funcs(core.NoAuthorization())

	b := core.NewTryRequest(s.Client, req)
	return executeWithCertParse(ctx, b)
}

// ReEnrollOptions options for re-enrolling a device
type ReEnrollOptions struct {
	Token string
	CSR   *x509.CertificateRequest
}

// ReEnroll an already enrolled device using an existing device certificate
func (s *Service) ReEnroll(ctx context.Context, opt ReEnrollOptions) op.Result[X509Certificate] {
	if opt.CSR == nil {
		return op.Failed[X509Certificate](fmt.Errorf("certificate signing request is required"), false)
	}

	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Transfer-Encoding", "base64").
		SetContentType("application/pkcs10").
		SetBody(base64.StdEncoding.EncodeToString(opt.CSR.Raw)).
		SetURL(ApiReEnroll)

	if opt.Token != "" {
		req.SetHeader("Authorization", "Bearer "+opt.Token)
	}

	b := core.NewTryRequest(s.Client, req)
	return executeWithCertParse(ctx, b)
}

// RequestAccessTokenOptions options for requesting a device access token
type RequestAccessTokenOptions struct {
	ClientCert *tls.Certificate
	Headers    http.Header
}

// RequestAccessToken using an x509 client certificate
func (s *Service) RequestAccessToken(ctx context.Context, opt RequestAccessTokenOptions) op.Result[jsonmodels.DeviceAccessToken] {
	deviceClient := s.Client

	if opt.ClientCert != nil {
		skipVerify := false
		transport := s.Client.Transport()
		if httpTransport, ok := transport.(*http.Transport); ok && httpTransport.TLSClientConfig != nil {
			skipVerify = httpTransport.TLSClientConfig.InsecureSkipVerify
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{*opt.ClientCert},
					InsecureSkipVerify: skipVerify,
				},
			},
		}

		deviceClient = resty.NewWithClient(httpClient).
			SetBaseURL(mtlsEndpoint(s.Client.BaseURL())).
			SetHeader("Accept", types.MimeTypeApplicationJSON)
	}

	req := deviceClient.R().
		SetMethod(resty.MethodPost).
		SetURL(ApiDeviceAccessToken).
		Funcs(core.NoAuthorization())

	if opt.Headers != nil {
		for k, v := range opt.Headers {
			req.SetHeader(k, strings.Join(v, ","))
		}
	}

	return core.Execute(ctx, core.NewTryRequest(deviceClient, req), jsonmodels.NewDeviceAccessToken)
}

// CreateCertificateSigningRequest creates a certificate signing request
func (s *Service) CreateCertificateSigningRequest(externalId string, key any) (*x509.CertificateRequest, error) {
	return certutil.CreateCertificateSigningRequest(pkix.Name{
		CommonName:         externalId,
		Organization:       []string{"Cumulocity"},
		OrganizationalUnit: []string{"Device"},
	}, key)
}

// GenerateOneTimePassword generates a one-time password
func (s *Service) GenerateOneTimePassword(opts ...password.PasswordOption) (string, error) {
	defaults := []password.PasswordOption{
		password.WithLengthConstraints(8, 32),
		password.WithLength(31),
		password.WithUrlCompatibleSymbols(2),
	}
	defaults = append(defaults, opts...)
	return password.NewRandomPassword(defaults...)
}

// mtlsEndpoint returns the host address for the mtls endpoint
func mtlsEndpoint(u string) string {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return u
	}
	out := fmt.Sprintf("%s://%s:%s", parsedURL.Scheme, parsedURL.Hostname(), "8443")
	if parsedURL.Path != "" {
		out = out + "/" + parsedURL.Path
	}
	return out
}

// executeWithCertParse executes a request and parses the PKCS7/PEM certificate response
func executeWithCertParse(ctx context.Context, req *core.TryRequest) op.Result[X509Certificate] {
	resp, err := core.ExecuteResponseOnly(ctx, req)
	if err != nil {
		return op.Failed[X509Certificate](err, true)
	}

	cert, parseErr := parsePKCS7Response(resp.Bytes(), resp.Header())
	if parseErr != nil {
		return op.Failed[X509Certificate](parseErr, false)
	}

	return op.OK(X509Certificate{cert})
}

// parsePKCS7Response parses the PKCS7 response
func parsePKCS7Response(body []byte, headers http.Header) (*x509.Certificate, error) {
	var contents []byte
	var err error

	if transferEncoding := headers.Get("Content-Transfer-Encoding"); transferEncoding == "base64" {
		contents, err = certutil.Base64Decode(body)
		if err != nil {
			return nil, fmt.Errorf("failed to decode response using base64: %w", err)
		}
	} else {
		contents = body
	}

	var cert *x509.Certificate
	contentType := headers.Get("Content-Type")

	if strings.HasPrefix(contentType, "application/pkcs7-mime") {
		p7, p7Err := pkcs7.Parse(contents)
		if p7Err != nil {
			return nil, p7Err
		}
		if len(p7.Certificates) == 0 {
			return nil, fmt.Errorf("response did not contain any x509 certificates")
		}
		cert = p7.Certificates[0]
	} else if strings.HasPrefix(contentType, "application/pkcs10") {
		cert, err = certutil.ParseCertificatePEM(contents)
		if err != nil {
			return nil, err
		}
	}

	if cert == nil {
		return nil, fmt.Errorf("failed to parse certificate from response")
	}

	return cert, nil
}

// X509Certificate wrapper for x509.Certificate
type X509Certificate struct {
	*x509.Certificate
}

// Helper types and methods for polling enrollment

type DeviceEnrollmentOption struct {
	ExternalID                string
	InitDelay                 time.Duration
	Interval                  time.Duration
	Timeout                   time.Duration
	OneTimePassword           string
	CertificateSigningRequest *x509.CertificateRequest
	OnProgressBefore          func()
	OnProgressError           func(error)
	Banner                    *DeviceEnrollmentBannerOptions
}

var DeviceEnrollmentDefaultTemplate = `
{{.Title}}

{{- if .ShowQRCode }}
Scan the QR Code
{{ .QRCode }}
{{- end}}

{{- if .ShowURL }}
Use the following URL

{{.Url}}
{{- end  }}

`

func NewDeviceEnrollmentBannerOptions(showQRCode bool, showURL bool) *DeviceEnrollmentBannerOptions {
	return &DeviceEnrollmentBannerOptions{
		Enable:     true,
		ShowQRCode: showQRCode,
		ShowURL:    showURL,
		Template:   DeviceEnrollmentDefaultTemplate,
	}
}

type DeviceEnrollmentBannerOptions struct {
	Enable     bool
	Template   string
	ShowQRCode bool
	ShowURL    bool
}

type DeviceEnrollmentPollResult struct {
	Err         error
	ExternalID  string
	Certificate *x509.Certificate
	Duration    time.Duration
}

func (r *DeviceEnrollmentPollResult) Ok() bool {
	return r.Err == nil
}

//go:embed device_registration.txt
var DeviceRegistrationHeader string

func (s *Service) printEnrollmentLog(externalID string, oneTimePassword string, opts DeviceEnrollmentBannerOptions) error {
	if opts.Template == "" {
		opts.Template = DeviceEnrollmentDefaultTemplate
	}

	bannerTemplate, err := template.New("registration").Parse(opts.Template)
	if err != nil {
		return err
	}

	fullURL := fmt.Sprintf(
		"%s/apps/devicemanagement/index.html#/deviceregistration?externalId=%s&one-time-password=%s",
		strings.TrimRight(s.Client.BaseURL(), "/"),
		externalID,
		oneTimePassword,
	)

	qrcode := bytes.NewBufferString("")
	qrterminal.GenerateWithConfig(fullURL, qrterminal.Config{
		Level:      qrterminal.M,
		Writer:     qrcode,
		HalfBlocks: true,
		QuietZone:  1,
	})

	b := bytes.NewBufferString("")
	bannerTemplate.Execute(b, struct {
		Title           string
		BaseURL         string
		Url             string
		ShowQRCode      bool
		ShowURL         bool
		QRCode          string
		ExternalID      string
		OneTimePassword string
		Divider         string
	}{
		Title:           DeviceRegistrationHeader,
		BaseURL:         s.Client.BaseURL(),
		Url:             fullURL,
		ExternalID:      externalID,
		OneTimePassword: oneTimePassword,
		ShowQRCode:      opts.ShowQRCode,
		ShowURL:         opts.ShowURL,
		QRCode:          qrcode.String(),
		Divider:         strings.Repeat("-", 80),
	})
	_, err = fmt.Fprintf(os.Stderr, "%s\n", b.String())
	return err
}

// PollEnroll continuously tries to download the x509 certificate for the given device
func (s *Service) PollEnroll(ctx context.Context, opts DeviceEnrollmentOption) <-chan DeviceEnrollmentPollResult {
	if opts.Interval == 0 {
		opts.Interval = 5 * time.Second
	}
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Minute
	}
	if opts.OneTimePassword == "" {
		opts.OneTimePassword, _ = s.GenerateOneTimePassword()
	}

	done := make(chan DeviceEnrollmentPollResult)

	if opts.Banner != nil && opts.Banner.Enable {
		if err := s.printEnrollmentLog(opts.ExternalID, opts.OneTimePassword, *opts.Banner); err != nil {
			slog.Warn("Failed to print enrollment banner", "err", err)
		}
	}

	go func() {
		startedAt := time.Now()

		if opts.InitDelay > 0 {
			time.Sleep(opts.InitDelay)
		}

		ticker := time.NewTicker(opts.Interval)
		timeoutTimer := time.NewTimer(opts.Timeout)

		defer func() {
			ticker.Stop()
			timeoutTimer.Stop()
		}()

		for {
			tick := time.Now()
			if opts.OnProgressBefore != nil {
				opts.OnProgressBefore()
			}

			result := s.Enroll(ctx, EnrollOptions{
				ExternalID:      opts.ExternalID,
				OneTimePassword: opts.OneTimePassword,
				CSR:             opts.CertificateSigningRequest,
			})

			if result.Err != nil {
				if opts.OnProgressError != nil {
					opts.OnProgressError(result.Err)
				}
			} else {
				done <- DeviceEnrollmentPollResult{
					ExternalID:  opts.ExternalID,
					Certificate: result.Data.Certificate,
					Duration:    tick.Sub(startedAt),
				}
				return
			}

			select {
			case <-ctx.Done():
				done <- DeviceEnrollmentPollResult{
					Err:         ctx.Err(),
					ExternalID:  opts.ExternalID,
					Certificate: nil,
					Duration:    time.Since(startedAt),
				}
				return
			case <-ticker.C:
				continue
			case tick := <-timeoutTimer.C:
				done <- DeviceEnrollmentPollResult{
					Err:        errors.New("timeout trying to download certificate"),
					ExternalID: opts.ExternalID,
					Duration:   tick.Sub(startedAt),
				}
				return
			}
		}
	}()

	return done
}
