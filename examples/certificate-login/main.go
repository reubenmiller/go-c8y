// Command certificate-login logs in to Cumulocity with a client certificate.
//
// RSA keys use the HTTPS token exchange (POST /tenant/oauth/certificate); EC
// and Ed25519 keys use mTLS on port 8443. The account must have opted in to
// certificate authentication (device users need no opt-in).
//
// Usage:
//
//	C8Y_HOST=https://example.cumulocity.com C8Y_TENANT=t12345 \
//	  go run ./examples/certificate-login user.crt user.key
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: certificate-login <certificate> <key>")
	}
	client, err := api.NewClient(api.ClientOptions{
		BaseURL: os.Getenv("C8Y_HOST"),
		Auth: authentication.AuthOptions{
			Tenant:         os.Getenv("C8Y_TENANT"),
			Certificate:    os.Args[1],
			CertificateKey: os.Args[2],
		},
	})
	if err != nil {
		log.Fatalf("Could not create the client: %v", err)
	}

	ctx := context.Background()
	if _, err := client.LoginWithOptions(ctx, api.LoginOptions{
		Method: authentication.LoginMethodCertificate,
	}); err != nil {
		log.Fatalf("certificate login failed: %v", err)
	}

	result := client.ManagedObjects.Count(ctx, managedobjects.ListOptions{})
	if result.Err != nil {
		log.Fatalf("count managed objects: %v", result.Err)
	}
	fmt.Printf("logged in with a certificate; %d managed objects visible\n", result.Data)
}
