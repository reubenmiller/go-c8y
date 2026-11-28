package enrollment

import (
	"bytes"
	"context"
	"crypto/x509"
	"crypto/x509/pkix"
	_ "embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/reubenmiller/go-c8y/v2/pkg/certutil"
	"github.com/reubenmiller/go-c8y/v2/pkg/password"
	"github.com/smallstep/pkcs7"
	"resty.dev/v3"
)

var apiEnroll = ".well-known/est/simpleenroll"
var apiReEnroll = ".well-known/est/simplereenroll"

// Service provides device enrollment functionality to enroll new devices and receive device certificates
type Service struct{ svc httpcore.Service }

func NewService(s *httpcore.Service) *Service {
	return &Service{svc: *s}
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

	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Transfer-Encoding", "base64").
		SetContentType("application/pkcs10").
		SetBody(base64.StdEncoding.EncodeToString(opt.CSR.Raw)).
		SetURL(apiEnroll)

	b := httpcore.NewTryRequest(s.svc.Client, req).
		SetBasicAuth(opt.ExternalID, opt.OneTimePassword)
	return executeWithCertParse(ctx, b)
}

// ReEnrollOptions options for re-enrolling a device
type ReEnrollOptions struct {
	// Certificate Signing Request to use to re-issue a certificate from Cumulocity
	CSR *x509.CertificateRequest

	// Override the default token. Don't set this unless you know what you are doing
	Token string
}

// ReEnroll an already enrolled device using an existing device certificate
func (s *Service) ReEnroll(ctx context.Context, opt ReEnrollOptions) op.Result[X509Certificate] {
	if opt.CSR == nil {
		return op.Failed[X509Certificate](fmt.Errorf("certificate signing request is required"), false)
	}

	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Content-Transfer-Encoding", "base64").
		SetContentType("application/pkcs10").
		SetBody(base64.StdEncoding.EncodeToString(opt.CSR.Raw)).
		SetURL(apiReEnroll)

	b := httpcore.NewTryRequest(s.svc.Client, req)
	if opt.Token != "" {
		b.SetToken(opt.Token)
	}
	return executeWithCertParse(ctx, b)
}

