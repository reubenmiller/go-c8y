package encode

import (
	"bufio"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/tidwall/gjson"
)

// Align controls horizontal cell alignment in rendered tables.
type Align int8

const (
	AlignLeft Align = iota
	AlignRight
)

// Column describes a resolved table column: its name, the width measured
// from the sampled rows, and the alignment derived from the sampled values.
type Column struct {
	Name  string
	Width int
	Align Align
}

// RowWriter renders resolved table rows. Implementations own the
// presentation (borders, colors); the Table engine owns column resolution,
// width sampling and cell extraction.
//
// WriteHeader is called exactly once, before any row. Cells passed to
// WriteRow are already fitted to the column widths when a CellTransform is
// configured; multi-line cells use embedded newlines.
type RowWriter interface {
	WriteHeader(columns []Column) error
	WriteRow(cells []string) error
	Close() error
}

// CellFormatter extracts and formats a column's cell value from a document,
// returning the display value and its alignment.
type CellFormatter func(doc jsondoc.JSONDoc, column string) (cell string, align Align)

// CellTransform fits a formatted cell value to the resolved column width,
// e.g. by truncating or wrapping into multiple lines separated by newlines.
// It is also applied during sampling (with MaxColumnWidth) so measured
// widths reflect the displayed values.
type CellTransform func(cell string, width int) string

// TableOptions configures the streaming table engine.
type TableOptions struct {
	// Columns are the gjson paths rendered as table columns.
	Columns []string
	// ColumnResolver lazily resolves the column names when Columns is empty.
	// It is called when the sample window is flushed; when it returns no
	// columns, they are derived from the leaf paths of the first document.
	ColumnResolver func() []string
	// SampleSize is the number of rows buffered to determine column widths
	// before output begins, keeping memory usage O(SampleSize) regardless
	// of stream length. Defaults to 50.
	SampleSize int
	// MinColumnWidth is the minimum content width of a column.
	MinColumnWidth int
	// MinEmptyColumnWidth is the minimum width used instead of
	// MinColumnWidth when none of the sampled rows have a value for the
	// column. Zero falls back to MinColumnWidth.
	MinEmptyColumnWidth int
	// MaxColumnWidth caps each column's width. Defaults to 60.
	MaxColumnWidth int
	// ColumnPadding is added to each column's measured content width.
	ColumnPadding int
	// MaxTableWidth limits the total table width: trailing columns that do
	// not fit are dropped (the last kept column may be shrunk to the
	// remaining space). Zero means unlimited.
	MaxTableWidth int
	// Formatter extracts and formats cell values. The default reads the
	// column path, aligning numbers right and everything else left.
	Formatter CellFormatter
	// Transform fits formatted cells to the resolved column width. The
	// default leaves values untouched (renderers may still truncate).
	Transform CellTransform
}

// Table is the streaming table engine: it buffers the first SampleSize
// documents, resolves columns, alignments and widths from the sample, then
// hands the header and rows to a RowWriter as they arrive.
type Table struct {
	rw      RowWriter
	opts    TableOptions
	sample  []jsondoc.JSONDoc
	columns []Column
	flushed bool
	closed  bool
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
	if opts.Formatter == nil {
		opts.Formatter = defaultCellFormatter
	}
	return &Table{rw: rw, opts: opts}
}

func defaultCellFormatter(doc jsondoc.JSONDoc, column string) (string, Align) {
	node := doc.Get(column)
	if node.Type == gjson.Number {
		return node.String(), AlignRight
	}
	return sanitizeCell(node.String()), AlignLeft
}

// Write renders a single document as a table row. Documents written while
// the sample window is open are buffered; the underlying bytes must remain
// valid until the table is flushed.
func (e *Table) Write(doc jsondoc.JSONDoc) error {
	if !e.flushed {
		e.sample = append(e.sample, doc)
		if len(e.sample) >= e.opts.SampleSize {
			return e.Flush()
		}
		return nil
	}
	return e.writeRow(doc)
}

// Flush resolves the columns from the sampled documents and renders the
// header and any buffered rows. It is called automatically once the sample
// window fills or the table is closed; call it earlier to bound the latency
// of streamed output (e.g. realtime subscriptions). Safe to call repeatedly.
func (e *Table) Flush() error {
	if e.flushed {
		return nil
	}
	e.flushed = true

	names := e.resolveColumnNames()
	e.columns = e.resolveColumns(names, e.sampleCells(names))

	if err := e.rw.WriteHeader(e.columns); err != nil {
		return err
	}

	sample := e.sample
	e.sample = nil
	for _, doc := range sample {
		if err := e.writeRow(doc); err != nil {
			return err
		}
	}
	return nil
}

// sampledCell is a measured cell from the sample window.
type sampledCell struct {
	value string
	align Align
}

// sampleCells extracts and measures the sampled rows. Cells are measured
// after applying the transform at the column width cap so wrapped/truncated
// display values determine the widths.
func (e *Table) sampleCells(names []string) [][]sampledCell {
	rows := make([][]sampledCell, 0, len(e.sample))
	for _, doc := range e.sample {
		row := make([]sampledCell, len(names))
		for i, name := range names {
			cell, align := e.opts.Formatter(doc, name)
			row[i] = sampledCell{value: cell, align: align}
		}
		rows = append(rows, row)
	}
	return rows
}

