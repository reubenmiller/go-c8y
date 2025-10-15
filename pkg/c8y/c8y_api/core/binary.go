package core

import (
	"crypto/sha256"
	"io"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"

	"resty.dev/v3"
)

var (
	hdrContentDisposition = http.CanonicalHeaderKey("Content-Disposition")
)

type BinaryResponse struct {
	Response *resty.Response
}

func (b *BinaryResponse) Reader() io.ReadCloser {
	return b.Response.Body
}

func (b *BinaryResponse) Close() error {
	return b.Response.Body.Close()
}

func (b *BinaryResponse) FileName() (v string) {
	if b.Response != nil {
		v = binaryName(b.Response.RawResponse)
	}
	return
}

// URL of the resolved binary
func (b *BinaryResponse) URL() (v string) {
	if b.Response.RawResponse != nil && b.Response.RawResponse.Request != nil {
		v = b.Response.RawResponse.Request.URL.String()
	}
	return
}

func (b *BinaryResponse) Size() int64 {
	return contentLength(b.Response.RawResponse)
}

func NewBinaryResponse(r *resty.Response) *BinaryResponse {
	resp := &BinaryResponse{
		Response: r,
	}
	return resp
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
