package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFirmwareFromFilename(t *testing.T) {
	testCases := []struct {
		filename        string
		versionPattern  string
		expectedName    string
		expectedVersion string
	}{
		// Yocto / OpenEmbedded
		{
			filename:        "core-image-minimal-raspberrypi4-64-20240115103000.rootfs.wic.bz2",
			expectedName:    "core-image-minimal-raspberrypi4-64",
			expectedVersion: "20240115103000",
		},
		{
			filename:        "core-image-tedge-rpi4-1.2.3.wic.xz",
			expectedName:    "core-image-tedge-rpi4",
			expectedVersion: "1.2.3",
		},
		{
			filename:        "tedge-image-qemuarm64-20240601.rootfs.tar.gz",
			expectedName:    "tedge-image-qemuarm64",
			expectedVersion: "20240601",
		},
		// Buildroot style (no version in filename)
		{
			filename:        "sdcard.img",
			expectedName:    "sdcard",
			expectedVersion: "",
		},
		{
			filename:        "buildroot-rpi4-2024.02.img.gz",
			expectedName:    "buildroot-rpi4",
			expectedVersion: "2024.02",
		},
		// Rugix Bakery
		{
			filename:        "tedge-rugix-pi-v1.4.2.img.xz",
			expectedName:    "tedge-rugix-pi",
			expectedVersion: "1.4.2",
		},
		{
			filename:        "customer-image-2.1.0.rugixb",
			expectedName:    "customer-image",
			expectedVersion: "2.1.0",
		},
		// Update frameworks
		{
			filename:        "device-update_1.0.0-rc1+g1234abc.swu",
			expectedName:    "device-update",
			expectedVersion: "1.0.0-rc1+g1234abc",
		},
		{
			filename:        "gateway-bundle-3.2.1.raucb",
			expectedName:    "gateway-bundle",
			expectedVersion: "3.2.1",
		},
		{
			filename:        "device-image-1.5.0.mender",
			expectedName:    "device-image",
			expectedVersion: "1.5.0",
		},
		// Plain binary firmware
		{
			filename:        "device-firmware-r12.bin",
			expectedName:    "device-firmware",
			expectedVersion: "12",
		},
		// Custom naming via versionPattern
		{
			filename:        "FW_MODEL7_BUILD20240501.custom",
			versionPattern:  `BUILD(\d+)`,
			expectedName:    "FW_MODEL7",
			expectedVersion: "20240501",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.filename, func(t *testing.T) {
			info, err := ParseFirmwareFromFilename("/tmp/"+tc.filename, tc.filename, tc.versionPattern)
			assert.NoError(t, err)
			assert.Equal(t, tc.expectedName, info.Name, "name")
			assert.Equal(t, tc.expectedVersion, info.Version, "version")
		})
	}
}

func TestParseFirmwareInvalidVersionPattern(t *testing.T) {
	_, err := ParseFirmwareFromFilename("/tmp/a.img", "a.img", "([")
	assert.Error(t, err)
}

func TestIsFirmwareFile(t *testing.T) {
	assert.True(t, IsFirmwareFile("image.wic.bz2"))
	assert.True(t, IsFirmwareFile("image.swu"))
	assert.True(t, IsFirmwareFile("bundle.raucb"))
	assert.True(t, IsFirmwareFile("update.rugixb"))
	assert.False(t, IsFirmwareFile("notes.txt"))
}

func TestExpandPlaceholders(t *testing.T) {
	info := &FirmwareInfo{
		Name:     "core-image-tedge-rpi4",
		Version:  "1.2.3",
		Filename: "core-image-tedge-rpi4-1.2.3.wic.xz",
	}

	assert.Equal(t, "core-image-tedge-rpi4", info.ExpandPlaceholders("{name}"))
	assert.Equal(t, "linux-core-image-tedge-rpi4", info.ExpandPlaceholders("linux-{name}"))
	assert.Equal(t, "core-image-tedge-rpi4-1.2.3", info.ExpandPlaceholders("{name}-{version}"))
	assert.Equal(t, "mydevicetype", info.ExpandPlaceholders("mydevicetype"))
	assert.Equal(t, "", info.ExpandPlaceholders(""))
}
