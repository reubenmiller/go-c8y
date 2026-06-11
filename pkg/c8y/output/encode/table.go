package encode

import (
	"bufio"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// RowWriter renders resolved table rows. Implementations own the
// presentation (borders, colors, truncation); the Table engine owns column
// resolution, width sampling and cell extraction.
//
// WriteHeader is called exactly once, before any row, with the resolved
// column names and the column widths measured from the sampled rows
// (rune count, capped at MaxColumnWidth). Rows arriving after the sample
// window may contain cells wider than the sampled widths — implementations
// decide whether to truncate or let the layout grow.
type RowWriter interface {
	WriteHeader(columns []string, widths []int) error
	WriteRow(cells []string) error
	Close() error
}

// TableOptions configures the streaming table engine.
type TableOptions struct {
	// Columns are the gjson paths rendered as table columns. When empty,
	// columns are derived from the leaf paths of the first document.
	Columns []string
	// SampleSize is the number of rows buffered to determine column widths
	// before output begins, keeping memory usage O(SampleSize) regardless
	// of stream length. Defaults to 50.
	SampleSize int
	// MaxColumnWidth caps each column's width. Defaults to 60.
	MaxColumnWidth int
}

// Table is the streaming table engine: it resolves columns, buffers the
// first SampleSize rows to measure column widths, then hands the header and
// rows to a RowWriter as they arrive.
type Table struct {
	rw      RowWriter
	opts    TableOptions
	columns []string
	sample  [][]string
	flushed bool
}

// NewTable returns a table renderer writing plain aligned text to w
// (two-space column separator, dashed divider, cells truncated to the
// sampled widths with an ellipsis marker).
func NewTable(w io.Writer, opts TableOptions) *Table {
	return NewTableWithWriter(&plainRowWriter{w: bufio.NewWriterSize(w, writeBufferSize)}, opts)
}

// NewTableWithWriter returns a table renderer that delegates presentation to
// a custom RowWriter (e.g. an adapter for a rich table rendering library).
func NewTableWithWriter(rw RowWriter, opts TableOptions) *Table {
	if opts.SampleSize <= 0 {
		opts.SampleSize = 50
	}
	if opts.MaxColumnWidth <= 0 {
		opts.MaxColumnWidth = 60
	}
	return &Table{
		rw:      rw,
		opts:    opts,
		columns: opts.Columns,
	}
}

func (e *Table) Write(doc jsondoc.JSONDoc) error {
	root := doc.Get()
	if e.columns == nil {
		e.columns = leafPaths(root)
	}

	row := make([]string, len(e.columns))
	for i, col := range e.columns {
		row[i] = sanitizeCell(root.Get(col).String())
	}

	if !e.flushed {
		e.sample = append(e.sample, row)
		if len(e.sample) >= e.opts.SampleSize {
			return e.flushSample()
		}
		return nil
	}
	return e.rw.WriteRow(row)
}

func (e *Table) Close() error {
	if !e.flushed {
		if err := e.flushSample(); err != nil {
			return err
		}
	}
	return e.rw.Close()
}

// flushSample fixes the column widths from the buffered rows, then emits the
// header and the buffered rows to the RowWriter.
func (e *Table) flushSample() error {
	e.flushed = true

	widths := make([]int, len(e.columns))
	for i, col := range e.columns {
		widths[i] = utf8.RuneCountInString(col)
	}
	for _, row := range e.sample {
		for i, cell := range row {
			if w := utf8.RuneCountInString(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}
	for i := range widths {
		widths[i] = min(widths[i], e.opts.MaxColumnWidth)
	}

	if err := e.rw.WriteHeader(e.columns, widths); err != nil {
		return err
	}

	rows := e.sample
	e.sample = nil
	for _, row := range rows {
		if err := e.rw.WriteRow(row); err != nil {
			return err
		}
	}
	return nil
}

// plainRowWriter is the built-in dependency-free presentation: aligned
// columns separated by two spaces, a dashed divider under the header, and
// cells truncated to the sampled widths with an ellipsis marker.
type plainRowWriter struct {
	w      *bufio.Writer
	widths []int
}

func (p *plainRowWriter) WriteHeader(columns []string, widths []int) error {
	p.widths = widths
	if len(columns) == 0 {
		return nil
	}
	if err := p.WriteRow(columns); err != nil {
		return err
	}
	divider := make([]string, len(columns))
	for i, w := range widths {
		divider[i] = strings.Repeat("-", w)
	}
	return p.WriteRow(divider)
}

func (p *plainRowWriter) WriteRow(cells []string) error {
	for i, cell := range cells {
		if i > 0 {
			if _, err := p.w.WriteString("  "); err != nil {
				return err
			}
		}
		width := p.widths[i]
		n := utf8.RuneCountInString(cell)
		if n > width {
			cell = truncateRunes(cell, width)
			n = width
		}
		if _, err := p.w.WriteString(cell); err != nil {
			return err
		}
		// Pad all but the last column.
		if i < len(cells)-1 {
			for ; n < width; n++ {
				if err := p.w.WriteByte(' '); err != nil {
					return err
				}
			}
		}
	}
	return p.w.WriteByte('\n')
}

func (p *plainRowWriter) Close() error {
	return p.w.Flush()
}

// sanitizeCell makes a value safe for single-line table output.
func sanitizeCell(s string) string {
	if strings.ContainsAny(s, "\r\n\t") {
		s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ", "\t", " ").Replace(s)
	}
	return s
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	count := 0
	for i := range s {
		count++
		if count > n-1 {
			// Reserve the last rune for the ellipsis marker.
			return s[:i] + "…"
		}
	}
	return s
}
