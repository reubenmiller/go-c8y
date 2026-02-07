package core

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"os"
	"path/filepath"

	"resty.dev/v3"
)

type UploadFileOptions struct {
	FilePath    string
	Reader      io.Reader
	Name        string
	ContentType string
}

// Check if a read is defined or not. Either the FilePath or Reader must be set
func (s *UploadFileOptions) IsZero() bool {
	return s.FilePath == "" && s.Reader == nil
}

func (s *UploadFileOptions) IsSet() bool {
	return !s.IsZero()
}

func (s *UploadFileOptions) GetReader() any {
	if s.FilePath != "" {
		return SafeOpenFile(s.FilePath)
	}
	return s.Reader
}

func selectFirstNonEmptyValue(contentType ...string) string {
	for _, v := range contentType {
		if v != "" {
			return v
		}
	}
	return ""
}

func NewMultiPartFileFields(opt UploadFileOptions) []*resty.MultipartField {
	obj := make(map[string]any)
	obj["name"] = selectFirstNonEmptyValue(opt.Name, filepath.Base(opt.FilePath))
	obj["type"] = selectFirstNonEmptyValue(opt.ContentType, mime.TypeByExtension(filepath.Ext(opt.FilePath)), "application/octet-stream")
	objB, _ := json.Marshal(obj)

	fields := make([]*resty.MultipartField, 0, 2)
	fields = append(fields, &resty.MultipartField{
		Name:        "object",
		Reader:      bytes.NewReader(objB),
		ContentType: "application/json",
	})
	fields = append(fields, &resty.MultipartField{
		Name:        "file",
		FileName:    obj["name"].(string),
		FilePath:    opt.FilePath,
		Reader:      opt.Reader,
		ContentType: obj["type"].(string),
	})

	return fields
}

func NewMultiPartFile(opt UploadFileOptions) []*resty.MultipartField {
	fields := make([]*resty.MultipartField, 0, 2)
	fields = append(fields, &resty.MultipartField{
		Name:     "file",
		FilePath: opt.FilePath,
		Reader:   opt.Reader,
	})
	return fields
}

type ReaderError struct {
	Err error
}

func (r ReaderError) Read(p []byte) (n int, err error) {
	return 0, r.Err
}

func (r ReaderError) Close() error {
	return nil
}

func SafeOpenFile(path string) io.ReadCloser {
	file, err := os.Open(path)
	if err != nil {
		return ReaderError{Err: err}
	}
	return file
}
