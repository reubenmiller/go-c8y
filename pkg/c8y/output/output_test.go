package output_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/encode"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/filter"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/output/shape"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const collectionBody = `{
	"self": "https://example.com/inventory/managedObjects?pageSize=5",
	"next": "https://example.com/inventory/managedObjects?pageSize=5&currentPage=2",
	"managedObjects": [
		{"id": "1", "name": "linux-device-01", "type": "c8y_Linux", "c8y_Hardware": {"model": "RPi4", "serialNumber": "A1"}},
		{"id": "2", "name": "windows-device-01", "type": "c8y_Windows", "c8y_Hardware": {"model": "NUC", "serialNumber": "B2"}},
		{"id": "3", "name": "linux-device-02", "type": "c8y_Linux", "c8y_Hardware": {"model": "RPi5", "serialNumber": "C3"}}
	],
	"statistics": {"pageSize": 5, "currentPage": 1}
}`

func collect(t *testing.T, seq output.Seq) []string {
	t.Helper()
	var items []string
	for doc, err := range seq {
		require.NoError(t, err)
		items = append(items, string(doc.Raw()))
	}
	return items
}

func TestFromBytes(t *testing.T) {
	items := collect(t, output.FromBytes([]byte(collectionBody), "managedObjects"))
	require.Len(t, items, 3)
	assert.Contains(t, items[0], `"linux-device-01"`)
	assert.Contains(t, items[2], `"linux-device-02"`)
}

func TestFromBytesMissingPath(t *testing.T) {
	for _, err := range output.FromBytes([]byte(collectionBody), "doesNotExist") {
		assert.Error(t, err)
		return
	}
	t.Fatal("expected an error to be yielded")
}

func TestFromReader(t *testing.T) {
	// The target array is preceded and followed by other keys, which must be
	// skipped and left unread respectively.
	items := collect(t, output.FromReader(strings.NewReader(collectionBody), "managedObjects"))
	require.Len(t, items, 3)
	assert.JSONEq(t,
		`{"id": "1", "name": "linux-device-01", "type": "c8y_Linux", "c8y_Hardware": {"model": "RPi4", "serialNumber": "A1"}}`,
		items[0])
}

func TestFromReaderTopLevelArray(t *testing.T) {
	items := collect(t, output.FromReader(strings.NewReader(`[{"id":"1"},{"id":"2"}]`), ""))
	require.Len(t, items, 2)
}

func TestRenderNDJSON(t *testing.T) {
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(collectionBody), "managedObjects"),
		encode.NewNDJSON(&buf))
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 3)
}

func TestRenderFilterSelectJSONArray(t *testing.T) {
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(collectionBody), "managedObjects"),
		encode.NewJSONArray(&buf),
		output.Filter(filter.Like("name", "linux*")),
		shape.Select("id", "name", "c8y_Hardware.serialNumber"),
	)
	require.NoError(t, err)
	assert.JSONEq(t, `[
		{"id": "1", "name": "linux-device-01", "c8y_Hardware": {"serialNumber": "A1"}},
		{"id": "3", "name": "linux-device-02", "c8y_Hardware": {"serialNumber": "C3"}}
	]`, buf.String())
}

func TestRenderCSV(t *testing.T) {
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(collectionBody), "managedObjects"),
		encode.NewCSV(&buf, encode.CSVOptions{
			Header:  true,
			Columns: []string{"id", "name", "c8y_Hardware.model"},
		}),
	)
	require.NoError(t, err)
	assert.Equal(t,
		"id,name,c8y_Hardware.model\n"+
			"1,linux-device-01,RPi4\n"+
			"2,windows-device-01,NUC\n"+
			"3,linux-device-02,RPi5\n",
		buf.String())
}

func TestRenderCSVDerivedColumns(t *testing.T) {
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(collectionBody), "managedObjects"),
		encode.NewTSV(&buf, encode.CSVOptions{Header: true}),
	)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 4)
	assert.Equal(t, "id\tname\ttype\tc8y_Hardware.model\tc8y_Hardware.serialNumber", lines[0])
}

// TestHeadStopsSource verifies the pipeline is pull-based: a downstream limit
// must stop the source from producing further items (i.e. no more pages would
// be fetched from a paginated source).
func TestHeadStopsSource(t *testing.T) {
	produced := 0
	src := output.Seq(func(yield func(jsondoc.JSONDoc, error) bool) {
		for {
			produced++
			if !yield(jsondoc.New([]byte(`{"id":"x"}`)), nil) {
				return
			}
		}
	})

	var buf bytes.Buffer
	err := output.Render(context.Background(), src, encode.NewNDJSON(&buf), output.Head(5))
	require.NoError(t, err)
	assert.Equal(t, 5, produced, "source must stop as soon as the sink stops pulling")
	assert.Len(t, strings.Split(strings.TrimSpace(buf.String()), "\n"), 5)
}

func TestRenderContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := output.Render(ctx,
		output.FromBytes([]byte(collectionBody), "managedObjects"),
		encode.NewNDJSON(&bytes.Buffer{}))
	assert.ErrorIs(t, err, context.Canceled)
}