// CreateCertificateSigningRequest creates a certificate signing request
func (s *Service) CreateCertificateSigningRequest(externalId string, key any) (*x509.CertificateRequest, error) {
	return certutil.CreateCertificateSigningRequest(pkix.Name{
		CommonName:         externalId,
		Organization:       []string{"Cumulocity"}, //nolint:misspell
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

// executeWithCertParse executes a request and parses the PKCS7/PEM certificate response
func executeWithCertParse(ctx context.Context, req *httpcore.TryRequest) op.Result[X509Certificate] {
	resp, err := httpcore.ExecuteResponseOnly(ctx, req)

	// Only capture request in dry run mode for inspection
	var httpReq *http.Request
	if resp != nil && ctxhelpers.IsDryRun(ctx) {
		httpReq = resp.Request.RawRequest
	}

	if err != nil {
		result := op.Failed[X509Certificate](err, true)
		if resp != nil {
			result = op.WithHTTPStatus(op.WithDuration(result, resp.Duration()), resp.StatusCode())
		} else {
			// Extract HTTP status from error if response is nil
			var apiErr *core.Error
			if errors.As(err, &apiErr) && apiErr.Code >= 100 && apiErr.Code < 600 {
				result = op.WithHTTPStatus(result, apiErr.Code)
				if apiErr.Duration > 0 {
					result = op.WithDuration(result, apiErr.Duration)
				}
			}
		}
		return op.WithRequest(result, httpReq)
	}

	// In dry run mode, skip certificate parsing and return a placeholder result
	if ctxhelpers.IsDryRun(ctx) {
		return op.WithRequest(op.WithHTTPStatus(op.WithDuration(op.OK(X509Certificate{}), resp.Duration()), resp.StatusCode()), httpReq)
	}

	cert, parseErr := parsePKCS7Response(resp.Bytes(), resp.Header())
	if parseErr != nil {
		return op.WithRequest(op.WithHTTPStatus(op.WithDuration(op.Failed[X509Certificate](parseErr, false), resp.Duration()), resp.StatusCode()), httpReq)
	}

	return op.WithRequest(op.WithHTTPStatus(op.WithDuration(op.OK(X509Certificate{cert}), resp.Duration()), resp.StatusCode()), httpReq)
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

// DeviceEnrollmentOptions are the options of [Service.PollEnroll].
type DeviceEnrollmentOptions struct {
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

// QRCodeRenderer renders the enrollment URL as a terminal-friendly string (e.g. an
// ASCII/Unicode QR code). It is supplied by the caller so that the core SDK does not
// depend on any QR-code rendering library; a CLI can plug in its renderer of choice.
type QRCodeRenderer func(url string) (string, error)

// NewDeviceEnrollmentBannerOptions creates banner options for PollEnroll. To render a
// QR code, pass a QRCodeRenderer; without one the QR section is left empty even when
// showQRCode is true (the core SDK does not bundle a QR-code library).
func NewDeviceEnrollmentBannerOptions(showQRCode bool, showURL bool, renderer ...QRCodeRenderer) *DeviceEnrollmentBannerOptions {
	opts := &DeviceEnrollmentBannerOptions{
		Enable:     true,
		ShowQRCode: showQRCode,
		ShowURL:    showURL,
		Template:   DeviceEnrollmentDefaultTemplate,
	}
	if len(renderer) > 0 {
		opts.QRCodeRenderer = renderer[0]
	}
	return opts
}

type DeviceEnrollmentBannerOptions struct {
	Enable     bool
	Template   string
	ShowQRCode bool
	ShowURL    bool

	// QRCodeRenderer renders the enrollment URL as a QR code for the banner. If nil,
	// the QR code section is left empty even when ShowQRCode is true.
	QRCodeRenderer QRCodeRenderer
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

// EnrollmentURL returns the device registration URL for the given external ID and
// one-time password. Callers (e.g. a CLI) can render this however they like — for
// example as a clickable link or a QR code.
func (s *Service) EnrollmentURL(externalID string, oneTimePassword string) string {
	return fmt.Sprintf(
		"%s/apps/devicemanagement/index.html#/deviceregistration?externalId=%s&one-time-password=%s",
		strings.TrimRight(s.svc.Client.BaseURL(), "/"),
		externalID,
		oneTimePassword,
	)
}

// WriteBanner writes the banner which asks a user to register the device, with
// the registration URL and, if a QRCodeRenderer is given, its QR code.
func (s *Service) WriteBanner(w io.Writer, externalID string, oneTimePassword string, opts DeviceEnrollmentBannerOptions) error {
	if opts.Template == "" {
		opts.Template = DeviceEnrollmentDefaultTemplate
	}

	bannerTemplate, err := template.New("registration").Parse(opts.Template)
	if err != nil {
		return err
	}

	fullURL := s.EnrollmentURL(externalID, oneTimePassword)

	qrcode := ""
	if opts.ShowQRCode && opts.QRCodeRenderer != nil {
		qrcode, err = opts.QRCodeRenderer(fullURL)
		if err != nil {
			return err
		}
	}

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
		BaseURL:         s.svc.Client.BaseURL(),
		Url:             fullURL,
		ExternalID:      externalID,
		OneTimePassword: oneTimePassword,
		ShowQRCode:      opts.ShowQRCode,
		ShowURL:         opts.ShowURL,
		QRCode:          qrcode,
		Divider:         strings.Repeat("-", 80),
	})
	_, err = fmt.Fprintf(w, "%s\n", b.String())
	return err
}

// PollEnroll continuously tries to download the x509 certificate for the given device
func (s *Service) PollEnroll(ctx context.Context, opts DeviceEnrollmentOptions) <-chan DeviceEnrollmentPollResult {
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
		if err := s.WriteBanner(os.Stderr, opts.ExternalID, opts.OneTimePassword, *opts.Banner); err != nil {
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
