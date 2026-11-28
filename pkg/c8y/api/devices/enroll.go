package devices

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices/enrollment"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/devices/registration"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// EnrollOptions describes a device for Enroll.
type EnrollOptions struct {
	// ExternalID of the device (required). It must be the common name of the CSR
	ExternalID string

	// Type of the device, used when Register registers it
	Type string

	// OneTimePassword the device requests its certificate with. A random one is generated when empty
	OneTimePassword string

	// CSR is the device's certificate signing request (required)
	CSR *x509.CertificateRequest

	// RemoveRequest removes any device request with the external ID first, so
	// a request left by an earlier attempt doesn't block the registration.
	// It needs the client's credentials; an error is ignored
	RemoveRequest bool

	// Register registers the device (as an agent, with the one-time password)
	// with the client's credentials, so no user needs to. When that fails, a
	// user registers the device (see OnRegistrationRequired)
	Register bool

	// Interval between the attempts to download the certificate. Defaults to 5s
	Interval time.Duration

	// Timeout after which Enroll gives up. Defaults to 10 minutes
	Timeout time.Duration

	// OnRegistrationRequired is called when the device waits for a user to
	// register it, for example to show the registration URL
	OnRegistrationRequired func(RegistrationRequired)

	// OnAttemptError is called with the error of each failed attempt to
	// download the certificate
	OnAttemptError func(error)
}

// RegistrationRequired tells a user how to register a device which waits to
// enroll.
type RegistrationRequired struct {
	ExternalID      string
	OneTimePassword string

	// URL of the device registration in the Device management application,
	// with the external ID and one-time password filled in
	URL string

	// Err is why the device could not be registered, if Register was set
	Err error
}

// Enroll gets a certificate for a device from the Cumulocity certificate
// authority. It registers the device (see EnrollOptions.Register) or waits for
// a user to register it, then requests the certificate with the CSR until the
// registration is done or the timeout passes.
//
// The enrollment needs no credentials (the device authenticates with its
// one-time password), so it works with a client which only has a base URL.
//
// In a dry run the requests are shown and nothing is returned.
func (s *Service) Enroll(ctx context.Context, opt EnrollOptions) op.Result[enrollment.X509Certificate] {
	if opt.ExternalID == "" {
		return op.Failed[enrollment.X509Certificate](fmt.Errorf("enroll: an external ID is required"), false)
	}
	if opt.CSR == nil {
		return op.Failed[enrollment.X509Certificate](fmt.Errorf("enroll: a certificate signing request is required"), false)
	}
	if opt.Interval <= 0 {
		opt.Interval = 5 * time.Second
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 10 * time.Minute
	}
	if opt.OneTimePassword == "" {
		otp, err := s.Enrollment.GenerateOneTimePassword()
		if err != nil {
			return op.Failed[enrollment.X509Certificate](err, false)
		}
		// Some consoles don't make a URL ending in "." clickable
		opt.OneTimePassword = strings.ReplaceAll(otp, ".", "_")
	}

	if opt.RemoveRequest {
		if result := s.Registration.Delete(ctxhelpers.InternalStep(ctx), opt.ExternalID); result.Err == nil {
			slog.Debug("Removed the device request", "externalID", opt.ExternalID)
		}
	}

	registered := false
	var registerErr error
	if opt.Register {
		result := s.Registration.Register(ctx, registration.RegisterOptions{
			ExternalID:    opt.ExternalID,
			Type:          opt.Type,
			AuthType:      registration.AuthTypeCertificates,
			EnrollmentOTP: opt.OneTimePassword,
			IsAgent:       true,
		})
		registered, registerErr = result.Err == nil, result.Err
	}

	dryRun := ctxhelpers.IsDryRun(ctx)
	if !registered && opt.OnRegistrationRequired != nil {
		opt.OnRegistrationRequired(RegistrationRequired{
			ExternalID:      opt.ExternalID,
			OneTimePassword: opt.OneTimePassword,
			URL:             s.Enrollment.EnrollmentURL(opt.ExternalID, opt.OneTimePassword),
			Err:             registerErr,
		})
	}

	timeout := time.NewTimer(opt.Timeout)
	defer timeout.Stop()

	// Until the device is registered, an attempt is rejected
	for {
		result := s.Enrollment.Enroll(ctx, enrollment.EnrollOptions{
			ExternalID:      opt.ExternalID,
			OneTimePassword: opt.OneTimePassword,
			CSR:             opt.CSR,
		})
		if result.Err == nil || dryRun {
			return result
		}
		if opt.OnAttemptError != nil {
			opt.OnAttemptError(result.Err)
		}
		if err := wait(ctx, timeout.C, opt.Interval); err != nil {
			// The last error is only described: it is the reason for the timeout, not the error
			return op.Failed[enrollment.X509Certificate](fmt.Errorf("%w. last error: %v", err, result.Err), false) //nolint:errorlint // the second error is described, not wrapped
		}
	}
}

// ErrEnrollTimeout is the error of an enrollment which timed out. It is a
// context.DeadlineExceeded error.
var ErrEnrollTimeout = fmt.Errorf("the device's certificate could not be downloaded before the timeout: %w", context.DeadlineExceeded)

// wait waits for the delay, or returns an error when the context is done or
// the timeout fires first.
func wait(ctx context.Context, timeout <-chan time.Time, delay time.Duration) error {
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timeout:
		return ErrEnrollTimeout
	case <-t.C:
		return nil
	}
}
