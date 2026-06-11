package encode

import (
	"encoding/csv"
	"io"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// CSVOptions configures the CSV/TSV renderer.
type CSVOptions struct {
	// Delimiter between fields. Defaults to ',' (use NewTSV for '\t').
	Delimiter rune
	// Header writes the column names as the first row.
	Header bool
	// Columns are the gjson paths to extract per document. When empty, the
	// columns are derived from the leaf paths of the first document and then
	// fixed for the remainder of the stream.
	Columns []string
}

// CSV renders documents as delimiter-separated rows, streaming one row per
// item. Nested values are addressed by dotted column paths.
type CSV struct {
	w          *csv.Writer
	opts       CSVOptions
	columns    []string
	row        []string
	headerDone bool
}

func NewCSV(w io.Writer, opts CSVOptions) *CSV {
	cw := csv.NewWriter(w)
	if opts.Delimiter != 0 {
		cw.Comma = opts.Delimiter
	}
	return &CSV{w: cw, opts: opts, columns: opts.Columns}
}

// NewTSV returns a CSV renderer with a tab delimiter.
func NewTSV(w io.Writer, opts CSVOptions) *CSV {
	opts.Delimiter = '\t'
	return NewCSV(w, opts)
}

func (e *CSV) Write(doc jsondoc.JSONDoc) error {
	root := doc.Get()
	if e.columns == nil {
		e.columns = leafPaths(root)
	}
	if e.row == nil {
		e.row = make([]string, len(e.columns))
	}
	if e.opts.Header && !e.headerDone {
		e.headerDone = true
		if err := e.w.Write(e.columns); err != nil {
			return err
		}
	}
	for i, col := range e.columns {
		e.row[i] = root.Get(col).String()
	}
	return e.w.Write(e.row)
}

func (e *CSV) Close() error {
	e.w.Flush()
	return e.w.Error()
}

// leafPaths returns the dotted paths of all leaves of a document in document
// order. Objects are traversed; arrays and scalars are leaves.
func leafPaths(res gjson.Result) []string {
	var paths []string
	var walk func(res gjson.Result, prefix string)
	walk = func(res gjson.Result, prefix string) {
		res.ForEach(func(k, v gjson.Result) bool {
			path := k.String()
			if prefix != "" {
				path = prefix + "." + path
			}
			if v.IsObject() {
				walk(v, path)
			} else {
				paths = append(paths, path)
			}
			return true
		})
	}
	walk(res, "")
	return paths
}
