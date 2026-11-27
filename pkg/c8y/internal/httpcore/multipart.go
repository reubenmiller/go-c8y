package httpcore

import (
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"maps"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

func NewMultiPartFileFields(opt core.UploadFileOptions) []*resty.MultipartField {
	obj := make(map[string]any)
	// Caller-supplied custom properties form the base; the explicit name/type
	// (flags) win, then fall back to a "name"/"type" in the properties, then to
	// the file name / detected MIME type.
	maps.Copy(obj, opt.Properties)
	name := selectFirstNonEmptyValue(opt.Name, asString(obj["name"]), filepath.Base(opt.FilePath))
	obj["name"] = name
	// The object metadata "type" is the explicit type, then a "type" of the
	// properties, then the type of the source file's extension; a file with no
	// or an unknown extension is application/octet-stream (its content is not
	// sniffed). This is what go-c8y-cli v2.59.0 sent: `--name x.json` on a
	// config.ini keeps text/plain, and a file named "nofile" is octet-stream.
	// An upload from a Reader (no FilePath) uses the extension of its name.
	typeSource := opt.FilePath
	if typeSource == "" {
		typeSource = name
	}
	obj["type"] = selectFirstNonEmptyValue(opt.ContentType, asString(obj["type"]), mime.TypeByExtension(filepath.Ext(typeSource)), "application/octet-stream")
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
	filePartType := stripMediaParams(selectFirstNonEmptyValue(opt.ContentType, detectUploadType(name, opt.FilePath), "application/octet-stream"))

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
	withUploadProgress(fields[0], opt)
	// The object part is a plain form field (resty Values), not a Reader part, so
	// resty writes it via CreateFormField — Content-Disposition only, no
	// Content-Type and no content sniffing — matching v1.
	fields = append(fields, &resty.MultipartField{
		Name:   "object",
		Values: []string{string(objB)},
	})

	return fields
}

func NewMultiPartFile(opt core.UploadFileOptions) []*resty.MultipartField {
	fields := make([]*resty.MultipartField, 0, 2)
	fields = append(fields, &resty.MultipartField{
		Name:     "file",
		FilePath: opt.FilePath,
		Reader:   opt.Reader,
	})
	withUploadProgress(fields[0], opt)
	return fields
}

// withUploadProgress reports the file part's progress to opt.Progress, using
// the HTTP client's own multipart callback (which also handles retries).
func withUploadProgress(f *resty.MultipartField, opt core.UploadFileOptions) {
	if opt.Progress == nil {
		return
	}
	if f.Reader != nil {
		f.FileSize = core.ReaderSize(f.Reader)
	}
	fn := opt.Progress
	f.ProgressCallback = func(p resty.MultipartFieldProgress) {
		total := p.FileSize
		if total == 0 {
			total = -1
		}
		fn(p.Written, total)
	}
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

// detectUploadType resolves the MIME type of the file part from the (overridable)
// upload name's extension, falling back to content sniffing the source file
// when the extension is unknown — matching v1's detectContentType (e.g. a
// config.ini or an extension-less file is sniffed to text/plain rather than
// left as the generic octet-stream). Returns "" when nothing can be determined.
func detectUploadType(name, filePath string) string {
	if ct := mime.TypeByExtension(filepath.Ext(name)); ct != "" {
		return ct
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
