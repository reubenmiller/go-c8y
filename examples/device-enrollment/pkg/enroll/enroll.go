package enroll

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/reubenmiller/example/pkg/cli"
	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

/*
Enroll command

Register a device using the Cumulocity Certificate Authority feature where a device
certificate will be requested using an auto generated password that the user needs to
then note, and register it in Cumulocity. To easy the process, a QR Code and a clickable
URL is printed on the console.
*/
type EnrollCmd struct {
	Timeout         time.Duration `name:"timeout" help:"Timeout for enrollment" type:"time.Duration"`
	RetryEvery      time.Duration `name:"retry-every" help:"Interval to retry downloading the certificate" type:"time.Duration"`
	OneTimePassword string        `name:"one-time-password" help:"one-time password used for enrollment authorization. Default is to auto generate a password" type:"string"`
	Overwrite       bool          `name:"overwrite" help:"Overwrite existing private key and public certificate"`
}

func (r *EnrollCmd) Run(ctx *cli.Context) error {
	deviceID := ctx.DeviceID

	if deviceID == "" {
		return fmt.Errorf("🚫 device id (external id) is not set")
	}

	client := c8y.NewClient(nil, ctx.Host, "", "", "", true)

	if r.Overwrite {
		slog.Info("Removing any existing private key, or certificate")
		if err := os.Remove(ctx.KeyFile); err != nil {
			slog.Warn("Failed to remove private key", "file", ctx.KeyFile, "error", err)
		}
		if err := os.Remove(ctx.CertFile); err != nil {
			slog.Warn("Failed to remove certificate", "file", ctx.CertFile, "error", err)
		}
	}

	// Create private key
	keyPem, keyWasGenerated, err := certutil.LoadOrGenerateKeyFile(ctx.KeyFile)
	if err != nil {
		panic(fmt.Errorf("failed to load or create private key. %w", err))
	}
	if keyWasGenerated {
		slog.Info("Created a new private key.", "file", ctx.KeyFile)
	} else {
		slog.Info("Loaded an existing private key.", "file", ctx.KeyFile)
	}

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	if err != nil {
		panic(fmt.Errorf("failed to parse private key. %w", err))
	}

	var cert *x509.Certificate
	if _, err := os.Stat(ctx.CertFile); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "\n📣 Starting device enrollment: externalID=%s\n", deviceID)

		// Create CSR
		csr, err := client.DeviceEnrollment.CreateCertificateSigningRequest(deviceID, key)
		if err != nil {
			panic(fmt.Errorf("failed to create certificate signing request. %w", err))
		}

		clientCtx := c8y.NewSilentLoggerContext(context.Background())

		// Enroll device
		result := <-client.DeviceEnrollment.PollEnroll(clientCtx, c8y.DeviceEnrollmentOption{
			ExternalID:      deviceID,
			OneTimePassword: r.OneTimePassword, // Generate random one-time password if empty

			// Initial delay before the first download attempt
			InitDelay: 2 * time.Second,

			// Check every 5 seconds
			Interval: r.RetryEvery,

			// Give up after 10 minutes
			Timeout: r.Timeout,

			// Print enrollment information
			Banner: &c8y.DeviceEnrollmentBannerOptions{
				Enable:     true,
				ShowQRCode: true,
				ShowURL:    true,
			},

			CertificateSigningRequest: csr,

			// Progress information
			OnProgressBefore: func() {
				fmt.Fprintf(os.Stderr, "\rTrying to download certificate: ")
			},
			OnProgressError: func(r *c8y.Response, err error) {
				if r != nil {
					fmt.Fprintf(os.Stderr, "WAITING (last statusCode=%s, time=%s)", r.Status(), time.Now().Format(time.RFC3339))
				} else {
					fmt.Fprintf(os.Stderr, "WAITING (last statusCode=%s, time=%s)", "0", time.Now().Format(time.RFC3339))
				}
			},
		})
		if result.Err != nil {
			fmt.Fprintf(os.Stderr, "🚫 Failed to download the device's certificate\n")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "✅ Successfully downloaded the device's certificate\n")

		cert = result.Certificate
		certPEM := certutil.MarshalCertificateToPEM(cert.Raw)
		os.WriteFile(ctx.CertFile, certPEM, 0644)
	} else {
		certPEM, err := os.ReadFile(ctx.CertFile)
		if err != nil {
			panic(fmt.Errorf("failed to read certificate file. %w", err))
		}

		cert, err = certutil.ParseCertificatePEM(certPEM)
		if err != nil {
			panic(fmt.Errorf("failed to parse certificate. %w", err))
		}
		fmt.Fprintf(os.Stderr, "\n📣 Using existing device certificate: externalID=%s\n", cert.Subject.CommonName)
	}
	return nil
}
