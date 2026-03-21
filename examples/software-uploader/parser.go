package main

import (
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

// SoftwareInfo represents parsed information from a filename
type SoftwareInfo struct {
	FilePath     string
	Filename     string
	Name         string
	Version      string
	SoftwareType string
	Architecture string            // e.g., "arm64", "amd64", "aarch64", "x86_64"
	Metadata     map[string]string // Additional metadata like release, epoch, etc.
}

// Parser interface for extension-specific parsing strategies
type Parser interface {
	// CanParse checks if this parser can handle the given filename
	CanParse(filename string) bool
	// Parse extracts software information from the filename
	Parse(filepath, filename string) (*SoftwareInfo, error)
}

// DebianParser handles .deb package naming convention
// Format: name_version_architecture.deb
// Example: tedge-flows_1.6.2~584+gd629c53_arm64.deb
type DebianParser struct{}

func (p *DebianParser) CanParse(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".deb")
}

func (p *DebianParser) Parse(filePath, filename string) (*SoftwareInfo, error) {
	// Remove .deb extension
	nameWithoutExt := strings.TrimSuffix(filename, ".deb")

	// Debian format: name_version_arch
	parts := strings.Split(nameWithoutExt, "_")

	info := &SoftwareInfo{
		FilePath:     filePath,
		Filename:     filename,
		SoftwareType: "apt",
		Metadata:     make(map[string]string),
	}

	if len(parts) >= 3 {
		// Standard format: name_version_arch
		info.Name = parts[0]
		info.Version = parts[1]
		info.Architecture = parts[2]
	} else if len(parts) == 2 {
		// Missing architecture: name_version
		info.Name = parts[0]
		info.Version = parts[1]
	} else {
		// Fall back to generic parsing
		info.Name = nameWithoutExt
		info.Version = "1.0.0"
	}

	return info, nil
}

// IPKParser handles OpenWrt/OpenEmbedded/Yocto .ipk package naming convention
// Format: name_version_architecture.ipk
// Example: tedge_1.6.2~584+gd629c53_aarch64.ipk
type IPKParser struct{}

func (p *IPKParser) CanParse(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".ipk")
}

func (p *IPKParser) Parse(filePath, filename string) (*SoftwareInfo, error) {
	// Remove .ipk extension
	nameWithoutExt := strings.TrimSuffix(filename, ".ipk")

	// IPK format mirrors Debian: name_version_arch
	parts := strings.Split(nameWithoutExt, "_")

	info := &SoftwareInfo{
		FilePath:     filePath,
		Filename:     filename,
		SoftwareType: "ipk",
		Metadata:     make(map[string]string),
	}

	if len(parts) >= 3 {
		info.Name = parts[0]
		info.Version = parts[1]
		info.Architecture = parts[2]
	} else if len(parts) == 2 {
		info.Name = parts[0]
		info.Version = parts[1]
	} else {
		info.Name = nameWithoutExt
		info.Version = "1.0.0"
	}

	return info, nil
}

// RPMParser handles .rpm package naming convention
// Format: name-version-release.architecture.rpm
// Example: tedge-flows-1.6.2~584+gd629c53-1.aarch64.rpm
type RPMParser struct{}

func (p *RPMParser) CanParse(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".rpm")
}

func (p *RPMParser) Parse(filePath, filename string) (*SoftwareInfo, error) {
	// Remove .rpm extension
	nameWithoutExt := strings.TrimSuffix(filename, ".rpm")

	info := &SoftwareInfo{
		FilePath:     filePath,
		Filename:     filename,
		SoftwareType: "rpm",
		Metadata:     make(map[string]string),
	}

	// RPM format: name-version-release.arch
	// Find the architecture (usually the last part before .rpm)
	lastDot := strings.LastIndex(nameWithoutExt, ".")
	if lastDot > 0 {
		info.Architecture = nameWithoutExt[lastDot+1:]
		nameWithoutExt = nameWithoutExt[:lastDot]
	}

	// Now split by dashes, but we need to be careful as name can contain dashes
	// Find the last two dash-separated parts (release and version)
	parts := strings.Split(nameWithoutExt, "-")
	if len(parts) >= 3 {
		// Take the last part as release
		release := parts[len(parts)-1]
		info.Metadata["release"] = release

		// Take the second-to-last as version
		version := parts[len(parts)-2]
		info.Version = version

		// Everything else is the name
		info.Name = strings.Join(parts[:len(parts)-2], "-")
	} else if len(parts) == 2 {
		// name-version format
		info.Name = parts[0]
		info.Version = parts[1]
	} else {
		// Fallback
		info.Name = nameWithoutExt
		info.Version = "1.0.0"
	}

	return info, nil
}

