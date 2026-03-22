# Software Package Uploader

A CLI tool for bulk uploading software packages to Cumulocity IoT. It recursively searches directories for software artifacts, intelligently parses filenames to extract names and versions, and uploads them with concurrent processing and progress tracking.

## Features

- 🔍 **Smart Filename Parsing**: Automatically extracts software name, version, and architecture from filenames
  - Extension-specific parsers for Debian (.deb), RPM (.rpm), Alpine (.apk), OpenWrt/Yocto (.ipk), Arch Linux (.pkg.tar.zst/.pkg.tar.xz), and archives (.tar.gz)
  - Supports complex version schemes with `~`, `+`, `-` (e.g., `1.6.2~584+gd629c53`)
  - Auto-detects architecture (arm64, amd64, aarch64, x86_64, noarch, any, etc.)
- 🏗️ **Architecture + Type Grouping**: Creates separate software items per architecture *and* package type (e.g. `noarch/rpm` and `noarch/apk` produce distinct items)
- 🗺️ **Custom Type Mappings**: Map filename glob patterns to software types with `--type-map` (e.g. `*.bin=firmware`)
- 🚀 **Concurrent Uploads**: Processes multiple version uploads simultaneously for better performance
- 📊 **Progress Tracking**: Real-time progress bars showing upload status
- 🎯 **Efficient Deduplication**: Groups files by software name and architecture to minimize API calls
- 🔄 **Idempotent**: Uses GetOrCreate pattern - safe to run multiple times
- � **Force Replacement**: Optional --force mode to replace existing version binaries
- 🛡️ **Error Handling**: Graceful error handling with detailed reporting
- 🔧 **Auto Type Detection**: Automatically detects software type from file extension (.deb→apt, .rpm→rpm, etc.)
- 🐛 **Debug Mode**: HTTP request/response logging for troubleshooting

## Installation

```bash
cd examples/software-uploader
go build -o software-uploader
```

## Usage

### Basic Usage

```bash
# Upload all .tar.gz files from a directory
./software-uploader --dir ./releases --pattern "*.tar.gz"

# Upload multiple file types using multiple --pattern flags
./software-uploader --dir ./artifacts --pattern "*.tar.gz" --pattern "*.zip" --pattern "*.bin"

# Upload all Debian packages
./software-uploader --dir ./releases --pattern "*.deb"

# Upload Linux packages (Debian, RPM, Alpine)
./software-uploader --dir ./releases --pattern "*.deb" --pattern "*.rpm" --pattern "*.apk"

# Dry run to preview what would be uploaded
./software-uploader --dir ./releases --pattern "*.tar.gz" --pattern "*.zip" --dry-run

# CI/CD usage (progress bar automatically disabled in non-TTY environments)
./software-uploader --dir ./dist --pattern "*.deb"

# Explicitly disable progress bar
./software-uploader --dir ./releases --pattern "*.deb" --no-progress
```

### CI/CD Integration

The tool automatically detects when running in a CI/CD environment (non-TTY output) and disables the progress bar. This ensures clean, parseable log output in CI systems like GitHub Actions, GitLab CI, Jenkins, etc.

**Automatic behavior:**
- **Interactive terminal**: Progress bar is shown
- **CI/CD pipeline**: Progress bar is automatically hidden
- **Explicit control**: Use `--no-progress` to force disable

Example CI usage:
```bash
# GitHub Actions, GitLab CI, etc. - progress bar automatically disabled
./software-uploader --dir ./artifacts --pattern "*.deb" --verbose
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dir` | *(required)* | Directory to search for software packages |
| `--pattern` | `*` | Glob pattern for matching files (repeatable) |
| `--type` | | Force a software type for all packages (overrides all detection) |
| `--type-map` | | Map filename glob to software type, e.g. `*.bin=firmware` (repeatable, first match wins) |
| `--name-prefix` | | Prefix added to every parsed software name |
| `--concurrency` | `5` | Number of concurrent uploads (1–20) |
| `--dry-run` | | Preview what would be uploaded without actually uploading |
| `--verbose` | | Enable detailed logging |
| `--debug` | | Verbose logging + HTTP request/response details |
| `--force` | | Replace existing version binaries |
| `--no-progress` | | Disable progress bar (auto-disabled in non-TTY environments) |

## Filename Parsing Examples

