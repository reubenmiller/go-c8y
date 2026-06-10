package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// exportedCollection mirrors the JSON format of a collection exported by the
// platform UI (including the bookkeeping fields exports carry)
const exportedCollection = `{
	"name": "custom_devmgmt",
	"type": "c8y_SmartRest2Template",
	"com_cumulocity_model_smartrest_csv_CsvSmartRestTemplate": {
		"requestTemplates": [],
		"responseTemplates": [
			{
				"msgId": "dm101",
				"condition": "set_wifi",
				"base": "set_wifi",
				"name": "set_wifi",
				"pattern": ["name", "ssid", "type"]
			}
		]
	},
	"__externalId": "custom_devmgmt"
}`

func TestLoadSmartRestCollection(t *testing.T) {
	writeCollection := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "collection.json")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
		return path
	}

	t.Run("exported example collection", func(t *testing.T) {
		collection, err := loadSmartRestCollection(writeCollection(t, exportedCollection), "")
		require.NoError(t, err)
		assert.Equal(t, "custom_devmgmt", collection.Name)
		assert.Empty(t, collection.Templates.RequestTemplates)
		require.Len(t, collection.Templates.ResponseTemplates, 1)
		assert.Equal(t, "dm101", collection.Templates.ResponseTemplates[0]["msgId"])
	})

	t.Run("manifest name overrides the file", func(t *testing.T) {
		collection, err := loadSmartRestCollection(writeCollection(t, exportedCollection), "renamed")
		require.NoError(t, err)
		assert.Equal(t, "renamed", collection.Name)
	})

	t.Run("name falls back to __externalId", func(t *testing.T) {
		path := writeCollection(t, `{
			"__externalId": "from_external_id",
			"com_cumulocity_model_smartrest_csv_CsvSmartRestTemplate": {"requestTemplates": [], "responseTemplates": []}
		}`)
		collection, err := loadSmartRestCollection(path, "")
		require.NoError(t, err)
		assert.Equal(t, "from_external_id", collection.Name)
	})

	t.Run("wrong type is rejected", func(t *testing.T) {
		path := writeCollection(t, `{"name": "x", "type": "c8y_Profile",
			"com_cumulocity_model_smartrest_csv_CsvSmartRestTemplate": {}}`)
		_, err := loadSmartRestCollection(path, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a SmartREST 2.0 template collection")
	})

	t.Run("missing template fragment is rejected", func(t *testing.T) {
		path := writeCollection(t, `{"name": "x", "type": "c8y_SmartRest2Template"}`)
		_, err := loadSmartRestCollection(path, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing")
	})

	t.Run("missing name is rejected", func(t *testing.T) {
		path := writeCollection(t, `{
			"com_cumulocity_model_smartrest_csv_CsvSmartRestTemplate": {"requestTemplates": [], "responseTemplates": []}
		}`)
		_, err := loadSmartRestCollection(path, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name not found")
	})

	t.Run("invalid json is rejected", func(t *testing.T) {
		path := writeCollection(t, `{not json`)
		_, err := loadSmartRestCollection(path, "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid JSON")
	})
}

func TestSmartRestResolvedSourceDefaultsToJSONPattern(t *testing.T) {
	spec := SmartRestTemplateSpec{Source: Source{Path: "./smartrest"}}
	assert.Equal(t, []string{"*.json"}, spec.resolvedSource().Patterns)

	// Explicit patterns win
	spec = SmartRestTemplateSpec{Source: Source{Path: "./smartrest", Patterns: []string{"custom*.json"}}}
	assert.Equal(t, []string{"custom*.json"}, spec.resolvedSource().Patterns)
}

func TestSyncSmartRestTemplatesDryRun(t *testing.T) {
	syncer, dir := newTestSyncer(t)
	syncer.DryRun = true
	require.NoError(t, os.WriteFile(filepath.Join(dir, "collection_a.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "collection_b.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "not-a-collection.txt"), []byte(``), 0o644))

	err := syncer.SyncSmartRestTemplates(context.Background(), []SmartRestTemplateSpec{
		{Source: Source{Path: "."}},
	})
	require.NoError(t, err)

	// Only the *.json files are picked up
	require.Len(t, syncer.Results, 2)
	for _, result := range syncer.Results {
		assert.Equal(t, ActionPlanned, result.Action)
		assert.Equal(t, SectionSmartRest, result.Section)
	}
	assert.Equal(t, "collection_a.json", syncer.Results[0].Item)
}

func TestSyncSmartRestTemplatesNameRequiresSingleFile(t *testing.T) {
	syncer, dir := newTestSyncer(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.json"), []byte(`{}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.json"), []byte(`{}`), 0o644))

	err := syncer.SyncSmartRestTemplates(context.Background(), []SmartRestTemplateSpec{
		{Name: "single", Source: Source{Path: "."}},
	})
	require.NoError(t, err)

	require.Len(t, syncer.Results, 1)
	assert.Equal(t, ActionFailed, syncer.Results[0].Action)
	assert.Contains(t, syncer.Results[0].Err.Error(), "'name' is set")
}
