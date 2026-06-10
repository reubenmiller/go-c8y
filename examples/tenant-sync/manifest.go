package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Manifest is the declarative description of the desired tenant state.
// Sections are applied in a fixed order so that later sections (e.g. device
// profiles) can reference items created by earlier ones (firmware, software,
// configuration).
type Manifest struct {
	TenantOptions  []TenantOptionSpec  `yaml:"tenantOptions" json:"tenantOptions,omitempty" jsonschema:"description=Tenant options to set (created if missing and updated when the value differs)"`
	Features       []FeatureSpec       `yaml:"features" json:"features,omitempty" jsonschema:"description=Feature toggles to enable or disable for the current tenant"`
	Applications   []ApplicationSpec   `yaml:"applications" json:"applications,omitempty" jsonschema:"description=Applications to subscribe to the current tenant (optionally created/uploaded from a source)"`
	Software       []SoftwareSpec      `yaml:"software" json:"software,omitempty" jsonschema:"description=Software packages to sync into the software repository"`
	Firmware       []FirmwareSpec      `yaml:"firmware" json:"firmware,omitempty" jsonschema:"description=Firmware (OS images) to sync into the firmware repository"`
	Configuration  []ConfigurationSpec `yaml:"configuration" json:"configuration,omitempty" jsonschema:"description=Configuration files to sync into the configuration repository"`
	DeviceProfiles []DeviceProfileSpec `yaml:"deviceProfiles" json:"deviceProfiles,omitempty" jsonschema:"description=Device profiles referencing firmware/software/configuration in the tenant"`
	Hooks          HooksSpec           `yaml:"hooks" json:"hooks,omitempty" jsonschema:"description=Commands executed before and after the sync (e.g. go-c8y-cli calls)"`
}

// HooksSpec defines commands executed around the sync
type HooksSpec struct {
	Pre  []HookSpec `yaml:"pre" json:"pre,omitempty" jsonschema:"description=Commands executed before the sync; a failing pre hook aborts the run"`
	Post []HookSpec `yaml:"post" json:"post,omitempty" jsonschema:"description=Commands executed after the sync (also when sections failed); failures are reported but do not abort"`
}

// HookSpec is a single command executed via the shell with the manifest
// directory as the working directory and the current environment (including
// C8Y_* session variables) passed through
type HookSpec struct {
	Name string `yaml:"name" json:"name,omitempty" jsonschema:"description=Display name of the hook"`
	Run  string `yaml:"run" json:"run" jsonschema:"description=Command executed with 'sh -c' from the manifest directory"`
}

// TenantOptionSpec sets a tenant option (category/key/value). The value is
// either given literally (value) or resolved from the tenant (valueFrom).
type TenantOptionSpec struct {
	Category string `yaml:"category" json:"category" jsonschema:"description=Tenant option category"`
	Key      string `yaml:"key" json:"key" jsonschema:"description=Tenant option key"`
	Value    string `yaml:"value" json:"value,omitempty" jsonschema:"description=Desired value of the tenant option (mutually exclusive with valueFrom)"`

	// ValueFrom resolves the value by a named lookup at apply time,
	// e.g. an application ID looked up by application name
	ValueFrom *TenantOptionValueFrom `yaml:"valueFrom" json:"valueFrom,omitempty" jsonschema:"description=Resolve the value by a named lookup at apply time (mutually exclusive with value)"`
}

// TenantOptionValueFrom resolves a tenant option value by reference.
// Exactly one field must be set.
type TenantOptionValueFrom struct {
	// Application resolves to the application ID looked up by name
	Application string `yaml:"application" json:"application,omitempty" jsonschema:"description=Application name resolved to its application ID,example=devicemanagement"`

	// Device resolves to the managed object ID of a device looked up by name
	Device string `yaml:"device" json:"device,omitempty" jsonschema:"description=Device name resolved to its managed object ID"`
}

// FeatureSpec enables or disables a feature toggle on the current tenant
type FeatureSpec struct {
	Key     string `yaml:"key" json:"key" jsonschema:"description=Feature toggle key"`
	Enabled *bool  `yaml:"enabled" json:"enabled,omitempty" jsonschema:"description=Desired feature state (defaults to true when omitted)"` // defaults to true when omitted
}

