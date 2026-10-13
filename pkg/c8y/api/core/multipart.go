package core

import (
	"encoding/json"
	"io"
	"maps"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"resty.dev/v3"
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

func selectFirstNonEmptyValue(contentType ...string) string {
	for _, v := range contentType {
		if v != "" {
			return v
		}
	}
	return ""
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
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

func NewMultiPartFileFields(opt UploadFileOptions) []*resty.MultipartField {
	obj := make(map[string]any)
	// Caller-supplied custom properties form the base; the explicit name/type
	// (flags) win, then fall back to a "name"/"type" in the properties, then to
	// the file name / detected MIME type.
	maps.Copy(obj, opt.Properties)
	name := selectFirstNonEmptyValue(opt.Name, asString(obj["name"]), filepath.Base(opt.FilePath))
	obj["name"] = name
	// The object metadata "type" prefers the (overridable) name's extension, then
	// the source file's extension, then sniffed content — keeping any
	// "; charset=..." parameter. `--name myCustomName.json` on a config.ini yields
	// application/json; `--name override` on data.csv keeps text/csv from the file.
	obj["type"] = selectFirstNonEmptyValue(opt.ContentType, asString(obj["type"]), detectUploadType(name, opt.FilePath, true), "application/octet-stream")
	var objB []byte
	if opt.ObjectIndent != "" {
		objB, _ = json.MarshalIndent(obj, "", opt.ObjectIndent)
	} else {
		objB, _ = json.Marshal(obj)
	}

	// The file part's own Content-Type comes from the upload name's extension or,
	// when unknown, the sniffed content (NOT the source file's extension), with
	// the charset parameter stripped — matching v1's detectContentType. So
	// `--name override` on data.csv sends a sniffed "text/plain" file part while
	// the object metadata keeps "text/csv; charset=utf-8".
	filePartType := stripMediaParams(selectFirstNonEmptyValue(opt.ContentType, detectUploadType(name, opt.FilePath, false), "application/octet-stream"))

	fields := make([]*resty.MultipartField, 0, 2)
	// v1 ordered the parts file-then-object; the curl/markdown dry-run renderers
	// iterate this slice in order, so emit "file" first to match the v1 output.
	fields = append(fields, &resty.MultipartField{
		Name:        "file",
		FileName:    name,
		FilePath:    opt.FilePath,
		Reader:      opt.Reader,
		ContentType: filePartType,
	})
	// The object part is a plain form field (resty Values), not a Reader part, so
	// resty writes it via CreateFormField — Content-Disposition only, no
	// Content-Type and no content sniffing — matching v1.
	fields = append(fields, &resty.MultipartField{
		Name:   "object",
		Values: []string{string(objB)},
	})

	return fields
}

// detectUploadType resolves a file's MIME type from the (overridable) upload
// name's extension, falling back to content sniffing the source file when the
// extension is unknown — matching v1's detectContentType (e.g. a config.ini or
// an extension-less file is sniffed to text/plain rather than left as the
// generic octet-stream). Returns "" when nothing can be determined.
func detectUploadType(name, filePath string, useFileExt bool) string {
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
	}
	// The object metadata (useFileExt) falls back to the source file's extension
	// before sniffing; the file part header skips straight to sniffing.
	if useFileExt {
		if ct := mime.TypeByExtension(filepath.Ext(filePath)); ct != "" {
			return ct
		}
	}
	if filePath != "" {
		if f, err := os.Open(filePath); err == nil {
			defer f.Close()
			buf := make([]byte, 512)
			n, _ := f.Read(buf)
			return http.DetectContentType(buf[:n])
		}
	}
	return ""
}

// stripMediaParams returns the bare media type, dropping any "; param=..."
// suffix (e.g. "text/csv; charset=utf-8" -> "text/csv").
func stripMediaParams(mediaType string) string {
	if i := strings.IndexByte(mediaType, ';'); i >= 0 {
		return strings.TrimSpace(mediaType[:i])
	}
	return mediaType
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
