package encode_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/encode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	columns []string
	widths  []int
	rows    [][]string
	headers int
	closed  bool
}

func (r *recordingRowWriter) WriteHeader(columns []string, widths []int) error {
	r.headers++
	r.columns = columns
	r.widths = widths
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
	assert.Equal(t, []string{"id", "name"}, rw.columns)
	assert.Equal(t, []int{2, len("device-0")}, rw.widths, "widths from sample (max of header and sampled cells)")
	require.Len(t, rw.rows, 6)
	assert.Equal(t, []string{"0", "device-0"}, rw.rows[0])
	assert.Equal(t, []string{"5", "device-5"}, rw.rows[5])
	assert.True(t, rw.closed)
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