// APKParser handles Alpine .apk package naming convention
// Format: name_version-release_architecture.apk
// Example: tedge-flows_1.6.2_rc584+gd629c53-r0_aarch64.apk
type APKParser struct{}

func (p *APKParser) CanParse(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".apk")
}

func (p *APKParser) Parse(filePath, filename string) (*SoftwareInfo, error) {
	// Remove .apk extension
	nameWithoutExt := strings.TrimSuffix(filename, ".apk")

	info := &SoftwareInfo{
		FilePath:     filePath,
		Filename:     filename,
		SoftwareType: "apk",
		Metadata:     make(map[string]string),
	}

	// APK format: name_version-release_arch
	// Architecture is typically the last underscore-separated part
	// Common architectures: aarch64, x86_64, x86, armhf, armv7, etc.
	// Need to handle x86_64 specially as it contains an underscore

	// List of known architectures
	knownArchs := []string{"x86_64", "aarch64", "x86", "armhf", "armv7", "arm", "i386", "i686", "noarch"}

	// Try to find a known architecture at the end
	for _, arch := range knownArchs {
		suffix := "_" + arch
		if strings.HasSuffix(nameWithoutExt, suffix) {
			info.Architecture = arch
			nameWithoutExt = nameWithoutExt[:len(nameWithoutExt)-len(suffix)]
			break
		}
	}

	// Now split name from version-release
	firstUnderscore := strings.Index(nameWithoutExt, "_")
	if firstUnderscore > 0 {
		info.Name = nameWithoutExt[:firstUnderscore]
		versionRelease := nameWithoutExt[firstUnderscore+1:]

		// Try to split version from release using the last dash
		if dashIdx := strings.LastIndex(versionRelease, "-"); dashIdx > 0 {
			info.Version = versionRelease[:dashIdx]
			info.Metadata["release"] = versionRelease[dashIdx+1:]
		} else {
			info.Version = versionRelease
		}
	} else {
		// No underscore found, use entire name
		info.Name = nameWithoutExt
		info.Version = "1.0.0"
	}

	return info, nil
}

// TarGzParser handles .tar.gz archive naming convention
// Usually follows: name-version.tar.gz or name_version.tar.gz
// Example: tedge_1.6.2-rc584+gd629c53_aarch64-unknown-linux-musl.tar.gz
type TarGzParser struct{}

func (p *TarGzParser) CanParse(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz")
}

func (p *TarGzParser) Parse(filePath, filename string) (*SoftwareInfo, error) {
	// Remove .tar.gz or .tgz extension
	nameWithoutExt := filename
	if strings.HasSuffix(strings.ToLower(filename), ".tar.gz") {
		nameWithoutExt = filename[:len(filename)-7]
	} else if strings.HasSuffix(strings.ToLower(filename), ".tgz") {
		nameWithoutExt = filename[:len(filename)-4]
	}

	info := &SoftwareInfo{
		FilePath:     filePath,
		Filename:     filename,
		SoftwareType: "archive",
		Metadata:     make(map[string]string),
	}

	// Try to detect architecture patterns like aarch64-unknown-linux-musl
	// These are typically at the end of the filename
	archPatterns := []struct {
		pattern string
		arch    string
	}{
		{"aarch64-unknown-linux-musl", "aarch64"},
		{"aarch64-unknown-linux-gnu", "aarch64"},
		{"x86_64-unknown-linux-musl", "x86_64"},
		{"x86_64-unknown-linux-gnu", "x86_64"},
		{"aarch64", "aarch64"},
		{"arm64", "arm64"},
		{"x86_64", "x86_64"},
		{"amd64", "amd64"},
		{"i386", "i386"},
		{"i686", "i686"},
		{"armhf", "armhf"},
		{"armv7", "armv7"},
	}

	for _, ap := range archPatterns {
		pattern := "_" + ap.pattern
		if idx := strings.LastIndex(nameWithoutExt, pattern); idx > 0 {
			info.Architecture = ap.arch
			nameWithoutExt = nameWithoutExt[:idx]
			break
		}
		pattern = "-" + ap.pattern
		if idx := strings.LastIndex(nameWithoutExt, pattern); idx > 0 {
			info.Architecture = ap.arch
			nameWithoutExt = nameWithoutExt[:idx]
			break
		}
	}

	// Use generic extraction
	name, version := extractNameAndVersion(nameWithoutExt)
	if version == "" {
		version = "1.0.0"
	}
	info.Name = name
	info.Version = version

	return info, nil
}