// resolveColumns resolves column widths and alignments, dropping trailing
// columns that do not fit within MaxTableWidth.
func (e *Table) resolveColumns(names []string, rows [][]sampledCell) []Column {
	const separatorOverhead = 3
	const tableEndBuffer = 3
	used := 0
	columns := make([]Column, 0, len(names))
	for i, name := range names {
		cellWidth, hasValue, align := e.measureColumn(rows, i)

		minWidth := e.opts.MinColumnWidth
		if !hasValue && e.opts.MinEmptyColumnWidth > 0 {
			minWidth = e.opts.MinEmptyColumnWidth
		}

		paddedWidth := cellWidth + e.opts.ColumnPadding
		if paddedWidth > e.opts.MaxColumnWidth {
			paddedWidth = e.opts.MaxColumnWidth
		}

		colWidth := max(paddedWidth, minWidth+e.opts.ColumnPadding, displayWidth(name))

		if e.opts.MaxTableWidth > 0 && used+colWidth+separatorOverhead > e.opts.MaxTableWidth {
			leftOver := e.opts.MaxTableWidth - used - separatorOverhead - tableEndBuffer
			if leftOver > minWidth {
				columns = append(columns, Column{Name: name, Width: leftOver, Align: align})
			}
			break
		}
		columns = append(columns, Column{Name: name, Width: colWidth, Align: align})
		used += colWidth + separatorOverhead
	}
	return columns
}

// measureColumn returns the widest display value of a column across the
// sampled rows, whether any row has a value, and the alignment of the first
// non-empty cell.
func (e *Table) measureColumn(rows [][]sampledCell, i int) (cellWidth int, hasValue bool, align Align) {
	align = AlignLeft
	alignSet := false
	for _, row := range rows {
		cell := row[i].value
		if cell != "" {
			hasValue = true
			if !alignSet {
				align = row[i].align
				alignSet = true
			}
		}
		if w := displayWidth(e.transform(cell, e.opts.MaxColumnWidth)); w > cellWidth {
			cellWidth = w
		}
	}
	return cellWidth, hasValue, align
}

// Close flushes any sampled rows and closes the underlying writer.
func (e *Table) Close() error {
	if e.closed {
		return nil
	}
	e.closed = true
	if err := e.Flush(); err != nil {
		return err
	}
	return e.rw.Close()
}

func (e *Table) resolveColumnNames() []string {
	if len(e.opts.Columns) > 0 {
		return e.opts.Columns
	}
	if e.opts.ColumnResolver != nil {
		if names := e.opts.ColumnResolver(); len(names) > 0 {
			return names
		}
	}
	if len(e.sample) > 0 {
		return leafPaths(e.sample[0].Get())
	}
	return nil
}

func (e *Table) writeRow(doc jsondoc.JSONDoc) error {
	cells := make([]string, len(e.columns))
	for i, col := range e.columns {
		cell, _ := e.opts.Formatter(doc, col.Name)
		cells[i] = e.transform(cell, col.Width)
	}
	return e.rw.WriteRow(cells)
}

func (e *Table) transform(cell string, width int) string {
	if e.opts.Transform == nil {
		return cell
	}
	return e.opts.Transform(cell, width)
}

// displayWidth is the rendered width of a cell value, using the widest line
// for multi-line (e.g. wrapped) values.
func displayWidth(s string) int {
	if !strings.Contains(s, "\n") {
		return runewidth.StringWidth(s)
	}
	width := 0
	for _, line := range strings.Split(s, "\n") {
		if w := runewidth.StringWidth(line); w > width {
			width = w
		}
	}
	return width
}

// plainRowWriter is the built-in dependency-free presentation: aligned
// columns separated by two spaces, a dashed divider under the header, and
// cells truncated to the column widths with an ellipsis marker.
type plainRowWriter struct {
	w       *bufio.Writer
	columns []Column
}

func (p *plainRowWriter) WriteHeader(columns []Column) error {
	p.columns = columns
	if len(columns) == 0 {
		return nil
	}
	names := make([]string, len(columns))
	divider := make([]string, len(columns))
	for i, col := range columns {
		names[i] = col.Name
		divider[i] = strings.Repeat("-", col.Width)
	}
	if err := p.WriteRow(names); err != nil {
		return err
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
		if err := p.writeCell(cell, i, i == len(cells)-1); err != nil {
			return err
		}
	}
	return p.w.WriteByte('\n')
}

// writeCell writes a single cell truncated and padded to its column width.
func (p *plainRowWriter) writeCell(cell string, i int, last bool) error {
	// Multi-line cells are not supported by the plain writer.
	if strings.Contains(cell, "\n") {
		cell = strings.ReplaceAll(cell, "\n", " ")
	}
	width := p.columns[i].Width
	n := runewidth.StringWidth(cell)
	if n > width {
		cell = truncateRunes(cell, width)
		n = width
	}
	pad := width - n
	alignRight := p.columns[i].Align == AlignRight
	if alignRight {
		if err := p.writePadding(pad); err != nil {
			return err
		}
	}
	if _, err := p.w.WriteString(cell); err != nil {
		return err
	}
	// Pad all but the last column.
	if !alignRight && !last {
		return p.writePadding(pad)
	}
	return nil
}

func (p *plainRowWriter) writePadding(pad int) error {
	for ; pad > 0; pad-- {
		if err := p.w.WriteByte(' '); err != nil {
			return err
		}
	}
	return nil
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
