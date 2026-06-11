package encode_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/encode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestTableBasic(t *testing.T) {
	body := `[
		{"id": "1", "name": "alpha", "type": "c8y_Linux"},
		{"id": "2", "name": "beta-long-name", "type": "c8y_Windows"}
	]`
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(body), ""),
		encode.NewTable(&buf, encode.TableOptions{Columns: []string{"id", "name", "type"}}))
	require.NoError(t, err)

	assert.Equal(t,
		"id  name            type\n"+
			"--  --------------  -----------\n"+
			"1   alpha           c8y_Linux\n"+
			"2   beta-long-name  c8y_Windows\n",
		buf.String())
}

func TestTableTruncatesAfterSample(t *testing.T) {
	// Rows after the sample window must be truncated to the sampled widths.
	var sb strings.Builder
	sb.WriteString(`[`)
	for i := range 10 {
		if i > 0 {
			sb.WriteByte(',')
		}
		name := "short"
		if i >= 3 {
			name = "this-name-is-much-longer-than-the-sampled-width"
		}
		fmt.Fprintf(&sb, `{"id": "%d", "name": "%s"}`, i, name)
	}
	sb.WriteString(`]`)

	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(sb.String()), ""),
		encode.NewTable(&buf, encode.TableOptions{
			Columns:    []string{"id", "name"},
			SampleSize: 3,
		}))
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 12) // header + divider + 10 rows
	// Sampled width for "name" is len("short")=5; later long values are
	// truncated with an ellipsis marker.
	assert.Equal(t, "0   short", lines[2])
	assert.Equal(t, "3   this…", lines[5])
}

func TestTableDerivedColumnsAndMaxWidth(t *testing.T) {
	body := `[{"id": "1", "description": "` + strings.Repeat("x", 100) + `"}]`
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(body), ""),
		encode.NewTable(&buf, encode.TableOptions{MaxColumnWidth: 20}))
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Equal(t, "id  description", lines[0])
	assert.Equal(t, "1   "+strings.Repeat("x", 19)+"…", lines[2])
}

func TestTableEmptyStream(t *testing.T) {
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(`[]`), ""),
		encode.NewTable(&buf, encode.TableOptions{Columns: []string{"id"}}))
	require.NoError(t, err)
	// Header and divider only.
	assert.Equal(t, "id\n--\n", buf.String())
}

// recordingRowWriter stands in for a rich renderer (e.g. tablewriter in
// streaming mode) to verify the engine/presentation split: the engine must
// deliver the header (with sampled widths) exactly once before any row, and
// rows in order.
type recordingRowWriter struct {
	columns []encode.Column
	rows    [][]string
	headers int
	closed  bool
}

func (r *recordingRowWriter) WriteHeader(columns []encode.Column) error {
	r.headers++
	r.columns = columns
	return nil
}

func (r *recordingRowWriter) WriteRow(cells []string) error {
	if r.headers == 0 {
		panic("row written before header")
	}
	r.rows = append(r.rows, append([]string(nil), cells...))
	return nil
}

func (r *recordingRowWriter) Close() error {
	r.closed = true
	return nil
}

func TestTableWithCustomRowWriter(t *testing.T) {
	var sb strings.Builder
	sb.WriteString(`[`)
	for i := range 6 {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `{"id": "%d", "name": "device-%d"}`, i, i)
	}
	sb.WriteString(`]`)

	rw := &recordingRowWriter{}
	err := output.Render(context.Background(),
		output.FromBytes([]byte(sb.String()), ""),
		encode.NewTableWithWriter(rw, encode.TableOptions{
			Columns:    []string{"id", "name"},
			SampleSize: 3,
		}))
	require.NoError(t, err)

	assert.Equal(t, 1, rw.headers, "header must be written exactly once")
	assert.Equal(t, []encode.Column{
		{Name: "id", Width: 2, Align: encode.AlignLeft},
		{Name: "name", Width: len("device-0"), Align: encode.AlignLeft},
	}, rw.columns, "columns with widths from sample (max of header and sampled cells)")
	require.Len(t, rw.rows, 6)
	assert.Equal(t, []string{"0", "device-0"}, rw.rows[0])
	assert.Equal(t, []string{"5", "device-5"}, rw.rows[5])
	assert.True(t, rw.closed)
}