func (f FeatureSpec) IsEnabled() bool {
	return f.Enabled == nil || *f.Enabled
}

// ApplicationSpec subscribes an application (by name) to the current tenant.
// With a source, the application is also created if missing and its binary
// (e.g. a microservice or web application zip) uploaded.
type ApplicationSpec struct {
	Name string `yaml:"name" json:"name" jsonschema:"description=Application name"`
	Type string `yaml:"type" json:"type,omitempty" jsonschema:"description=Application type (lookup filter; also used when creating from a source where it defaults to MICROSERVICE),example=MICROSERVICE,example=HOSTED"` // optional application type filter (e.g. MICROSERVICE, HOSTED)

	// ContextPath used when creating the application (defaults to the name)
	ContextPath string `yaml:"contextPath" json:"contextPath,omitempty" jsonschema:"description=Context path used when creating the application (defaults to the name)"`

	// Source provides the application binary (a single zip file). The
	// application is created if missing and the binary uploaded on creation
	// (or on every run with --force).
	Source *Source `yaml:"source" json:"source,omitempty" jsonschema:"description=Where the application binary (zip) comes from; must resolve to a single file"`
}

// Source describes where artifacts come from. Exactly one of Path, URL or
// GitHub should be set.
type Source struct {
	// Path is a local file or directory. Directories are searched recursively
	// using Patterns.
	Path     string   `yaml:"path" json:"path,omitempty" jsonschema:"description=Local file or directory (relative paths are resolved against the manifest directory)"`
	Patterns []string `yaml:"patterns" json:"patterns,omitempty" jsonschema:"description=Glob patterns matched against filenames when path is a directory (default: *)"`

	// URL references an external file without uploading a binary.
	URL string `yaml:"url" json:"url,omitempty" jsonschema:"description=External URL referenced without uploading a binary"`

	// GitHub pulls release assets from a GitHub repository.
	GitHub *GitHubSource `yaml:"github" json:"github,omitempty" jsonschema:"description=Pull release assets from a GitHub repository"`

	// Optional marks the source as optional: when the local path does not
	// exist or no files/assets match, the entry is skipped instead of failing.
	Optional bool `yaml:"optional" json:"optional,omitempty" jsonschema:"description=Skip this entry instead of failing when the path does not exist or nothing matches"`
}

func (s *Source) IsSet() bool {
	return s != nil && (s.Path != "" || s.URL != "" || s.GitHub != nil)
}

func (s *Source) Validate() error {
	count := 0
	if s.Path != "" {
		count++
	}
	if s.URL != "" {
		count++
	}
	if s.GitHub != nil {
		count++
	}
	if count == 0 {
		return fmt.Errorf("source requires one of: path, url, github")
	}
	if count > 1 {
		return fmt.Errorf("source fields path, url and github are mutually exclusive")
	}
	if s.GitHub != nil && !strings.Contains(s.GitHub.Repo, "/") {
		return fmt.Errorf("github.repo must be in the form owner/repo, got %q", s.GitHub.Repo)
	}
	return nil
}

// GitHubSource selects release assets from a GitHub repository
type GitHubSource struct {
	// Repo in the form "owner/name"
	Repo string `yaml:"repo" json:"repo" jsonschema:"description=Repository in the form owner/name,pattern=^[^/]+/[^/]+$"`

	// Release selects which release(s) to use:
	//   "latest" (default) - the latest non-prerelease release
	//   "latest-N"         - the latest N releases (e.g. "latest-5")
	//   "all"              - every release (use with care)
	//   "<tag>"            - a specific release tag (e.g. "v1.2.3")
	Release string `yaml:"release" json:"release,omitempty" jsonschema:"description=Which release(s) to use: latest (default) or latest-N for the latest N releases or all or a specific tag,example=latest,example=latest-5,example=all,example=v1.2.3"`

	// Assets are glob patterns matched against asset filenames (default: ["*"])
	Assets []string `yaml:"assets" json:"assets,omitempty" jsonschema:"description=Glob patterns matched against release asset filenames (default: *)"`

	// IncludePrereleases includes prereleases when Release is "latest" or "all"
	IncludePrereleases bool `yaml:"includePrereleases" json:"includePrereleases,omitempty" jsonschema:"description=Include prereleases when release is latest or all"`

	// LinkOnly references the asset download URL instead of downloading and
	// uploading the binary to Cumulocity. Only useful for public repositories
	// since devices must be able to fetch the URL.
	LinkOnly bool `yaml:"linkOnly" json:"linkOnly,omitempty" jsonschema:"description=Reference the asset download URL instead of uploading the binary (public repositories only)"`

	// Token for the GitHub API (falls back to GITHUB_TOKEN / GH_TOKEN env vars).
	// Supports ${VAR} expansion, e.g. token: ${MY_GITHUB_TOKEN}
	Token string `yaml:"token" json:"token,omitempty" jsonschema:"description=GitHub API token (falls back to GITHUB_TOKEN / GH_TOKEN env vars; supports ${VAR} expansion)"`
}