The tool uses extension-specific parsers to accurately extract software name, version, and architecture:

### Supported Package Formats

| Format | Extension(s) | `softwareType` | Naming Convention |
|--------|-------------|----------------|-------------------|
| Debian | `.deb` | `apt` | `name_version_arch.deb` |
| RPM | `.rpm` | `rpm` | `name-version-release.arch.rpm` |
| Alpine | `.apk` | `apk` | `name_version-release_arch.apk` |
| OpenWrt/Yocto | `.ipk` | `ipk` | `name_version_arch.ipk` |
| Arch Linux | `.pkg.tar.zst`, `.pkg.tar.xz` | `pacman` | `name-version-pkgrel-arch.pkg.tar.zst` |
| Archives | `.tar.gz`, `.tgz`, `.tar.bz2`, `.tar.xz`, `.tar.zst`, `.zip`, `.7z`, `.rar` | `archive` | `name-version_arch.tar.gz` |
| Binary | `.bin` | `binary` | `name-version.bin` |
| Java | `.jar`, `.war`, `.ear` | `java` | |
| Snap | `.snap` | `snap` | |
| Flatpak | `.flatpak` | `flatpak` | |
| AppImage | `.appimage` | `appimage` | |
| Windows | `.exe`, `.msi` | `windows` | |
| macOS | `.dmg`, `.pkg` | `macos` | |

### Parsing Examples

| Filename | Name | Version | Arch | Type |
|----------|------|---------|------|------|
| `tedge-flows_1.6.2~584+gd629c53_arm64.deb` | tedge-flows | 1.6.2~584+gd629c53 | arm64 | apt |
| `tedge-mapper-thingsboard_0.0.1_all.deb` | tedge-mapper-thingsboard | 0.0.1 | all | apt |
| `tedge-flows-1.6.2~584+gd629c53-1.aarch64.rpm` | tedge-flows | 1.6.2~584+gd629c53 | aarch64 | rpm |
| `tedge-mapper-thingsboard-0.0.1-1.noarch.rpm` | tedge-mapper-thingsboard | 0.0.1 | noarch | rpm |
| `tedge-flows_1.6.2_rc584+gd629c53-r0_aarch64.apk` | tedge-flows | 1.6.2_rc584+gd629c53 | aarch64 | apk |
| `tedge-mapper-thingsboard_0.0.1_noarch.apk` | tedge-mapper-thingsboard | 0.0.1 | noarch | apk |
| `tedge_1.6.2~584+gd629c53_aarch64.ipk` | tedge | 1.6.2~584+gd629c53 | aarch64 | ipk |
| `pacman-6.0.2-6-x86_64.pkg.tar.zst` | pacman | 6.0.2 | x86_64 | pacman |
| `linux-firmware-20241210.b81c7f9-1-any.pkg.tar.xz` | linux-firmware | 20241210.b81c7f9 | any | pacman |
| `tedge_1.6.2-rc584+gd629c53_aarch64-unknown-linux-musl.tar.gz` | tedge | 1.6.2-rc584+gd629c53 | aarch64 | archive |
| `myapp-1.2.3.tar.gz` | myapp | 1.2.3 | | archive |
| `device-firmware_v2.0.1.bin` | device-firmware | 2.0.1 | | binary |

1. **Extension-Specific Parsing**: Uses specialized parsers for different package formats
   - **Debian (.deb)**: `name_version_architecture.deb` format
   - **RPM (.rpm)**: `name-version-release.architecture.rpm` format
   - **Alpine (.apk)**: `name_version-release_architecture.apk` format
   - **OpenWrt/Yocto (.ipk)**: `name_version_architecture.ipk` format (same convention as Debian)
   - **Arch Linux (.pkg.tar.zst/.pkg.tar.xz)**: `name-version-pkgrel-architecture.pkg.tar.zst` format
   - **Archives (.tar.gz, .tgz, etc.)**: Detects architecture patterns in filename
   - **Generic**: Falls back to pattern matching for other formats
2. **Version Detection**: Supports complex versioning including:
   - Semantic versioning (1.2.3)
   - Pre-release tags (1.0.0-beta.1, 1.6.2-rc584)
   - Build metadata (1.0.0+build.123, 1.6.2~584+gd629c53)
   - Tildes for Debian epochs (1.6.2~584)