// ArchLinuxParser handles Arch Linux pacman package naming convention
// Format: name-version-pkgrel-architecture.pkg.tar.zst (or .pkg.tar.xz)
// Example: pacman-6.0.2-6-x86_64.pkg.tar.zst
type ArchLinuxParser struct{}

func (p *ArchLinuxParser) CanParse(filename string) bool {
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".pkg.tar.zst") || strings.HasSuffix(lower, ".pkg.tar.xz")
}

func (p *ArchLinuxParser) Parse(filePath, filename string) (*SoftwareInfo, error) {
	lower := strings.ToLower(filename)
	nameWithoutExt := filename
	if strings.HasSuffix(lower, ".pkg.tar.zst") {
		nameWithoutExt = filename[:len(filename)-12]
	} else if strings.HasSuffix(lower, ".pkg.tar.xz") {
		nameWithoutExt = filename[:len(filename)-11]
	}

	info := &SoftwareInfo{
		FilePath:     filePath,
		Filename:     filename,
		SoftwareType: "pacman",
		Metadata:     make(map[string]string),
	}

	// Arch format: name-version-pkgrel-arch
	// pkgver must not contain hyphens, so we can safely split from the right.
	parts := strings.Split(nameWithoutExt, "-")
	if len(parts) >= 4 {
		info.Architecture = parts[len(parts)-1]
		info.Metadata["pkgrel"] = parts[len(parts)-2]
		info.Version = parts[len(parts)-3]
		info.Name = strings.Join(parts[:len(parts)-3], "-")
	} else if len(parts) == 3 {
		info.Architecture = parts[2]
		info.Version = parts[1]
		info.Name = parts[0]
	} else if len(parts) == 2 {
		info.Version = parts[1]
		info.Name = parts[0]
	} else {
		info.Name = nameWithoutExt
		info.Version = "1.0.0"
	}

	return info, nil
}

// GenericParser is a fallback parser for unknown file types
type GenericParser struct {
	softwareType string
}

func (p *GenericParser) CanParse(filename string) bool {
	return true // Always can parse as fallback
}

func (p *GenericParser) Parse(filePath, filename string) (*SoftwareInfo, error) {
	// Strip extensions
	nameWithoutExt := stripExtensions(filename)

	// Try to extract version
	name, version := extractNameAndVersion(nameWithoutExt)
	if version == "" {
		version = "1.0.0"
	}

	info := &SoftwareInfo{
		FilePath:     filePath,
		Filename:     filename,
		Name:         name,
		Version:      version,
		SoftwareType: p.softwareType,
		Metadata:     make(map[string]string),
	}

	return info, nil
}

// ParserRegistry manages the collection of parsers
type ParserRegistry struct {
	parsers []Parser
}

// NewParserRegistry creates a new parser registry with default parsers
func NewParserRegistry() *ParserRegistry {
	return &ParserRegistry{
		parsers: []Parser{
			&ArchLinuxParser{}, // must come before TarGzParser (.pkg.tar.zst/.pkg.tar.xz)
			&DebianParser{},
			&IPKParser{},
			&RPMParser{},
			&APKParser{},
			&TarGzParser{},
		},
	}
}