func TestTableNumericAlignment(t *testing.T) {
	body := `[{"name": "dev-a", "count": 5}, {"name": "dev-b", "count": 1234}]`
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(body), ""),
		encode.NewTable(&buf, encode.TableOptions{Columns: []string{"name", "count"}}))
	require.NoError(t, err)
	assert.Equal(t,
		"name   count\n"+
			"-----  -----\n"+
			"dev-a      5\n"+
			"dev-b   1234\n",
		buf.String())
}

func TestTableMaxTableWidthDropsColumns(t *testing.T) {
	body := `[{"a": "aaaaaaaaaa", "b": "bbbbbbbbbb", "c": "cccccccccc"}]`
	rw := &recordingRowWriter{}
	err := output.Render(context.Background(),
		output.FromBytes([]byte(body), ""),
		encode.NewTableWithWriter(rw, encode.TableOptions{
			Columns:       []string{"a", "b", "c"},
			MaxTableWidth: 30,
		}))
	require.NoError(t, err)
	// a: used 10+3=13; b: 13+10+3=26 fits; c: 26+10+3 > 30 and the
	// leftover space (30-26-3-3) is not usable, so c is dropped.
	require.Len(t, rw.columns, 2, "third column must be dropped")
	assert.Equal(t, "a", rw.columns[0].Name)
	assert.Equal(t, 10, rw.columns[0].Width)
	assert.Equal(t, "b", rw.columns[1].Name)
	assert.Equal(t, 10, rw.columns[1].Width)
	require.Len(t, rw.rows, 1)
	assert.Len(t, rw.rows[0], 2, "rows must only contain kept columns")
}

func TestTableCustomFormatterAndTransform(t *testing.T) {
	body := `[{"name": "device-with-a-long-name", "value": 1500}]`
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(body), ""),
		encode.NewTable(&buf, encode.TableOptions{
			Columns:        []string{"name", "value"},
			MaxColumnWidth: 10,
			Formatter: func(doc jsondoc.JSONDoc, column string) (string, encode.Align) {
				v := doc.Get(column)
				if v.Type == gjson.Number {
					return v.String() + " mW", encode.AlignRight
				}
				return v.String(), encode.AlignLeft
			},
			Transform: func(cell string, width int) string {
				if width > 0 && len(cell) > width {
					return cell[:width-1] + ">"
				}
				return cell
			},
		}))
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Equal(t, "device-wi>  1500 mW", lines[2])
}

func TestTableFlushBoundsLatency(t *testing.T) {
	// A stream stalls before the sample window fills; Flush must render what
	// has arrived, and later rows must still stream.
	rw := &recordingRowWriter{}
	table := encode.NewTableWithWriter(rw, encode.TableOptions{
		Columns:    []string{"id"},
		SampleSize: 50,
	})
	require.NoError(t, table.Write(jsondoc.New([]byte(`{"id": "1"}`))))
	assert.Equal(t, 0, rw.headers, "still sampling")
	require.NoError(t, table.Flush())
	assert.Equal(t, 1, rw.headers)
	assert.Len(t, rw.rows, 1)
	require.NoError(t, table.Write(jsondoc.New([]byte(`{"id": "2"}`))))
	assert.Len(t, rw.rows, 2)
	require.NoError(t, table.Close())
	assert.True(t, rw.closed)
	require.NoError(t, table.Close(), "close must be idempotent")
}

func TestTableMultilineValuesSanitized(t *testing.T) {
	body := `[{"id": "1", "text": "line1\nline2"}]`
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(body), ""),
		encode.NewTable(&buf, encode.TableOptions{Columns: []string{"id", "text"}}))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "line1 line2")
}
