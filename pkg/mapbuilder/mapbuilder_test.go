package mapbuilder_test

import (
	"encoding/json"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/mapbuilder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMapBuilderFromJSON(t *testing.T) {
	b, err := mapbuilder.NewMapBuilderFromJSON(`{"name":"device","count":2}`)
	require.NoError(t, err)
	assert.Equal(t, "device", b.Get("name"))

	name, ok := b.GetString("name")
	assert.True(t, ok)
	assert.Equal(t, "device", name)

	_, ok = b.GetString("count")
	assert.False(t, ok, "non-string value should report ok=false")
}

func TestNewMapBuilderFromJSONInvalid(t *testing.T) {
	_, err := mapbuilder.NewMapBuilderFromJSON(`{not json}`)
	assert.Error(t, err)
}

func TestMapBuilderSet(t *testing.T) {
	b := mapbuilder.NewMapBuilder()
	require.NoError(t, b.Set("name", "value"))
	require.NoError(t, b.Set("nested.deep.key", 42))

	assert.Equal(t, "value", b.Get("name"))

	nested, ok := b.Get("nested").(map[string]any)
	require.True(t, ok)
	deep, ok := nested["deep"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, 42, deep["key"])
}

func TestMapBuilderSetMapAndGetMap(t *testing.T) {
	b := mapbuilder.NewMapBuilder()
	b.SetMap(map[string]any{"a": 1})
	assert.Equal(t, map[string]any{"a": 1}, b.GetMap())
}

func TestMapBuilderMarshalJSON(t *testing.T) {
	t.Run("uninitialized body errors", func(t *testing.T) {
		b := mapbuilder.NewMapBuilder()
		_, err := b.MarshalJSON()
		assert.Error(t, err)
	})

	t.Run("marshals body", func(t *testing.T) {
		b := mapbuilder.NewMapBuilderWithInit(map[string]any{"name": "test"})
		out, err := b.MarshalJSON()
		require.NoError(t, err)
		assert.JSONEq(t, `{"name":"test"}`, string(out))
	})
}

func TestNewMapBuilderFromJsonnetSnippet(t *testing.T) {
	b, err := mapbuilder.NewMapBuilderFromJsonnetSnippet(`{name: "device", value: 1 + 2}`)
	require.NoError(t, err)
	assert.Equal(t, "device", b.Get("name"))
	assert.EqualValues(t, 3, b.Get("value"))
}

func TestNewMapBuilderFromJsonnetSnippetInvalid(t *testing.T) {
	_, err := mapbuilder.NewMapBuilderFromJsonnetSnippet(`{invalid +`)
	assert.Error(t, err)
}

func TestMapBuilderMergeJsonnet(t *testing.T) {
	t.Run("forward applies snippet over existing", func(t *testing.T) {
		b := mapbuilder.NewMapBuilderWithInit(map[string]any{"a": "base", "keep": true})
		require.NoError(t, b.MergeJsonnet(`{a: "override", b: "new"}`, false))

		assert.Equal(t, "override", b.Get("a"))
		assert.Equal(t, "new", b.Get("b"))
		assert.Equal(t, true, b.Get("keep"))
	})

	t.Run("reverse uses snippet as base", func(t *testing.T) {
		b := mapbuilder.NewMapBuilderWithInit(map[string]any{"a": "existing"})
		// reverse=true: snippet is base, existing data applied on top, so existing wins
		require.NoError(t, b.MergeJsonnet(`{a: "snippet", b: "snippet-only"}`, true))

		assert.Equal(t, "existing", b.Get("a"))
		assert.Equal(t, "snippet-only", b.Get("b"))
	})
}

func TestMapBuilderRoundTrip(t *testing.T) {
	b, err := mapbuilder.NewMapBuilderFromJSON(`{"a":1}`)
	require.NoError(t, err)
	require.NoError(t, b.Set("b", 2))

	out, err := b.MarshalJSON()
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(out, &got))
	assert.EqualValues(t, 1, got["a"])
	assert.EqualValues(t, 2, got["b"])
}
