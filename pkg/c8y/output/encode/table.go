package encode

import (
	"bufio"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
)

// TableOptions configures the streaming table renderer.
type TableOptions struct {
	// Columns are the gjson paths rendered as table columns. When empty,
	// columns are derived from the leaf paths of the first document.
	Columns []string
	// SampleSize is the number of rows buffered to determine column widths
	// before output begins. Later rows are truncated to the sampled widths,
	// keeping memory usage O(SampleSize) regardless of stream length.
	// Defaults to 50.
	SampleSize int
	// MaxColumnWidth caps each column's width. Defaults to 60.
	MaxColumnWidth int
}

// Table renders documents as an aligned text table while streaming: the
// first SampleSize rows are buffered to size the columns, then everything
// is flushed and subsequent rows are written as they arrive.
type Table struct {
	w       *bufio.Writer
	opts    TableOptions
	columns []string
	widths  []int
	sample  [][]string
	flushed bool
}

func NewTable(w io.Writer, opts TableOptions) *Table {
	if opts.SampleSize <= 0 {
		opts.SampleSize = 50
	}
	if opts.MaxColumnWidth <= 0 {
		opts.MaxColumnWidth = 60
	}
	return &Table{
		w:       bufio.NewWriterSize(w, writeBufferSize),
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
	return e.writeRow(row)
}

func (e *Table) Close() error {
	if !e.flushed {
		if err := e.flushSample(); err != nil {
			return err
		}
	}
	return e.w.Flush()
}

// flushSample fixes the column widths from the buffered rows, then writes
// the header and the buffered rows.
func (e *Table) flushSample() error {
	e.flushed = true
	e.widths = make([]int, len(e.columns))
	for i, col := range e.columns {
		e.widths[i] = utf8.RuneCountInString(col)
	}
	for _, row := range e.sample {
		for i, cell := range row {
			if w := utf8.RuneCountInString(cell); w > e.widths[i] {
				e.widths[i] = w
			}
		}
	}
	for i := range e.widths {
		e.widths[i] = min(e.widths[i], e.opts.MaxColumnWidth)
	}

	if len(e.columns) > 0 {
		if err := e.writeRow(e.columns); err != nil {
			return err
		}
		divider := make([]string, len(e.columns))
		for i, w := range e.widths {
			divider[i] = strings.Repeat("-", w)
		}
		if err := e.writeRow(divider); err != nil {
			return err
		}
	}

	rows := e.sample
	e.sample = nil
	for _, row := range rows {
		if err := e.writeRow(row); err != nil {
			return err
		}
	}
	return nil
}

func (e *Table) writeRow(row []string) error {
	for i, cell := range row {
		if i > 0 {
			if _, err := e.w.WriteString("  "); err != nil {
				return err
			}
		}
		width := e.widths[i]
		n := utf8.RuneCountInString(cell)
		if n > width {
			cell = truncateRunes(cell, width)
			n = width
		}
		if _, err := e.w.WriteString(cell); err != nil {
			return err
		}
		// Pad all but the last column.
		if i < len(row)-1 {
			for ; n < width; n++ {
				if err := e.w.WriteByte(' '); err != nil {
					return err
				}
			}
		}
	}
	return e.w.WriteByte('\n')
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
