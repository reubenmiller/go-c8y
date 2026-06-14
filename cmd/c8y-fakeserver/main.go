// Command c8y-fakeserver runs the stateful in-memory fake Cumulocity server
// (internal/pkg/fakeserver) as a standalone process so out-of-process
// consumers — notably the go-c8y-cli commander test suite — can point
// C8Y_HOST at a real localhost endpoint without a tenant.
//
// Usage:
//
//	c8y-fakeserver --addr 127.0.0.1:8111
//
// It prints the listening URL (the line "C8Y_HOST=<url>") and blocks until
// interrupted. Credentials: any non-empty basic auth is accepted for normal
// endpoints; the seeded admin user is "admin"/"admin-pass" on the credential
// endpoint.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/fakeserver"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "address to listen on (host:port); port 0 picks a free port")
	flag.Parse()

	fs, err := fakeserver.NewServer(*addr)
	if err != nil {
		log.Fatalf("failed to start fake server: %v", err)
	}
	defer fs.Server.Close()

	// Emit the URL in an env-assignable form so callers can `eval` or grep it.
	fmt.Printf("C8Y_HOST=%s\n", fs.URL())
	fmt.Fprintf(os.Stderr, "fake Cumulocity server listening on %s (tenant t12345, user admin)\n", fs.URL())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Fprintln(os.Stderr, "shutting down")
}
