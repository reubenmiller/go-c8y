package c8y

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tidwall/gjson"
)

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

type Cacheable func(*http.Request) bool

func NewCachedClient(httpClient *http.Client, cacheDir string, cacheTTL time.Duration, isCacheable Cacheable, opts CacheOptions) *http.Client {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "go-c8y-cache")
	}
	if isCacheable == nil {
		isCacheable = isCacheableRequest
	}
	return &http.Client{
		Transport: NewCachedTransport(httpClient.Transport, cacheTTL, cacheDir, isCacheable, opts),
	}
}

func isCacheableRequest(req *http.Request) bool {
	if strings.EqualFold(req.Method, "GET") || strings.EqualFold(req.Method, "HEAD") {
		return true
	}

	if strings.EqualFold(req.Method, "POST") && strings.Contains(req.URL.Path, "/service/") {
		return true
	}

	return false
}

func isCacheableResponse(res *http.Response) bool {
	return res.StatusCode < 300
}

// NewCachedTransport creates an http.RoundTripper that caches HTTP responses to disk.
// This is a convenience function for using with clients like Resty that need a direct RoundTripper.
//
// Cached responses include the following headers to indicate cache status:
//   - X-Cache: "HIT" (from cache) or "MISS" (fresh response)
//   - X-From-Cache: "true" (only present on cached responses)
//   - Age: number of seconds since the response was cached
//
// When using the API client methods, cache headers are available in Result.Meta:
//
//	result := client.Alarms.List(ctx, options)
//	if result.Meta["x-cache"] == "HIT" {
//		fmt.Printf("Response from cache (age: %s seconds)\n", result.Meta["age"])
//	}
//
// When using Resty directly:
//
//	client := resty.New()
//	transport := c8y.NewCachedTransport(nil, 5*time.Minute, cacheDir, nil, c8y.CacheOptions{})
//	client.SetTransport(transport)
//
//	resp, err := client.R().Get("/some/endpoint")
//	if resp.Header().Get("X-Cache") == "HIT" {
//		fmt.Printf("Response from cache (age: %s seconds)\n", resp.Header().Get("Age"))
//	}
//
// To set TLS config on the cached transport:
//
//	transport := c8y.NewCachedTransport(nil, 5*time.Minute, cacheDir, nil, c8y.CacheOptions{})
//	// Option 1: Use SetTLSClientConfig (propagates to underlying transport)
//	if cached, ok := transport.(*c8y.CachedRoundTripper); ok {
//		cached.SetTLSClientConfig(tlsConfig)
//	}
//	// Option 2: Access base transport directly
//	if cached, ok := transport.(*c8y.CachedRoundTripper); ok {
//		if httpTransport, ok := cached.BaseTransport().(*http.Transport); ok {
//			httpTransport.TLSClientConfig = tlsConfig
//		}
//	}
//
// To chain multiple transports, pass one as the baseTransport to the next:
//
//	// Chain: Default → Logging → Caching → Resty
//	base := http.DefaultTransport
//	logged := NewLoggingTransport(base)
//	cached := c8y.NewCachedTransport(logged, 5*time.Minute, cacheDir, nil, c8y.CacheOptions{})
//	client.SetTransport(cached)
//
// If baseTransport is nil, http.DefaultTransport is used.
func NewCachedTransport(baseTransport http.RoundTripper, ttl time.Duration, dir string, isCacheable Cacheable, options CacheOptions) http.RoundTripper {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	if isCacheable == nil {
		isCacheable = isCacheableRequest
	}
	return &CachedRoundTripper{
		base: baseTransport,
		storage: fileStorage{
			dir: dir,
			ttl: ttl,
			mu:  &sync.RWMutex{},
		},
		isCacheable: isCacheable,
		options:     options,
	}
}

// CachedRoundTripper implements http.RoundTripper with response caching.
type CachedRoundTripper struct {
	base        http.RoundTripper
	storage     fileStorage
	isCacheable Cacheable
	options     CacheOptions
}

// RoundTrip implements http.RoundTripper.
func (c *CachedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if !c.isCacheable(req) {
		return c.base.RoundTrip(req)
	}

	key, keyErr := cacheKey(req, c.options)
	// Ignore read from cache in write only mode
	if keyErr == nil && c.options.Mode != StoreModeWrite {
		if res, err := c.storage.read(key); err == nil {
			res.Request = req
			// Add cache indicators
			res.Header.Set("X-Cache", "HIT")
			res.Header.Set("X-From-Cache", "true")
			return res, nil
		}
	}

	res, err := c.base.RoundTrip(req)
	if err == nil && keyErr == nil && isCacheableResponse(res) {
		_ = c.storage.store(key, res)
		// Indicate this is a fresh response that was cached
		res.Header.Set("X-Cache", "MISS")
	}
	return res, err
}

// BaseTransport returns the underlying transport, allowing direct configuration.
// Useful when you need to configure settings not exposed by the wrapper.
//
// Example:
//
//	if cached, ok := transport.(*c8y.CachedRoundTripper); ok {
//		if httpTransport, ok := cached.BaseTransport().(*http.Transport); ok {
//			httpTransport.TLSClientConfig = tlsConfig
//		}
//	}
func (c *CachedRoundTripper) BaseTransport() http.RoundTripper {
	return c.base
}

