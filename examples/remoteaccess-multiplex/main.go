// Example: local port forwarding to a device via Cumulocity Cloud Remote Access, optionally
// multiplexing all connections over a single remote access session (thin-edge.io devices with
// remote access multiplexing support). Devices without support fall back to one session per
// connection automatically.
//
// Serve a local port (like "c8y remoteaccess server"):
//
//	go run ./examples/remoteaccess-multiplex -device 12345 -config 1 -multiplex
//
// Compare request timings with and without multiplexing (HTTP or HTTPS service on the device):
//
//	go run ./examples/remoteaccess-multiplex -device 12345 -config 1 -compare -https
//
// The client is created from the environment variables C8Y_HOST, C8Y_TENANT, C8Y_USER, C8Y_PASSWORD.
package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y"
	"github.com/reubenmiller/go-c8y/pkg/remoteaccess"
)

func main() {
	device := flag.String("device", "", "managed object id of the device")
	config := flag.String("config", "", "remote access configuration id (PASSTHROUGH)")
	listen := flag.String("listen", "127.0.0.1:0", "local address to listen on")
	multiplex := flag.Bool("multiplex", false, "multiplex all connections over a single remote access session")
	compare := flag.Bool("compare", false, "measure request timings with and without multiplexing, then exit")
	useHTTPS := flag.Bool("https", false, "the service on the device uses HTTPS (certificate is not verified)")
	path := flag.String("path", "/", "HTTP path requested by -compare")
	requests := flag.Int("requests", 10, "number of sequential requests (-compare)")
	parallel := flag.Int("parallel", 6, "number of parallel requests per burst (-compare)")
	flag.Parse()

	if *device == "" || *config == "" {
		flag.Usage()
		os.Exit(2)
	}
	client := c8y.NewClientFromEnvironment(nil, true)

	if !*compare {
		ra := newRemoteAccess(client, *device, *config, *multiplex, *listen)
		log.Printf("Listening on %s (multiplex=%v)", ra.GetListenerAddress(), *multiplex)
		if err := ra.Serve(); err != nil {
			log.Fatal(err)
		}
		return
	}

	scheme := "http"
	if *useHTTPS {
		scheme = "https"
	}
	fmt.Printf("%-14s %10s %22s %24s\n", "mode", "first", "following (median)", fmt.Sprintf("%d parallel (total)", *parallel))
	for _, mode := range []bool{false, true} {
		ra := newRemoteAccess(client, *device, *config, mode, "127.0.0.1:0")
		go func() { _ = ra.Serve() }()
		url := fmt.Sprintf("%s://%s%s", scheme, ra.GetListenerAddress(), *path)
		result := benchmark(url, *requests, *parallel)
		name := "per-connection"
		if mode {
			name = "multiplexed"
		}
		fmt.Printf("%-14s %10s %22s %24s\n", name, result.first.Round(time.Millisecond),
			result.following.Round(time.Millisecond), result.parallel.Round(time.Millisecond))
	}
}

func newRemoteAccess(client *c8y.Client, device, config string, multiplex bool, listen string) *remoteaccess.RemoteAccessClient {
	ra := remoteaccess.NewRemoteAccessClient(client, remoteaccess.RemoteAccessOptions{
		ManagedObjectID: device,
		RemoteAccessID:  config,
		Multiplex:       multiplex,
	})
	if err := ra.Listen(listen); err != nil {
		log.Fatal(err)
	}
	return ra
}

type result struct {
	first     time.Duration
	following time.Duration
	parallel  time.Duration
}

// benchmark uses a new connection for every request (no keep-alive), as many clients do and as
// servers closing every connection (e.g. "Connection: close") require
func benchmark(url string, requests int, parallel int) result {
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // devices use self-signed certificates
		},
	}
	get := func() time.Duration {
		start := time.Now()
		resp, err := httpClient.Get(url)
		if err != nil {
			log.Printf("request failed: %v", err)
			return 0
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return time.Since(start)
	}

	r := result{first: get()}
	samples := make([]time.Duration, 0, requests)
	for i := 0; i < requests; i++ {
		samples = append(samples, get())
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	r.following = samples[len(samples)/2]

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			get()
		}()
	}
	wg.Wait()
	r.parallel = time.Since(start)
	return r
}