// SoftwareSpec syncs software packages into the software repository.
// Name, version, architecture and software type are parsed from filenames
// (same logic as the software-uploader example).
type SoftwareSpec struct {
	// NamePrefix is added to every parsed software name
	NamePrefix string `yaml:"namePrefix" json:"namePrefix,omitempty" jsonschema:"description=Prefix added to every parsed software name"`

	// Type forces the softwareType for all packages from this source
	Type string `yaml:"type" json:"type,omitempty" jsonschema:"description=Force the softwareType for all packages from this source (overrides auto-detection)"`

	// TypeMap maps filename glob patterns to software types, first match wins.
	// Example: ["*.bin=firmware", ".deb=custom-apt"]
	TypeMap []string `yaml:"typeMap" json:"typeMap,omitempty" jsonschema:"description=Map filename glob patterns to software types (pattern=softwaretype; first match wins),example=*.bin=firmware"`

	// Version overrides the parsed version (useful for single-file sources)
	Version string `yaml:"version" json:"version,omitempty" jsonschema:"description=Override the parsed version (useful for single-file sources)"`

	Source Source `yaml:"source" json:"source" jsonschema:"description=Where the software packages come from"`
}

// FirmwareSpec syncs firmware (OS images) into the firmware repository.
// Supports common build system outputs (Yocto, Buildroot, Rugix Bakery,
// SWUpdate, RAUC, Mender, ...) plus custom naming via versionPattern.
type FirmwareSpec struct {
	// Name overrides the parsed firmware name. When set, all files from the
	// source are uploaded as versions of this single firmware item.
	Name string `yaml:"name" json:"name,omitempty" jsonschema:"description=Override the parsed firmware name (all files become versions of this single firmware item)"`

	Description string `yaml:"description" json:"description,omitempty" jsonschema:"description=Description of the firmware item"`

	// DeviceType restricts the firmware to a device type (c8y_Filter.type).
	// Supports placeholders derived from each parsed artifact: {name},
	// {version} and {filename}, e.g. "linux-{name}".
	DeviceType string `yaml:"deviceType" json:"deviceType,omitempty" jsonschema:"description=Restrict the firmware to a device type (c8y_Filter.type); supports {name} {version} and {filename} placeholders derived from the artifact filename"`

	// Version overrides the version for all files (only sensible for
	// single-file sources). Defaults to: parsed from filename, then the
	// GitHub release tag.
	Version string `yaml:"version" json:"version,omitempty" jsonschema:"description=Override the version for all files (defaults to the version parsed from the filename and then the GitHub release tag)"`

	// VersionPattern is a regular expression with a single capture group used
	// to extract the version from the filename for custom naming schemes.
	// Example: 'myimage-(\d+\.\d+\.\d+)\.custom$'
	VersionPattern string `yaml:"versionPattern" json:"versionPattern,omitempty" jsonschema:"description=Regular expression with a single capture group to extract the version from the filename for custom naming schemes"`

	Source Source `yaml:"source" json:"source" jsonschema:"description=Where the firmware images come from"`
}

