package api

import (
	"encoding/json"
	"io"
	"maps"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"resty.dev/v3"
)

// FormFile is a [RequestOptions.FormData] value which is sent as a file part.
//
// Example:
//
//	FormData: map[string]io.Reader{
//	    "file": &api.FormFile{Reader: f, FileName: "report.csv", ContentType: "text/csv"},
//	}
type FormFile struct {
	io.Reader

	// FileName is the part's file name. It defaults to the field name.
	FileName string

	// ContentType is the part's content type. It defaults to the type of the
	// file name's extension, or else to the type detected from the contents.
	ContentType string
}

// FormField is a [RequestOptions.FormData] value which is sent as a plain
// form field (a part without a file name).
//
// Example:
//
//	FormData: map[string]io.Reader{
//	    "description": &api.FormField{Reader: strings.NewReader("nightly export")},
//	}
type FormField struct {
	io.Reader

	// ContentType is the part's content type. Empty sends the field without
	// one.
	ContentType string
}

// multipartFields returns the multipart parts of form, in field name order.
func multipartFields(form map[string]io.Reader) ([]*resty.MultipartField, error) {
	fields := make([]*resty.MultipartField, 0, len(form))
	for _, name := range slices.Sorted(maps.Keys(form)) {
		field, err := multipartField(name, form[name])
		if err != nil {
			return nil, err
		}
		fields = append(fields, field)
	}
	return fields, nil
}

func multipartField(name string, r io.Reader) (*resty.MultipartField, error) {
	switch v := r.(type) {
	case *FormFile:
		return fileField(name, v.Reader, v.FileName, v.ContentType), nil
	case FormFile:
		return fileField(name, v.Reader, v.FileName, v.ContentType), nil
	case *FormField:
		return textField(name, v.Reader, v.ContentType)
	case FormField:
		return textField(name, v.Reader, v.ContentType)
	case *os.File:
		return fileField(name, v, filepath.Base(v.Name()), ""), nil
	}
	if name == "object" {
		// The binary metadata of the Cumulocity upload APIs
		return textField(name, r, types.MimeTypeApplicationJSON)
	}
	return &resty.MultipartField{
		Name:        name,
		FileName:    name,
		ContentType: types.MimeTypeApplicationOctetStream,
		Reader:      r,
	}, nil
}

func fileField(name string, r io.Reader, fileName, contentType string) *resty.MultipartField {
	if fileName == "" {
		fileName = name
	}
	if contentType == "" {
		// An empty content type is detected from the contents when sent
		contentType = mime.TypeByExtension(filepath.Ext(fileName))
	}
	return &resty.MultipartField{
		Name:        name,
		FileName:    fileName,
		ContentType: contentType,
		Reader:      r,
	}
}

func textField(name string, r io.Reader, contentType string) (*resty.MultipartField, error) {
	if contentType != "" {
		return &resty.MultipartField{Name: name, ContentType: contentType, Reader: r}, nil
	}
	// The HTTP client detects a content type for every part with a reader, so
	// the value is sent as a field value instead
	b, err := io.ReadAll(r)
	if c, ok := r.(io.Closer); ok {
		_ = c.Close()
	}
	if err != nil {
		return nil, err
	}
	return &resty.MultipartField{Name: name, Values: []string{string(b)}}, nil
}

// responseError returns the error of a response with an HTTP error status, in
// the same form as the typed service methods: a *core.Error with the
// Cumulocity error type and message when the body is a Cumulocity error,
// otherwise with the body as the message.
func responseError(resp *resty.Response, httpResp *http.Response, body []byte) *core.Error {
	if len(body) == 0 {
		// The client parses a JSON error body into the response's error result
		if apiErr, ok := resp.ResultError().(*core.APIError); ok && apiErr != nil {
			body, _ = json.Marshal(apiErr)
		}
	}
	err := &core.Error{
		Code:       httpResp.StatusCode,
		Message:    strings.TrimSpace(string(body)),
		MessageRaw: string(body),
		Response:   httpResp,
		Duration:   resp.Duration(),
		ReceivedAt: resp.ReceivedAt(),
	}
	var apiErr core.APIError
	if json.Unmarshal(body, &apiErr) == nil && (apiErr.ErrorType != "" || apiErr.Message != "" || len(apiErr.Messages) > 0) {
		err.Type = apiErr.ErrorType
		err.Message = apiErr.Error()
	}
	if err.Message == "" {
		err.Message = http.StatusText(httpResp.StatusCode)
	}
	return err
}
