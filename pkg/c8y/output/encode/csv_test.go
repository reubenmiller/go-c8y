package encode_test

import (
	"bytes"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/encode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSVAutoFlushStreamsEachRow verifies that with AutoFlush enabled each row
// reaches the underlying writer as soon as it is written, instead of being held
// in csv.Writer's internal buffer until Close. This guards the streamed output
// path (e.g. a terminal) where items may arrive seconds apart and must be shown
// as they are produced.
func TestCSVAutoFlushStreamsEachRow(t *testing.T) {
	var buf bytes.Buffer
	e := encode.NewCSV(&buf, encode.CSVOptions{Columns: []string{"time"}, AutoFlush: true})

	require.NoError(t, e.Write(jsondoc.New([]byte(`{"time":"t1"}`))))
	assert.Equal(t, "t1\n", buf.String(), "first row should be flushed before the second Write")

	require.NoError(t, e.Write(jsondoc.New([]byte(`{"time":"t2"}`))))
	assert.Equal(t, "t1\nt2\n", buf.String(), "second row should be flushed before Close")

	require.NoError(t, e.Close())
	assert.Equal(t, "t1\nt2\n", buf.String())
}

// TestCSVWithoutAutoFlushBuffersUntilClose documents the default: rows are
// buffered in csv.Writer and only reach the underlying writer on Close, which
// favours throughput for bulk/non-interactive output.
func TestCSVWithoutAutoFlushBuffersUntilClose(t *testing.T) {
	var buf bytes.Buffer
	e := encode.NewCSV(&buf, encode.CSVOptions{Columns: []string{"time"}})

	require.NoError(t, e.Write(jsondoc.New([]byte(`{"time":"t1"}`))))
	require.NoError(t, e.Write(jsondoc.New([]byte(`{"time":"t2"}`))))
	assert.Empty(t, buf.String(), "rows should stay buffered until Close")

	require.NoError(t, e.Close())
	assert.Equal(t, "t1\nt2\n", buf.String())
}

// TestTSVAutoFlushStreamsEachRow is the tab-delimited counterpart, ensuring the
// streamed terminal path works for -o tsv as well as -o csv.
func TestTSVAutoFlushStreamsEachRow(t *testing.T) {
	var buf bytes.Buffer
	e := encode.NewTSV(&buf, encode.CSVOptions{Columns: []string{"a", "b"}, AutoFlush: true})

	require.NoError(t, e.Write(jsondoc.New([]byte(`{"a":"1","b":"2"}`))))
	assert.Equal(t, "1\t2\n", buf.String(), "row should be flushed before Close")

	require.NoError(t, e.Close())
	assert.Equal(t, "1\t2\n", buf.String())
}