// ConfigurationSpec syncs configuration files into the configuration repository
type ConfigurationSpec struct {
	// Name of the configuration item (defaults to the filename)
	Name string `yaml:"name" json:"name,omitempty" jsonschema:"description=Name of the configuration item (defaults to the filename without extension)"`

	// ConfigurationType (e.g. "mosquitto.conf", "properties")
	ConfigurationType string `yaml:"configurationType" json:"configurationType" jsonschema:"description=Configuration type,example=mosquitto.conf,example=properties"`

	Description string `yaml:"description" json:"description,omitempty" jsonschema:"description=Description of the configuration item"`

	// DeviceType restricts the configuration to a device type
	DeviceType string `yaml:"deviceType" json:"deviceType,omitempty" jsonschema:"description=Restrict the configuration to a device type"`

	Source Source `yaml:"source" json:"source" jsonschema:"description=Where the configuration file comes from"`
}

// DeviceProfileSpec creates a device profile referencing firmware, software
// and configuration already present in the tenant (typically synced by the
// earlier sections of the same manifest).
type DeviceProfileSpec struct {
	Name       string `yaml:"name" json:"name" jsonschema:"description=Name of the device profile"`
	DeviceType string `yaml:"deviceType" json:"deviceType,omitempty" jsonschema:"description=Restrict the profile to a device type (c8y_Filter.type)"`

	Firmware      *ProfileFirmwareRef       `yaml:"firmware" json:"firmware,omitempty" jsonschema:"description=Firmware version included in the profile"`
	Software      []ProfileSoftwareRef      `yaml:"software" json:"software,omitempty" jsonschema:"description=Software versions included in the profile"`
	Configuration []ProfileConfigurationRef `yaml:"configuration" json:"configuration,omitempty" jsonschema:"description=Configuration items included in the profile"`
}

type ProfileFirmwareRef struct {
	Name    string `yaml:"name" json:"name" jsonschema:"description=Firmware name (must exist in the tenant)"`
	Version string `yaml:"version" json:"version" jsonschema:"description=Firmware version (must exist in the tenant)"`
}

type ProfileSoftwareRef struct {
	Name    string `yaml:"name" json:"name" jsonschema:"description=Software name (must exist in the tenant)"`
	Version string `yaml:"version" json:"version" jsonschema:"description=Software version (must exist in the tenant)"`
	Action  string `yaml:"action" json:"action,omitempty" jsonschema:"description=Software action (defaults to install),example=install"` // defaults to "install"
}

type ProfileConfigurationRef struct {
	Name string `yaml:"name" json:"name" jsonschema:"description=Configuration item name (must exist in the tenant)"`
	Type string `yaml:"type" json:"type,omitempty" jsonschema:"description=Configuration type of the referenced item"`
}

