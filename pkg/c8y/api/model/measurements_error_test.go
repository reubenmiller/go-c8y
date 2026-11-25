package model

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_MeasurementSeries_InvalidTimestampReturnsError(t *testing.T) {
	body := []byte(`{"series":[{"unit":"C","name":"T","type":"c8y_T"}],"values":{"not-a-time":[{"min":1,"max":2}]}}`)
	assert.NotPanics(t, func() {
		var g MeasurementSeriesAggregateGroup
		err := json.Unmarshal(body, &g)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not-a-time")
	})
	assert.NotPanics(t, func() {
		var g MeasurementSeriesGroup
		err := json.Unmarshal([]byte(`{"series":[{"unit":"C","name":"T","type":"c8y_T"}],"values":{"not-a-time":[1]}}`), &g)
		require.Error(t, err)
	})
}
