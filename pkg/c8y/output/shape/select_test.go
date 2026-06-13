package shape

import (
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectNumericKeyNotArrayIndex guards against a memory-exhaustion bug: a
// large integer-looking object key (e.g. a Cumulocity c8y_Dashboard widget id)
// must be kept as an object key, not interpreted by sjson as an array index.
// Treating "15426326034650895" as an index made sjson pre-allocate an array of
// ~10^16 elements, consuming >128GB of RAM before the process was killed.
func TestSelectNumericKeyNotArrayIndex(t *testing.T) {
	doc := []byte(`{"c8y_Dashboard":{"15426326034650895":{"name":"test"}}}`)

	t.Run("wildcard", func(t *testing.T) {
		out, err := compileSelector([]string{"**"}).apply(jsondoc.New(doc))
		require.NoError(t, err)
		assert.JSONEq(t, string(doc), string(out.Raw()))
	})

	t.Run("exact", func(t *testing.T) {
		out, err := compileSelector([]string{"c8y_Dashboard.15426326034650895.name"}).apply(jsondoc.New(doc))
		require.NoError(t, err)
		assert.JSONEq(t, string(doc), string(out.Raw()))
	})
}

// TestSelectArrayIndexPreserved ensures the numeric-key guard does not break
// genuine array element selection: a numeric segment whose parent is an array
// must still address that index.
func TestSelectArrayIndexPreserved(t *testing.T) {
	doc := []byte(`{"items":["a","b","c"]}`)
	out, err := compileSelector([]string{"items.1"}).apply(jsondoc.New(doc))
	require.NoError(t, err)
	assert.JSONEq(t, `{"items":[null,"b"]}`, string(out.Raw()))
}