// GetParser returns the appropriate parser for the given filename
func (r *ParserRegistry) GetParser(filename string) Parser {
	for _, parser := range r.parsers {
		if parser.CanParse(filename) {
			return parser
		}
	}

	// Fallback to generic parser
	softwareType := detectSoftwareType(filename)
	return &GenericParser{softwareType: softwareType}
}

// Global parser registry
var defaultRegistry = NewParserRegistry()

// ParseSoftwareFromFilename extracts software name and version from a filename
// using extension-specific parsers for better accuracy.
//
// Examples:
//   - "tedge-flows_1.6.2~584+gd629c53_arm64.deb" -> name: "tedge-flows", version: "1.6.2~584+gd629c53", arch: "arm64"
//   - "tedge-flows-1.6.2~584+gd629c53-1.aarch64.rpm" -> name: "tedge-flows", version: "1.6.2~584+gd629c53", arch: "aarch64"
//   - "myapp-1.2.3.tar.gz" -> name: "myapp", version: "1.2.3"
func ParseSoftwareFromFilename(filePath string, defaultType string, namePrefix string) (*SoftwareInfo, error) {
	filename := filepath.Base(filePath)

	// Get the appropriate parser
	parser := defaultRegistry.GetParser(filename)

	// Parse using the selected parser
	info, err := parser.Parse(filePath, filename)
	if err != nil {
		return nil, err
	}

	// Override software type if explicitly provided
	if defaultType != "" {
		info.SoftwareType = defaultType
	}

	// If no software type determined, try to detect it
	if info.SoftwareType == "" {
		info.SoftwareType = detectSoftwareType(filename)
	}

	// Decode name if it is url encoded
	if unescapedName, err := url.QueryUnescape(info.Name); err == nil {
		info.Name = unescapedName
	}

	// Add a prefix to the name if provided
	if namePrefix != "" {
		info.Name = namePrefix + info.Name
	}

	return info, nil
}

// detectSoftwareType detects the software type based on file extension
func detectSoftwareType(filename string) string {
	lower := strings.ToLower(filename)

	// Arch Linux packages use triple extensions — check before single-ext logic
	if strings.HasSuffix(lower, ".pkg.tar.zst") || strings.HasSuffix(lower, ".pkg.tar.xz") {
		return "pacman"
	}

	// Package manager specific extensions
	typeMap := map[string]string{
		".deb":      "apt",
		".rpm":      "rpm",
		".apk":      "apk",
		".ipk":      "ipk",
		".jar":      "java",
		".war":      "java",
		".ear":      "java",
		".msi":      "windows",
		".exe":      "windows",
		".dmg":      "macos",
		".pkg":      "macos",
		".snap":     "snap",
		".flatpak":  "flatpak",
		".appimage": "appimage",
	}

	// Check single extensions
	ext := filepath.Ext(filename)
	if softwareType, ok := typeMap[strings.ToLower(ext)]; ok {
		return softwareType
	}

	// Generic archive types default to "archive"
	archiveExtensions := map[string]bool{
		".tar.gz":  true,
		".tgz":     true,
		".tar.bz2": true,
		".tar.xz":  true,
		".tar.zst": true,
		".zip":     true,
		".tar":     true,
		".7z":      true,
		".rar":     true,
		".gz":      true,
		".bz2":     true,
		".xz":      true,
	}

	for ext := range archiveExtensions {
		if strings.HasSuffix(lower, ext) {
			return "archive"
		}
	}

	// Binary files default to "binary"
	if strings.HasSuffix(lower, ".bin") {
		return "binary"
	}

	// Unknown type
	return ""
}

// stripExtensions removes common file extensions from filename
func stripExtensions(filename string) string {
	// Handle triple/double extensions first
	doubleExtensions := []string{
		".pkg.tar.zst", ".pkg.tar.xz",
		".tar.gz", ".tar.bz2", ".tar.xz", ".tar.zst",
	}

	lower := strings.ToLower(filename)
	for _, ext := range doubleExtensions {
		if strings.HasSuffix(lower, ext) {
			return filename[:len(filename)-len(ext)]
		}
	}

	// Handle single extensions
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}

