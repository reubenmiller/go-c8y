package core

import (
	"io"
	"os"
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

	// Progress, when set, is called with the number of file bytes uploaded so
	// far and the file size (-1 when the size of a Reader is not known). It
	// counts the file content only, not the multipart envelope. It is called
	// from the goroutine that sends the request, and is also called when the
	// body is read for a dry run.
	Progress ProgressFunc
}

// Check if a read is defined or not. Either the FilePath or Reader must be set
func (s *UploadFileOptions) IsZero() bool {
	return s.FilePath == "" && s.Reader == nil
}

func (s *UploadFileOptions) IsSet() bool {
	return !s.IsZero()
}

func (s *UploadFileOptions) GetReader() any {
	r := s.Reader
	if s.FilePath != "" {
		r = SafeOpenFile(s.FilePath)
	}
	if s.Progress != nil && r != nil {
		if f, ok := r.(*os.File); ok {
			return NewProgressReader(f, ReaderSize(f), s.Progress)
		}
		if _, isErr := r.(ReaderError); !isErr {
			return NewProgressReader(r, ReaderSize(r), s.Progress)
		}
	}
	return r
}

func (s *UploadFileOptions) Close() error {
	if reader, ok := s.Reader.(*os.File); ok {
		return reader.Close()
	}
	return nil
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
