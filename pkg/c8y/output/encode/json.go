// Package encode provides streaming renderers for the output pipeline:
// JSON array, NDJSON and CSV/TSV. All renderers write incrementally per
// item and hold no more than one document plus an I/O buffer in memory.
package encode

import (
	"bufio"
	"io"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

const writeBufferSize = 64 * 1024

// NDJSON renders one compact JSON document per line.
type NDJSON struct {
	w *bufio.Writer
}

func NewNDJSON(w io.Writer) *NDJSON {
	return &NDJSON{w: bufio.NewWriterSize(w, writeBufferSize)}
}

func (e *NDJSON) Write(doc jsondoc.JSONDoc) error {
	if _, err := e.w.Write(doc.Raw()); err != nil {
		return err
	}
	return e.w.WriteByte('\n')
}

func (e *NDJSON) Close() error {
	return e.w.Flush()
}

// JSONArray renders all documents as a single JSON array, written
// incrementally as items arrive.
type JSONArray struct {
	w     *bufio.Writer
	wrote bool
}

func NewJSONArray(w io.Writer) *JSONArray {
	return &JSONArray{w: bufio.NewWriterSize(w, writeBufferSize)}
}

func (e *JSONArray) Write(doc jsondoc.JSONDoc) error {
	var err error
	if e.wrote {
		err = e.w.WriteByte(',')
	} else {
		err = e.w.WriteByte('[')
		e.wrote = true
	}
	if err != nil {
		return err
	}
	_, err = e.w.Write(doc.Raw())
	return err
}

func (e *JSONArray) Close() error {
	if !e.wrote {
		if _, err := e.w.WriteString("[]"); err != nil {
			return err
		}
	} else if err := e.w.WriteByte(']'); err != nil {
		return err
	}
	if err := e.w.WriteByte('\n'); err != nil {
		return err
	}
	return e.w.Flush()
}
