package shape

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var selectorDoc = []byte(`{
	"id": "12345",
	"name": "device01",
	"type": "c8y_Linux",
	"c8y_Hardware": {"model": "RPi4", "serialNumber": "SN-001", "revision": 2},
	"c8y_IsDevice": {},
	"childAdditions": [{"id": "1"}, {"id": "2"}, {"id": "10"}],
	"my.dotted.key": "dotvalue"
}`)

func apply(t *testing.T, doc []byte, patterns ...string) *Selection {
	t.Helper()
	sel, err := NewSelector(patterns...).Apply(doc)
	require.NoError(t, err)
	return sel
}

func mustJSON(t *testing.T, sel *Selection) string {
	t.Helper()
	out, err := sel.JSON()
	require.NoError(t, err)
	return string(out)
}

func TestSelectorConcretePaths(t *testing.T) {
	sel := apply(t, selectorDoc, "id,name")
	assert.Equal(t, []string{"id", "name"}, sel.Keys())
	assert.JSONEq(t, `{"id": "12345", "name": "device01"}`, mustJSON(t, sel))
}

func TestSelectorIsCaseInsensitive(t *testing.T) {
	sel := apply(t, selectorDoc, "NAME")
	assert.Equal(t, []string{"name"}, sel.Keys())
}

func TestSelectorSubtreeByPathPrefix(t *testing.T) {
	// selecting a path selects its nested subtree
	sel := apply(t, selectorDoc, "c8y_Hardware")
	assert.JSONEq(t, `{"c8y_Hardware": {"model": "RPi4", "serialNumber": "SN-001", "revision": 2}}`, mustJSON(t, sel))
}

func TestSelectorWildcardWithinSegment(t *testing.T) {
	// '*' does not cross separators
	sel := apply(t, selectorDoc, "c8y_Hardware.s*")
	assert.Equal(t, []string{"c8y_Hardware.serialNumber"}, sel.Keys())
}

func TestSelectorGlobstar(t *testing.T) {
	sel := apply(t, selectorDoc, "**.serialNumber")
	assert.Equal(t, []string{"c8y_Hardware.serialNumber"}, sel.Keys())
}

func TestSelectorNaturalOrderOfArrayKeys(t *testing.T) {
	sel := apply(t, selectorDoc, "childAdditions.*.id")
	assert.Equal(t, []string{"childAdditions.0.id", "childAdditions.1.id", "childAdditions.2.id"}, sel.Keys())
	assert.JSONEq(t, `{"childAdditions": [{"id": "1"}, {"id": "2"}, {"id": "10"}]}`, mustJSON(t, sel))
}

func TestSelectorEmptyObjectIsLeaf(t *testing.T) {
	sel := apply(t, selectorDoc, "c8y_IsDevice")
	assert.JSONEq(t, `{"c8y_IsDevice": {}}`, mustJSON(t, sel))
}

func TestSelectorNegation(t *testing.T) {
	sel := apply(t, selectorDoc, "c8y_Hardware.*", "!c8y_Hardware.revision")
	assert.Equal(t, []string{"c8y_Hardware.model", "c8y_Hardware.serialNumber"}, sel.Keys())
}

func TestSelectorAliasConcrete(t *testing.T) {
	sel := apply(t, selectorDoc, "deviceName:name")
	assert.Equal(t, []string{"deviceName"}, sel.Keys())
	assert.JSONEq(t, `{"deviceName": "device01"}`, mustJSON(t, sel))
}

func TestSelectorAliasTrailingWildcard(t *testing.T) {
	sel := apply(t, selectorDoc, "hw:c8y_Hardware.*")
	assert.ElementsMatch(t, []string{"hw.model", "hw.serialNumber", "hw.revision"}, sel.Keys())
}

func TestSelectorUnmatchedPatternKeepsColumn(t *testing.T) {
	sel := apply(t, selectorDoc, "id", "doesNotExist")
	assert.Equal(t, []string{"id", "doesNotExist"}, sel.Keys())
	assert.Equal(t, "12345,", sel.CSV(","))
	groups := sel.Groups()
	require.Len(t, groups, 2)
	assert.Equal(t, "doesNotExist", groups[1].Pattern)
	assert.Empty(t, groups[1].Keys)
}

func TestSelectorCSVQuoting(t *testing.T) {
	doc := []byte(`{"name": "with,comma", "plain": "value", "count": 3}`)
	sel := apply(t, doc, "name,plain,count")
	assert.Equal(t, `"with,comma",value,3`, sel.CSV(","))
}

func TestSelectorNumberLiteralsPreserved(t *testing.T) {
	doc := []byte(`{"value": 10.50, "big": 1e3}`)
	sel := apply(t, doc, "value,big")
	assert.Equal(t, `{"value":10.50,"big":1e3}`, mustJSON(t, sel))
}

func TestSelectorIntegerLikeMapKeys(t *testing.T) {
	// integer-looking map keys must stay object keys, not array indices
	doc := []byte(`{"levels": {"10": "ten", "2": "two"}}`)
	sel := apply(t, doc, "levels.*")
	assert.JSONEq(t, `{"levels": {"2": "two", "10": "ten"}}`, mustJSON(t, sel))
}

func TestSelectorDottedKeys(t *testing.T) {
	// Keys containing literal dots are not selectable: the flattener escapes
	// the dots in the key but glob patterns cannot express that escaping.
	// This matches go-c8y-cli's behavior (the pattern resolves to nothing
	// and is kept as an unmatched column).
	sel := apply(t, selectorDoc, "my\\.dotted\\.key")
	assert.JSONEq(t, `{}`, mustJSON(t, sel))
	assert.Equal(t, []string{"my\\.dotted\\.key"}, sel.Keys())
}

func TestSelectorFlatJSON(t *testing.T) {
	sel := apply(t, selectorDoc, "c8y_Hardware.model,id")
	out, err := sel.FlatJSON()
	require.NoError(t, err)
	assert.JSONEq(t, `{"c8y_Hardware.model": "RPi4", "id": "12345"}`, string(out))
}

func TestSelectorGlobstarOnly(t *testing.T) {
	s := NewSelector("**")
	assert.True(t, s.SelectsEverything())
	sel, err := s.Apply(selectorDoc)
	require.NoError(t, err)
	assert.NotEmpty(t, sel.Keys())

	assert.False(t, NewSelector("id").SelectsEverything())
}

func TestSelectorInvalidJSON(t *testing.T) {
	_, err := NewSelector("id").Apply([]byte(`not json`))
	assert.Error(t, err)
}