3. **Architecture Detection**: Automatically identifies CPU architectures
   - Common: arm64, amd64, aarch64, x86_64, i386, armhf, noarch, all, any
   - Target triples: aarch64-unknown-linux-musl, x86_64-unknown-linux-gnu
4. **Grouping by Name + Architecture + Type**: Creates separate software items per unique (name, arch, type) combination
   - Example: `noarch/rpm` and `noarch/apk` produce **distinct** software items even when arch matches
   - Allows proper device targeting in Cumulocity
5. **Device Type Filtering**: Architecture-specific packages (e.g. `arm64`, `aarch64`, `x86_64`) set `c8y_Filter.type` so they are only offered to matching devices. Architecture-agnostic values (`all`, `noarch`, `any`) do **not** set a device type filter — the package is available to all devices.
5. **Type Auto-Detection**: Automatically assigns software type based on extension
   - .deb→apt, .rpm→rpm, .apk→apk, .ipk→ipk, .pkg.tar.zst/.pkg.tar.xz→pacman, .jar→java, etc.
6. **Custom Type Mappings**: Override auto-detected types with `--type-map` glob rules (see below)

## Example Output

```
🔍 Scanning directory: ./releases
📦 Found 27 files matching pattern(s): *.deb, *.rpm, *.apk

📋 Upload Plan:
  • tedge-mapper-thingsboard [all/apt]:    1 version(s) [0.0.1]
  • tedge-mapper-thingsboard [noarch/apk]: 1 version(s) [0.0.1]
  • tedge-mapper-thingsboard [noarch/rpm]: 1 version(s) [0.0.1]
  • tedge-flows [aarch64/apt]:             2 version(s) [1.6.2_rc584+gd629c53, 1.6.2~584+gd629c53]
  • tedge-flows [arm64/apt]:               1 version(s) [1.6.2~584+gd629c53]

📊 Summary: 5 software package(s), 6 version(s) total

✅ Successfully processed 6 version(s)
   📤 Newly uploaded: 4
   ♻️  Already existed: 2

⏱️  Total time: 5.1s

──────────────────────────────────────────────────
Total: 6 processed (4 new, 0 replaced, 2 existing), 0 failed
```

Each entry in the plan shows `name [arch/type]` so it's immediately clear which package format and architecture each software item targets. The upload statistics then tell you how many versions were new vs. already present.

### Custom Type Mappings

Use `--type-map pattern=softwaretype` to override auto-detected types. This is useful when filenames don't follow a standard convention, or when you want to assign a custom type like `firmware`.

```bash
# Tag all .bin files as firmware
./software-uploader --dir ./dist --type-map '*.bin=firmware'

# Multiple mappings — checked in order, first match wins
./software-uploader --dir ./dist \
  --type-map '*.bin=firmware' \
  --type-map 'tedge-mapper-*=plugin'

# Bare extension shorthand (equivalent to *.deb)
./software-uploader --dir ./dist --type-map '.deb=custom-apt'
```

**Priority order** (highest to lowest):
1. `--type` — forces a type for all packages, overrides everything
2. `--type-map` — per-pattern rules, first match wins
3. Parser auto-detection (based on file extension)

### Force Replacement Mode

Use the `--force` flag to replace existing version binaries. This is useful when you've rebuilt packages and want to update them in Cumulocity:

```bash
./software-uploader --dir ./releases --pattern "*.deb" --force
```

With `--force`, the tool will:
1. Check if each version already exists
2. If it exists, delete the old binary from Cumulocity
3. Upload the new binary file
4. Update the version to reference the new binary
5. Count it as "Replaced" in the statistics

**Example output with --force:**
```
✅ Successfully processed 9 version(s)
   🔄 Replaced: 9

⏱️  Total time: 8.5s

──────────────────────────────────────────────────
Total: 9 processed (0 new, 9 replaced, 0 existing), 0 failed
```

### Verbose Logging

Use the `--verbose` flag to see detailed information about what's happening:

```bash
./software-uploader --dir ./releases --pattern "*.deb" --verbose
```

This will log:
- **Query strings** used to lookup existing software items
- **Software creation**: Whether each software item was newly created or already existed (with IDs)
- **Version uploads**: Whether each version was newly uploaded or already existed (with IDs)
- **Result details**: Status and metadata from each upload operation
- **Errors**: Detailed error messages if any operations fail

