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

// Value option
type Value func() (string, error)

// From Arg returns a from the app's arguments
func FromArg(i int) Value {
	return func() (string, error) {
		if i < len(os.Args) {
			return os.Args[i], nil
		}
		return "", nil
	}
}

// FromEnv returns a value from the environment
func FromEnv(key string) Value {
	return func() (string, error) {
		return os.Getenv(key), nil
	}
}

// GetValueWithOptions set a value from
func GetValueWithOptions(opts ...Value) string {
	for _, opt := range opts {
		if v, err := opt(); err == nil && v != "" {
			return v
		}
	}
	return ""
}

// Register a device using the Cumulocity Certificate Authority feature where a device
// certificate will be requested using an auto generated password that the user needs to
// then note, and register it in Cumulocity. To easy the process, a QR Code and a clickable
// URL is printed on the console.
//
// The example only needs to values to work:
// C8Y_HOST - Cumulocity URL to the tenant you want to register with (via env)
// DEVICE_ID - Via Argument, Env or default to the device's hostname
func main() {
	// Only the target Cumulocity is required as registration will
	c8yHost := GetValueWithOptions(
		FromEnv("C8Y_HOST"),
		FromEnv("C8Y_URL"),
		FromEnv("C8Y_BASEURL"),
	)
	if c8yHost == "" {
		log.Fatal("🚫 The C8Y_HOST is not set")
	}
	client := c8y.NewClient(nil, c8yHost, "", "", "", true)

	// choose first non-empty value
	deviceID := GetValueWithOptions(
		FromArg(1),
		FromEnv("DEVICE_ID"),
		os.Hostname,
	)

	if deviceID == "" {
		log.Fatal("🚫 The device id (external id) is not set")
	}

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

	cert := result.Certificate
	certPEM := certutil.MarshalCertificateToPEM(cert.Raw)
	fmt.Printf("%s", certPEM)
}
