package microservice

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ParsePollRate(t *testing.T) {
	cases := []struct {
		value    string
		expected time.Duration
		wantErr  bool
	}{
		{"30s", 30 * time.Second, false},
		{"5m", 5 * time.Minute, false},
		{"1h30m", 90 * time.Minute, false},
		// Legacy cron-style format used by the previous scheduler
		{"@every 30s", 30 * time.Second, false},
		{"@every 5m", 5 * time.Minute, false},
		{" @every 10s ", 10 * time.Second, false},
		// Unsupported values
		{"", 0, true},
		{"-5s", 0, true},
		{"0 * * * *", 0, true},
		{"@daily", 0, true},
	}

	for _, tc := range cases {
		d, err := ParsePollRate(tc.value)
		if tc.wantErr {
			assert.Error(t, err, "value: %q", tc.value)
		} else {
			require.NoError(t, err, "value: %q", tc.value)
			assert.Equal(t, tc.expected, d, "value: %q", tc.value)
		}
	}
}

func Test_OperationPolling_StartStop(t *testing.T) {
	ms := New(Options{})
	ms.Config.SetDefault("agent.operations.pollRate", "1h")

	// Stop is safe when polling has not been started
	ms.StopOperationPolling()

	ms.StartOperationPolling()
	require.NotNil(t, ms.stopPolling)

	// Starting twice does not replace the running poller
	stop := ms.stopPolling
	ms.StartOperationPolling()
	assert.Equal(t, stop, ms.stopPolling)

	ms.StopOperationPolling()
	assert.Nil(t, ms.stopPolling)

	// Stop is idempotent
	ms.StopOperationPolling()
}
