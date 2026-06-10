package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetentionRuleSelector(t *testing.T) {
	assert.Equal(t, "MEASUREMENT/*/*/*", RetentionRuleSpec{DataType: "MEASUREMENT"}.Selector())
	assert.Equal(t, "EVENT/*/c8y_LocationUpdate/*", RetentionRuleSpec{
		DataType: "EVENT",
		Type:     "c8y_LocationUpdate",
	}.Selector())
	assert.Equal(t, "ALARM/c8y_Custom/x/12345", RetentionRuleSpec{
		DataType:     "ALARM",
		FragmentType: "c8y_Custom",
		Type:         "x",
		Source:       "12345",
	}.Selector())
}

func TestLoadManifestRetentionRules(t *testing.T) {
	path := writeManifest(t, `
retentionRules:
  - dataType: MEASUREMENT
    maximumAge: 365
  - dataType: EVENT
    type: c8y_LocationUpdate
    maximumAge: 30
    editable: false
`)
	manifest, err := LoadManifest(path)
	require.NoError(t, err)

	require.Len(t, manifest.RetentionRules, 2)
	assert.Equal(t, int64(365), manifest.RetentionRules[0].MaximumAge)
	assert.Nil(t, manifest.RetentionRules[0].Editable)
	require.NotNil(t, manifest.RetentionRules[1].Editable)
	assert.False(t, *manifest.RetentionRules[1].Editable)
}

func TestSyncRetentionRulesDryRun(t *testing.T) {
	syncer := &Syncer{DryRun: true}

	err := syncer.SyncRetentionRules(context.Background(), []RetentionRuleSpec{
		{DataType: "MEASUREMENT", MaximumAge: 365},
		{DataType: "EVENT", Type: "c8y_LocationUpdate", MaximumAge: 30},
	})
	require.NoError(t, err)

	require.Len(t, syncer.Results, 2)
	assert.Equal(t, ActionPlanned, syncer.Results[0].Action)
	assert.Equal(t, SectionRetentionRules, syncer.Results[0].Section)
	assert.Equal(t, "MEASUREMENT/*/*/*", syncer.Results[0].Item)
	assert.Equal(t, "ensure maximumAge=365", syncer.Results[0].Detail)
	assert.Equal(t, "EVENT/*/c8y_LocationUpdate/*", syncer.Results[1].Item)
}
