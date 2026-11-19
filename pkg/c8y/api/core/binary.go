package core

import (
	"crypto/sha256"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
)

var (
	hdrContentDisposition = http.CanonicalHeaderKey("Content-Disposition")
)

// BinaryResponse is the response of a request that downloads a binary. The
// body is not read by the SDK: read it with Reader and close it with Close.
type BinaryResponse struct {
	// Response is the HTTP response. Its Body is the unread binary content.
	Response *http.Response
}

// Reader returns the unread response body.
func (b *BinaryResponse) Reader() io.ReadCloser {
	if b.Response == nil || b.Response.Body == nil {
		return http.NoBody
	}
	return b.Response.Body
}

// Close closes the response body.
func (b *BinaryResponse) Close() error {
	if b.Response == nil || b.Response.Body == nil {
		return nil
	}
	return b.Response.Body.Close()
}

// FileName returns the file name from the Content-Disposition header, or from
// the request URL when the header is missing.
func (b *BinaryResponse) FileName() (v string) {
	if b.Response != nil {
		v = binaryName(b.Response)
	}
	return
}

// URL returns the URL that the binary was downloaded from.
func (b *BinaryResponse) URL() (v string) {
	if b.Response != nil && b.Response.Request != nil && b.Response.Request.URL != nil {
		v = b.Response.Request.URL.String()
	}
	return
}

// Size returns the Content-Length of the response, or 0 when unknown.
func (b *BinaryResponse) Size() int64 {
	if b.Response == nil {
		return 0
	}
	return contentLength(b.Response)
}

// StatusCode returns the HTTP status code of the response.
func (b *BinaryResponse) StatusCode() int {
	if b.Response == nil {
		return 0
	}
	return b.Response.StatusCode
}

func isStringEmpty(str string) bool {
	return len(strings.TrimSpace(str)) == 0
}

func contentLength(r *http.Response) int64 {
	cntContentLength := r.Header.Get("Content-Length")
	if len(cntContentLength) > 0 {
		length, err := strconv.ParseInt(cntContentLength, 10, 64)
		if err == nil {
			return length
		}
	}
	return int64(0)
}

func binaryName(r *http.Response) (name string) {
	if r == nil {
		return
	}

	cntDispositionValue := r.Header.Get(hdrContentDisposition)
	if len(cntDispositionValue) > 0 {
		if _, params, err := mime.ParseMediaType(cntDispositionValue); err == nil {
			name = params["filename"]
		}
	}

	if r.Request == nil {
		return
	}

	if isStringEmpty(name) {
		if isStringEmpty(r.Request.URL.Path) || r.Request.URL.Path == "/" {
			h := sha256.New()
			if _, err := h.Write([]byte(r.Request.Host)); err == nil {
				name = string(h.Sum(nil))
			}
		} else {
			name = path.Base(r.Request.URL.Path)
		}
	}
	return
}
