package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
)

func stopOnError(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	client := c8y.NewClientFromEnvironment(nil, false)

	if len(os.Args) < 2 {
		log.Fatal("Expected the device id (external id) as the first argument")
	}
	deviceID := os.Args[1]

	// Create private key
	keyPem, err := certutil.MakeEllipticPrivateKeyPEM()
	stopOnError(err)

	key, err := certutil.ParsePrivateKeyPEM(keyPem)
	stopOnError(err)

	fmt.Fprintf(os.Stderr, "\n📣 Starting device enrollment: externalID=%s\n", deviceID)

	// Create CSR
	csr, err := certutil.CreateCertificateSigningRequest(deviceID, key)
	stopOnError(err)

	ctx := c8y.NewSilentLoggerContext(context.Background())

	// Enroll device
	result := <-client.DeviceEnrollment.PollEnroll(ctx, c8y.DeviceEnrollmentOption{
		ExternalID:      deviceID,
		OneTimePassword: "", // Generate random one-time password

		// Initial delay before the first download attempt
		InitDelay: 2 * time.Second,

		// Check every 5 seconds
		Interval: 5 * time.Second,

		// Give up after 10 minutes
		Timeout: 10 * time.Minute,

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
			fmt.Fprintf(os.Stderr, "WAITING (last statusCode=%s, time=%s)", r.Status(), time.Now().Format(time.RFC3339))
		},
	})
	if result.Err != nil {
		fmt.Fprintf(os.Stderr, "🚫 Failed to download the device's certificate\n")
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "✅ Successfully downloaded the device's certificate\n")

	cert := result.Certificate
	certPEM := certutil.MarshalCertificateToPEM(cert.Raw)
	fmt.Printf("%s", certPEM)
}