// extractNameAndVersion attempts to split a string into name and version
func extractNameAndVersion(input string) (name, version string) {
	// Regex patterns for version detection (ordered by specificity)
	patterns := []struct {
		regex *regexp.Regexp
		desc  string
	}{
		{
			// Semver with optional v prefix and separators like -, _, .v, _v
			// Examples: myapp-1.2.3, myapp_v2.0.1, myapp.v3.4.5-beta.1, tedge_1.6.2-rc584+gd629c53
			regex: regexp.MustCompile(`^(.+?)[-_.]v?(\d+\.\d+\.\d+(?:[-+~][a-zA-Z0-9.+~-]+)?)$`),
			desc:  "semver with separator",
		},
		{
			// Semver at the end without separator
			// Examples: myapp1.2.3 (less common but possible)
			regex: regexp.MustCompile(`^(.+?)(\d+\.\d+\.\d+(?:[-+~][a-zA-Z0-9.+~-]+)?)$`),
			desc:  "semver without separator",
		},
		{
			// Major.minor version with optional v prefix
			// Examples: myapp-1.2, myapp_v2.0
			regex: regexp.MustCompile(`^(.+?)[-_.]v?(\d+\.\d+)$`),
			desc:  "major.minor",
		},
		{
			// Single version number with v prefix
			// Examples: myapp-v1, myapp_v2
			regex: regexp.MustCompile(`^(.+?)[-_.]v(\d+)$`),
			desc:  "single version with v",
		},
	}

	// Try each pattern
	for _, pattern := range patterns {
		matches := pattern.regex.FindStringSubmatch(input)
		if len(matches) == 3 {
			name = strings.TrimSpace(matches[1])
			version = strings.TrimSpace(matches[2])

			// Clean up name - remove trailing separators
			name = strings.TrimRight(name, "-_.")

			return name, version
		}
	}

	// No version pattern found
	return input, ""
}

// ValidateSoftwareInfo checks if the parsed information is valid
func ValidateSoftwareInfo(info *SoftwareInfo) error {
	if info.Name == "" {
		return &ValidationError{Field: "name", Message: "name cannot be empty"}
	}
	if info.Version == "" {
		return &ValidationError{Field: "version", Message: "version cannot be empty"}
	}
	return nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// GroupBySoftwareName groups software info by name for consolidation
func GroupBySoftwareName(infos []*SoftwareInfo) map[string][]*SoftwareInfo {
	groups := make(map[string][]*SoftwareInfo)

	for _, info := range infos {
		groups[info.Name] = append(groups[info.Name], info)
	}

	return groups
}

// GroupBySoftwareNameAndArch groups software info by name, architecture, and software type
// This allows separate software items for different architectures and package types
// (e.g. noarch.rpm and noarch.apk are distinct packages)
func GroupBySoftwareNameAndArch(infos []*SoftwareInfo) map[string][]*SoftwareInfo {
	groups := make(map[string][]*SoftwareInfo)

	for _, info := range infos {
		key := GetSoftwareKey(info.Name, info.Architecture, info.SoftwareType)
		groups[key] = append(groups[key], info)
	}

	return groups
}

// GetSoftwareKey returns a unique key for the software (name + architecture + softwareType)
func GetSoftwareKey(name, arch, softwareType string) string {
	key := name
	if arch != "" {
		key += "_" + arch
	}
	if softwareType != "" {
		key += "_" + softwareType
	}
	return key
}

// SoftwareSummary provides a summary of software packages to be uploaded
type SoftwareSummary struct {
	TotalFiles    int
	TotalSoftware int
	TotalVersions int
	Groups        map[string][]*SoftwareInfo
}

// CreateSummary creates a summary of the upload plan
func CreateSummary(infos []*SoftwareInfo) *SoftwareSummary {
	groups := GroupBySoftwareNameAndArch(infos)

	totalVersions := 0
	for _, versions := range groups {
		totalVersions += len(versions)
	}

	return &SoftwareSummary{
		TotalFiles:    len(infos),
		TotalSoftware: len(groups),
		TotalVersions: totalVersions,
		Groups:        groups,
	}
}
