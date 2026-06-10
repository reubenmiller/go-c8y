package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// With explicit urls on every reference no repository lookups are needed, so
// the body can be built without a client.
func TestBuildProfileBodyWithManualURLs(t *testing.T) {
	syncer := &Syncer{}

	body, err := syncer.buildProfileBody(context.Background(), DeviceProfileSpec{
		Name:       "rpi4-base",
		DeviceType: "raspberrypi4-64",
		Firmware: &ProfileFirmwareRef{
			Name:    "core-image",
			Version: "1.0.0",
			URL:     "https://example.com/firmware/core-image-1.0.0.wic.bz2",
		},
		Software: []ProfileSoftwareRef{
			{
				Name:    "tedge",
				Version: "1.6.2",
				Type:    "apt",
				URL:     "https://example.com/software/tedge_1.6.2.deb",
			},
		},
		Configuration: []ProfileConfigurationRef{
			{
				Name: "mosquitto",
				Type: "mosquitto.conf",
				URL:  "https://example.com/config/mosquitto.conf",
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "rpi4-base", body["name"])
	assert.Equal(t, "c8y_Profile", body["type"])
	assert.Equal(t, map[string]any{"type": "raspberrypi4-64"}, body["c8y_Filter"])

	profile := body["c8y_DeviceProfile"].(map[string]any)
	assert.Equal(t, map[string]any{
		"name":    "core-image",
		"version": "1.0.0",
		"url":     "https://example.com/firmware/core-image-1.0.0.wic.bz2",
	}, profile["firmware"])
	assert.Equal(t, []map[string]any{{
		"name":         "tedge",
		"version":      "1.6.2",
		"softwareType": "apt",
		"url":          "https://example.com/software/tedge_1.6.2.deb",
		"action":       "install",
	}}, profile["software"])
	assert.Equal(t, []map[string]any{{
		"name": "mosquitto",
		"type": "mosquitto.conf",
		"url":  "https://example.com/config/mosquitto.conf",
	}}, profile["configuration"])
}

// The sentinel url values "-" and "none" disable the repository lookup and
// leave the url off the profile entry entirely.
func TestBuildProfileBodyWithDisabledURLs(t *testing.T) {
	syncer := &Syncer{}

	body, err := syncer.buildProfileBody(context.Background(), DeviceProfileSpec{
		Name: "no-urls",
		Firmware: &ProfileFirmwareRef{
			Name:    "core-image",
			Version: "1.0.0",
			URL:     "none",
		},
		Software: []ProfileSoftwareRef{
			{Name: "tedge", Version: "1.6.2", URL: "-"},
		},
		Configuration: []ProfileConfigurationRef{
			{Name: "mosquitto", Type: "mosquitto.conf", URL: "none"},
		},
	})
	require.NoError(t, err)

	// c8y_Filter is mandatory and empty without a deviceType restriction
	assert.Equal(t, map[string]any{}, body["c8y_Filter"])

	profile := body["c8y_DeviceProfile"].(map[string]any)
	assert.Equal(t, map[string]any{
		"name":    "core-image",
		"version": "1.0.0",
	}, profile["firmware"])
	assert.Equal(t, []map[string]any{{
		"name":    "tedge",
		"version": "1.6.2",
		"action":  "install",
	}}, profile["software"])
	assert.Equal(t, []map[string]any{{
		"name": "mosquitto",
		"type": "mosquitto.conf",
	}}, profile["configuration"])
}

func TestProfileRefURL(t *testing.T) {
	lookupCalled := false
	lookup := func() (string, error) {
		lookupCalled = true
		return "https://looked-up.example.com", nil
	}

	url, include, err := profileRefURL("https://manual.example.com", lookup)
	require.NoError(t, err)
	assert.True(t, include)
	assert.Equal(t, "https://manual.example.com", url)
	assert.False(t, lookupCalled, "explicit url must not trigger a lookup")

	for _, sentinel := range []string{"-", "none"} {
		url, include, err = profileRefURL(sentinel, lookup)
		require.NoError(t, err)
		assert.False(t, include)
		assert.Empty(t, url)
		assert.False(t, lookupCalled, "sentinel url must not trigger a lookup")
	}

	url, include, err = profileRefURL("", lookup)
	require.NoError(t, err)
	assert.True(t, include)
	assert.Equal(t, "https://looked-up.example.com", url)
	assert.True(t, lookupCalled)
}