// TLSClientConfig() *tls.Config
//     SetTLSClientConfig(*tls.Config) error

func (c *CachedRoundTripper) TLSClientConfig() *tls.Config {
	if rt, ok := c.BaseTransport().(*http.Transport); ok {
		return rt.TLSClientConfig
	}
	return nil
}

// SetTLSClientConfig sets the TLS config on the underlying transport if possible.
func (c *CachedRoundTripper) SetTLSClientConfig(tlsConfig *tls.Config) error {
	if c.base == nil {
		return errors.New("no base transport")
	}

	switch t := c.base.(type) {
	case *http.Transport:
		t.TLSClientConfig = tlsConfig
		return nil
	case interface{ SetTLSClientConfig(*tls.Config) error }:
		return t.SetTLSClientConfig(tlsConfig)
	}

	return errors.New("base transport does not support TLS configuration")
}

func copyStream(r io.ReadCloser) (io.ReadCloser, io.ReadCloser) {
	b := &bytes.Buffer{}
	nr := io.TeeReader(r, b)
	return io.NopCloser(b), &readCloser{
		Reader: nr,
		Closer: r,
	}
}

type readCloser struct {
	io.Reader
	io.Closer
}

type StoreMode int

const (
	// StoreModeReadWrite read and write to cache
	StoreModeReadWrite StoreMode = 0

	// StoreModeWrite only write to cache, don't read from it.
	StoreModeWrite StoreMode = 1
)

// CacheOptions Cache key generation options
type CacheOptions struct {
	// ExcludeAuth excludes Authorization header value
	ExcludeAuth bool

	// ExcludeHost excludes Host from the full URL value
	ExcludeHost bool

	// Mode cache store mode which controls the read and writes into cache
	Mode StoreMode

	// BodyKeys Only cache on specific json keys on the body
	BodyKeys []string
}

func cacheKey(req *http.Request, opt CacheOptions) (string, error) {
	h := sha256.New()
	fmt.Fprintf(h, "%s:", req.Method)
	if opt.ExcludeHost {
		// only include path and query
		fmt.Fprintf(h, "%s:", req.URL.RequestURI())
	} else {
		fmt.Fprintf(h, "%s:", req.URL.String())
	}
	fmt.Fprintf(h, "%s:", req.Header.Get("Accept"))

	if !opt.ExcludeAuth {
		fmt.Fprintf(h, "%s:", req.Header.Get("Authorization"))
	}

	if req.Body != nil {
		var bodyCopy io.ReadCloser
		req.Body, bodyCopy = copyStream(req.Body)
		defer bodyCopy.Close()

		if len(opt.BodyKeys) > 0 && strings.Contains(req.Header.Get("Accept"), "json") && strings.Contains(req.Header.Get("Accept"), "application") {
			bodyBytes, err := io.ReadAll(bodyCopy)

			if err != nil {
				return "", err
			}

			fragments := gjson.GetManyBytes(bodyBytes, opt.BodyKeys...)

			for i, fragment := range fragments {
				if fragment.Exists() {
					fmt.Fprintf(h, "%s:%s", opt.BodyKeys[i], fragment.Raw)
				}
			}

		} else {
			if _, err := io.Copy(h, bodyCopy); err != nil {
				return "", err
			}
		}
	}

	digest := h.Sum(nil)
	return fmt.Sprintf("%x", digest), nil
}

type fileStorage struct {
	dir string
	ttl time.Duration
	mu  *sync.RWMutex
}

func (fs *fileStorage) filePath(key string) string {
	if len(key) >= 6 {
		return filepath.Join(fs.dir, key[0:2], key[2:4], key[4:])
	}
	return filepath.Join(fs.dir, key)
}

func (fs *fileStorage) read(key string) (*http.Response, error) {
	cacheFile := fs.filePath(key)

	fs.mu.RLock()
	defer fs.mu.RUnlock()

	f, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	age := time.Since(stat.ModTime())
	if age > fs.ttl {
		return nil, errors.New("cache expired")
	}

	slog.Info("Using cached response", "file", cacheFile, "age", age, "ttl", fs.ttl)

	body := &bytes.Buffer{}
	_, err = io.Copy(body, f)
	if err != nil {
		return nil, err
	}

	res, err := http.ReadResponse(bufio.NewReader(body), nil)
	if res.Header.Get("ETag") == "" {
		res.Header.Set("ETag", key)
	}
	if res.Header.Get("Last-Modified") == "" {
		res.Header.Set("Last-Modified", stat.ModTime().UTC().Format(TimeFormat))
	}
	// Set Age header to indicate how old the cached response is (in seconds)
	res.Header.Set("Age", fmt.Sprintf("%d", int(age.Seconds())))
	return res, err
}

func (fs *fileStorage) store(key string, res *http.Response) error {
	cacheFile := fs.filePath(key)

	fs.mu.Lock()
	defer fs.mu.Unlock()

	err := os.MkdirAll(filepath.Dir(cacheFile), 0755)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(cacheFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	var origBody io.ReadCloser
	if res.Body != nil {
		origBody, res.Body = copyStream(res.Body)
		defer res.Body.Close()
	}
	err = res.Write(f)
	if origBody != nil {
		res.Body = origBody
	}
	return err
}
