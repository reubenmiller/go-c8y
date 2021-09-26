package c8y

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

type Cachable func(*http.Request) bool

func NewCachedClient(httpClient *http.Client, cacheDir string, cacheTTL time.Duration, isCachable Cachable, keyOptions KeyOptions) *http.Client {
	if cacheDir == "" {
		cacheDir = filepath.Join(os.TempDir(), "go-c8y-cache")
	}
	if isCachable == nil {
		isCachable = isCacheableRequest
	}
	return &http.Client{
		Transport: CacheResponse(cacheTTL, cacheDir, isCachable, keyOptions)(httpClient.Transport),
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
	return res.StatusCode < 500 && res.StatusCode != 403
}

// CacheResponse produces a RoundTripper that caches HTTP responses to disk for a specified amount of time
func CacheResponse(ttl time.Duration, dir string, isCachable Cachable, keyOptions KeyOptions) ClientOption {
	fs := fileStorage{
		dir: dir,
		ttl: ttl,
		mu:  &sync.RWMutex{},
	}

	return func(tr http.RoundTripper) http.RoundTripper {
		return &funcTripper{roundTrip: func(req *http.Request) (*http.Response, error) {

			if !isCachable(req) {
				return tr.RoundTrip(req)
			}

			key, keyErr := cacheKey(req, keyOptions)
			if keyErr == nil {
				if res, err := fs.read(key); err == nil {
					res.Request = req
					return res, nil
				}
			}

			res, err := tr.RoundTrip(req)
			if err == nil && keyErr == nil && isCacheableResponse(res) {
				_ = fs.store(key, res)
			}
			return res, err
		}}
	}
}

func copyStream(r io.ReadCloser) (io.ReadCloser, io.ReadCloser) {
	b := &bytes.Buffer{}
	nr := io.TeeReader(r, b)
	return ioutil.NopCloser(b), &readCloser{
		Reader: nr,
		Closer: r,
	}
}

type readCloser struct {
	io.Reader
	io.Closer
}

// KeyOptions Cache key generation options
type KeyOptions struct {
	// ExcludeAuth excludes Authorization header value
	ExcludeAuth bool

	// ExcludeHost excludes Host from the full URL value
	ExcludeHost bool
}

func cacheKey(req *http.Request, opt KeyOptions) (string, error) {
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
		if _, err := io.Copy(h, bodyCopy); err != nil {
			return "", err
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

	Logger.Infof("Using cached response. file: %s, age: %s, ttl: %s", cacheFile, age, fs.ttl)

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
