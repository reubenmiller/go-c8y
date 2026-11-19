package core

import (
	"io"
	"mime"
	"os"
	"path/filepath"
)

type UploadFileOptions struct {
	FilePath    string
	Reader      io.Reader
	Name        string
	ContentType string

	// Properties are extra fields merged into the multipart "object" metadata
	// (e.g. custom fragments on a binary). An explicit Name/ContentType takes
	// precedence over a "name"/"type" supplied here.
	Properties map[string]any

	// ObjectIndent, when non-empty, pretty-prints the "object" metadata part with
	// this indent string (e.g. "   "). Empty keeps the compact single-line form.
	// The CLI sets it for a template-built object so the dry-run multipart shows
	// the same indented JSON v1 produced, while a plain --data object stays
	// compact.
	ObjectIndent string
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

func (s *UploadFileOptions) Close() error {
	if reader, ok := s.Reader.(*os.File); ok {
		return reader.Close()
	}
	return nil
}

func NewUploadFileOptions(filePath string, contentType ...string) (*UploadFileOptions, error) {
	opt := &UploadFileOptions{
		Name: filepath.Base(filePath),
	}
	if len(contentType) > 0 {
		opt.ContentType = contentType[0]
	} else {
		opt.ContentType = mime.TypeByExtension(filepath.Ext(filePath))
	}
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	opt.Reader = file
	return opt, nil
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
