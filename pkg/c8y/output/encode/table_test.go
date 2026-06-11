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

func TestTableMultilineValuesSanitized(t *testing.T) {
	body := `[{"id": "1", "text": "line1\nline2"}]`
	var buf bytes.Buffer
	err := output.Render(context.Background(),
		output.FromBytes([]byte(body), ""),
		encode.NewTable(&buf, encode.TableOptions{Columns: []string{"id", "text"}}))
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "line1 line2")
}
