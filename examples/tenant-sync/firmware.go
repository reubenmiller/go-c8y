package main

import (
	"fmt"
	"regexp"
	"strings"
)

// FirmwareInfo represents a firmware image parsed from a filename
type FirmwareInfo struct {
	FilePath   string
	Filename   string
	Name       string
	Version    string
	URL        string // external reference instead of a local file
	DeviceType string // expanded from the manifest deviceType template
}

// ExpandPlaceholders substitutes {name}, {version} and {filename} in a
// template string with the values parsed from this artifact
func (info *FirmwareInfo) ExpandPlaceholders(template string) string {
	return strings.NewReplacer(
		"{name}", info.Name,
		"{version}", info.Version,
		"{filename}", info.Filename,
	).Replace(template)
}

// firmwareExtensions are known OS image / firmware artifact extensions,
// longest first so multi-part extensions win (e.g. ".rootfs.wic.bz2").
// Covers the default outputs of common embedded build systems:
//   - Yocto/OpenEmbedded: .wic[.gz|.bz2|.xz|.zst], .rootfs.*, .ext4, .squashfs, .ubi, .tar.*
//   - Buildroot:          sdcard.img, .img[.gz|.xz|.zst], .ext2/.ext4, .squashfs
//   - Rugix Bakery:       .img[.xz], .rugixb (Rugix bundles)
//   - Update frameworks:  .swu (SWUpdate), .raucb (RAUC), .mender
var firmwareExtensions = []string{
	".rootfs.wic.bz2", ".rootfs.wic.gz", ".rootfs.wic.xz", ".rootfs.wic.zst",
	".rootfs.tar.bz2", ".rootfs.tar.gz", ".rootfs.tar.xz",
	".rootfs.wic", ".rootfs.ext4", ".rootfs.squashfs",
	".wic.bz2", ".wic.gz", ".wic.xz", ".wic.zst", ".wic",
	".img.bz2", ".img.gz", ".img.xz", ".img.zst", ".img",
	".ext2.gz", ".ext4.gz", ".ext2", ".ext3", ".ext4",
	".tar.bz2", ".tar.gz", ".tar.xz", ".tar.zst", ".tgz",
	".squashfs", ".sqfs", ".ubi", ".ubifs", ".itb",
	".swu", ".raucb", ".rugixb", ".mender", ".simg",
	".bin", ".fw", ".hex", ".dfu", ".zip",
}

// trailing version patterns checked against the extension-stripped filename,
// most specific first
var firmwareVersionPatterns = []*regexp.Regexp{
	// Semver-ish with optional pre-release/build metadata, separated by - or _
	// e.g. "core-image-tedge-rpi4-1.2.3", "image_v2.0.1-rc1+g1234abc"
	regexp.MustCompile(`^(.*?)[-_]v?(\d+\.\d+(?:\.\d+)*(?:[~+-][0-9A-Za-z~+.-]+)?)$`),
	// Yocto build timestamps: 8 (date) to 14 (datetime) digits
	// e.g. "core-image-minimal-raspberrypi4-64-20240115103000"
	regexp.MustCompile(`^(.*?)[-_](\d{8,14})$`),
	// Single number, e.g. "firmware-r12"
	regexp.MustCompile(`^(.*?)[-_]r?(\d+)$`),
}

// IsFirmwareFile checks whether the filename has a known firmware extension
func IsFirmwareFile(filename string) bool {
	lower := strings.ToLower(filename)
	for _, ext := range firmwareExtensions {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	return false
}

// stripFirmwareExtension removes the (possibly multi-part) firmware extension
func stripFirmwareExtension(filename string) string {
	lower := strings.ToLower(filename)
	for _, ext := range firmwareExtensions {
		if strings.HasSuffix(lower, ext) {
			return filename[:len(filename)-len(ext)]
		}
	}
	// Unknown extension: strip the last extension only
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		return filename[:idx]
	}
	return filename
}

// ParseFirmwareFromFilename extracts a firmware name and version from an OS
// image filename. versionPattern is an optional regular expression with a
// single capture group for custom naming schemes.
//
// Examples:
//   - "core-image-minimal-raspberrypi4-64-20240115103000.rootfs.wic.bz2"
//     -> name: "core-image-minimal-raspberrypi4-64", version: "20240115103000"
//   - "tedge-rugix-pi-v1.4.2.img.xz" -> name: "tedge-rugix-pi", version: "1.4.2"
//   - "buildroot-sdcard.img" -> name: "buildroot-sdcard", version: "" (needs a hint)
func ParseFirmwareFromFilename(filePath, filename, versionPattern string) (*FirmwareInfo, error) {
	info := &FirmwareInfo{
		FilePath: filePath,
		Filename: filename,
	}

	base := stripFirmwareExtension(filename)
	// Yocto image names can carry a ".rootfs" suffix before the image extension
	base = strings.TrimSuffix(base, ".rootfs")

	if versionPattern != "" {
		re, err := regexp.Compile(versionPattern)
		if err != nil {
			return nil, fmt.Errorf("invalid versionPattern %q: %w", versionPattern, err)
		}
		if match := re.FindStringSubmatch(filename); len(match) >= 2 {
			info.Version = match[1]
			// Name is everything before the full pattern match
			if idx := strings.Index(base, match[0]); idx > 0 {
				info.Name = strings.Trim(base[:idx], "-_.")
			} else {
				info.Name = base
			}
			return info, nil
		}
		// Pattern did not match; fall through to default parsing
	}

	for _, re := range firmwareVersionPatterns {
		if match := re.FindStringSubmatch(base); len(match) == 3 {
			info.Name = strings.Trim(match[1], "-_.")
			info.Version = match[2]
			return info, nil
		}
	}

	// No version in the filename; the caller can fall back to a version hint
	// (e.g. the GitHub release tag) or an explicit version from the manifest
	info.Name = strings.Trim(base, "-_.")
	return info, nil
}