// LoadManifest reads a manifest file, expanding ${VAR} environment variable
// references before parsing so secrets (e.g. GitHub tokens) can be injected.
func LoadManifest(path string) (*Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	expanded := os.Expand(string(raw), func(name string) string {
		// Leave unknown references intact so they are easy to spot in errors
		if value, ok := os.LookupEnv(name); ok {
			return value
		}
		return "${" + name + "}"
	})

	manifest := &Manifest{}
	decoder := yaml.NewDecoder(strings.NewReader(expanded))
	// Strict parsing: unknown fields are errors so typos are caught early
	decoder.KnownFields(true)
	if err := decoder.Decode(manifest); err != nil {
		if errors.Is(err, io.EOF) {
			// Empty manifest
			return manifest, nil
		}
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return manifest, nil
}

// Validate checks the manifest for structural errors before any API calls
func (m *Manifest) Validate() error {
	var errs []string

	for i, opt := range m.TenantOptions {
		if opt.Category == "" || opt.Key == "" {
			errs = append(errs, fmt.Sprintf("tenantOptions[%d]: category and key are required", i))
		}
		if opt.Value != "" && opt.ValueFrom != nil {
			errs = append(errs, fmt.Sprintf("tenantOptions[%d]: value and valueFrom are mutually exclusive", i))
		}
		if opt.Value == "" && opt.ValueFrom == nil {
			errs = append(errs, fmt.Sprintf("tenantOptions[%d]: one of value or valueFrom is required", i))
		}
		if opt.ValueFrom != nil {
			refs := 0
			if opt.ValueFrom.Application != "" {
				refs++
			}
			if opt.ValueFrom.Device != "" {
				refs++
			}
			if refs != 1 {
				errs = append(errs, fmt.Sprintf("tenantOptions[%d]: valueFrom requires exactly one of: application, device", i))
			}
		}
	}
	for i, feature := range m.Features {
		if feature.Key == "" {
			errs = append(errs, fmt.Sprintf("features[%d]: key is required", i))
		}
	}
	for i, app := range m.Applications {
		if app.Name == "" {
			errs = append(errs, fmt.Sprintf("applications[%d]: name is required", i))
		}
		if app.Source != nil {
			if err := app.Source.Validate(); err != nil {
				errs = append(errs, fmt.Sprintf("applications[%d]: %v", i, err))
			}
		}
	}
	for i, sw := range m.Software {
		if err := sw.Source.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("software[%d]: %v", i, err))
		}
		for _, mapping := range sw.TypeMap {
			if !strings.Contains(mapping, "=") {
				errs = append(errs, fmt.Sprintf("software[%d]: invalid typeMap entry %q: expected pattern=softwaretype", i, mapping))
			}
		}
	}
	for i, fw := range m.Firmware {
		if err := fw.Source.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("firmware[%d]: %v", i, err))
		}
		if fw.Source.URL != "" && fw.Name == "" {
			errs = append(errs, fmt.Sprintf("firmware[%d]: name is required when using a url source", i))
		}
		if err := validatePlaceholders(fw.DeviceType); err != nil {
			errs = append(errs, fmt.Sprintf("firmware[%d].deviceType: %v", i, err))
		}
	}
	for i, cfg := range m.Configuration {
		if err := cfg.Source.Validate(); err != nil {
			errs = append(errs, fmt.Sprintf("configuration[%d]: %v", i, err))
		}
		if cfg.ConfigurationType == "" {
			errs = append(errs, fmt.Sprintf("configuration[%d]: configurationType is required", i))
		}
	}
	for i, profile := range m.DeviceProfiles {
		if profile.Name == "" {
			errs = append(errs, fmt.Sprintf("deviceProfiles[%d]: name is required", i))
		}
		if profile.Firmware != nil && (profile.Firmware.Name == "" || profile.Firmware.Version == "") {
			errs = append(errs, fmt.Sprintf("deviceProfiles[%d].firmware: name and version are required", i))
		}
		for j, sw := range profile.Software {
			if sw.Name == "" || sw.Version == "" {
				errs = append(errs, fmt.Sprintf("deviceProfiles[%d].software[%d]: name and version are required", i, j))
			}
		}
		for j, cfg := range profile.Configuration {
			if cfg.Name == "" {
				errs = append(errs, fmt.Sprintf("deviceProfiles[%d].configuration[%d]: name is required", i, j))
			}
		}
	}

	for stage, hooks := range map[string][]HookSpec{"pre": m.Hooks.Pre, "post": m.Hooks.Post} {
		for i, hook := range hooks {
			if hook.Run == "" {
				errs = append(errs, fmt.Sprintf("hooks.%s[%d]: run is required", stage, i))
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid manifest:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// placeholderPattern matches {placeholder} references in template strings
var placeholderPattern = regexp.MustCompile(`\{([a-zA-Z]+)\}`)

// validatePlaceholders checks a template string only references the
// placeholders derived from a parsed firmware artifact
func validatePlaceholders(template string) error {
	for _, match := range placeholderPattern.FindAllStringSubmatch(template, -1) {
		switch match[1] {
		case "name", "version", "filename":
		default:
			return fmt.Errorf("unknown placeholder {%s}: supported placeholders are {name}, {version} and {filename}", match[1])
		}
	}
	return nil
}

// TypeMappings converts the string typeMap entries into parser TypeMapping values
func (s *SoftwareSpec) TypeMappings() []TypeMapping {
	mappings := make([]TypeMapping, 0, len(s.TypeMap))
	for _, entry := range s.TypeMap {
		idx := strings.LastIndex(entry, "=")
		if idx <= 0 || idx == len(entry)-1 {
			continue // validated in Manifest.Validate
		}
		mappings = append(mappings, TypeMapping{
			Pattern:      entry[:idx],
			SoftwareType: entry[idx+1:],
		})
	}
	return mappings
}