Example verbose output:
```
level=DEBUG msg="Looking up software with architecture" name=tedge-flows type=archive architecture=arm64 query="name eq 'tedge-flows' and softwareType eq 'archive' and deviceType eq 'arm64'"
level=INFO msg="Created new software item" id=123456 name=tedge-flows type=archive architecture=arm64
level=DEBUG msg="Uploading version" software_id=123456 version="1.6.2~584+gd629c53" file="tedge-flows_1.6.2~584+gd629c53_arm64.deb"
level=DEBUG msg="Version upload result details" software_id=123456 version_id=789012 version="1.6.2~584+gd629c53" status=Created meta_found=false http_status=201
level=INFO msg="Uploaded new software version" software_id=123456 version_id=789012 version="1.6.2~584+gd629c53" file="tedge-flows_1.6.2~584+gd629c53_arm64.deb"
```

### Debug Mode

Use the `--debug` flag to enable HTTP request/response logging for troubleshooting:

```bash
./software-uploader --dir ./releases --pattern "*.deb" --debug
```

This enables both verbose logging AND shows:
- Full HTTP request URLs and query parameters
- Request headers and body
- Response status codes and bodies
- Timing information for each API call

Use `--debug` when diagnosing issues with API calls, authentication, or query behavior.
```

### Verifying Version Statistics

The version statistics are determined by the upload operation:
- **Newly uploaded** (`meta_found=false`): Version was newly created
- **Replaced** (--force mode): Version existed but binary was replaced with a new one
- **Already existed** (`meta_found=true`): Version already existed and was not modified

To verify the statistics are accurate, run with `--verbose` and check the `meta_found` values in the debug logs. This will show you exactly which versions were created vs found for each file.

With `--force` mode, existing versions will always be replaced, regardless of the `meta_found` value.

## Identifying Uploaded Software

All software items uploaded by this tool are marked with a `c8y_SoftwareUploader` fragment containing metadata:

```json
{
  "c8y_SoftwareUploader": {
    "uploadedAt": "2026-02-06T10:30:00Z",
    "tool": "software-uploader"
  }
}
```

### Querying Software Uploaded by This Tool

You can query for all software uploaded by this tool using the Inventory API:

```bash
# Using c8y CLI
c8y inventory list --query "has(c8y_SoftwareUploader) and type eq 'c8y_Software'"

# In the Cumulocity UI
# Navigate to: Administration → Management → Software Repository
# Filter: has(c8y_SoftwareUploader)
```

This makes it easy to:
- Identify which software was uploaded by this tool vs manually created
- Track when software was last updated (`uploadedAt` timestamp)
- Clean up or manage bulk-uploaded software
- Audit automated uploads

## Integration with SDK

The core upload logic in `uploader.go` is designed to be SDK-portable. The main functions can be extracted and integrated into the go-c8y SDK's repository package:

- `ParseSoftwareFromFilename()`: Filename parsing logic
- `GroupBySOFTWARE()`: Consolidation logic
- `UploadSoftwareVersions()`: Concurrent upload orchestration

## Architecture

```
┌─────────────┐
│   main.go   │  CLI interface, flags, progress UI
└──────┬──────┘
       │
       ┌──────────┐
       │ parser.go │  Filename parsing & validation
       └────┬─────┘
            │
       ┌────────────┐
       │uploader.go │  Core business logic, SDK integration
       └────────────┘
            │
       ┌────────────┐
       │  go-c8y SDK│  API calls, authentication
       └────────────┘
```

## Error Handling

The tool provides detailed error reporting:

- File access errors
- Authentication failures
- Upload failures with specific version information
- Network errors with retry suggestions

Failed uploads are logged with context to help troubleshooting:

```
❌ Failed uploads:
  - myapp v1.2.0: upload failed: connection timeout
  - backend-service v2.1.0: file not readable: permission denied
```

## Performance

- **Concurrent Uploads**: Default 5 workers, configurable up to 20
- **Memory Efficient**: Streams file content, doesn't load entire files into memory
- **Progress Tracking**: Minimal overhead with efficient progress updates

## Future Enhancements

- Support for checksums/signatures
- Metadata extraction from sidecar files
- Custom naming patterns via templates
- Watch mode for continuous uploads
- Integration with CI/CD pipelines
