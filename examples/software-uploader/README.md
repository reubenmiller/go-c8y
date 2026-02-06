# Software Package Uploader

A CLI tool for bulk uploading software packages to Cumulocity IoT. It recursively searches directories for software artifacts, intelligently parses filenames to extract names and versions, and uploads them with concurrent processing and progress tracking.

## Features

- 🔍 **Smart Filename Parsing**: Automatically extracts software name, version, and architecture from filenames
  - Extension-specific parsers for Debian (.deb), RPM (.rpm), Alpine (.apk), and archives (.tar.gz)
  - Supports complex version schemes with `~`, `+`, `-` (e.g., `1.6.2~584+gd629c53`)
  - Auto-detects architecture (arm64, amd64, aarch64, x86_64, etc.)
- 🏗️ **Architecture Grouping**: Creates separate software items per architecture for proper device targeting
- 🚀 **Concurrent Uploads**: Processes multiple version uploads simultaneously for better performance
- 📊 **Progress Tracking**: Real-time progress bars showing upload status
- 🎯 **Efficient Deduplication**: Groups files by software name and architecture to minimize API calls
- 🔄 **Idempotent**: Uses GetOrCreate pattern - safe to run multiple times
- 🛡️ **Error Handling**: Graceful error handling with detailed reporting
- 🔧 **Auto Type Detection**: Automatically detects software type from file extension (.deb→apt, .rpm→rpm, etc.)

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
```

### Flags

- `--dir` (required): Directory to search for software packages
- `--pattern` (default: `"*"`): Glob pattern for matching files (can be specified multiple times)
- `--type` (optional): Software type to assign (e.g., "firmware", "application", "debian-package")
- `--concurrency` (default: `5`): Number of concurrent uploads (1-20)
- `--dry-run`: Preview what would be uploaded without actually uploading
- `--verbose`: Enable detailed logging to track software creation and version uploads

## Filename Parsing Examples

The tool uses extension-specific parsers to accurately extract software name, version, and architecture:

### Linux Package Formats

| Filename | Name | Version | Arch | Type |
|----------|------|---------|------|------|
| `tedge-flows_1.6.2~584+gd629c53_arm64.deb` | tedge-flows | 1.6.2~584+gd629c53 | arm64 | apt |
| `tedge-flows-1.6.2~584+gd629c53-1.aarch64.rpm` | tedge-flows | 1.6.2~584+gd629c53 | aarch64 | rpm |
| `tedge-flows_1.6.2_rc584+gd629c53-r0_aarch64.apk` | tedge-flows | 1.6.2_rc584+gd629c53 | aarch64 | apk |
| `tedge_1.6.2-rc584+gd629c53_aarch64-unknown-linux-musl.tar.gz` | tedge | 1.6.2-rc584+gd629c53 | aarch64 | archive |

### Generic Formats

| Filename | Name | Version | Type |
|----------|------|---------|------|
| `myapp-1.2.3.tar.gz` | myapp | 1.2.3 | archive |
| `device-firmware_v2.0.1.bin` | device-firmware | 2.0.1 | binary |
| `com.example.app-3.4.5-beta.1.zip` | com.example.app | 3.4.5-beta.1 | archive |

1. **Extension-Specific Parsing**: Uses specialized parsers for different package formats
   - **Debian (.deb)**: `name_version_architecture.deb` format
   - **RPM (.rpm)**: `name-version-release.architecture.rpm` format
   - **Alpine (.apk)**: `name_version-release_architecture.apk` format
   - **Archives (.tar.gz)**: Detects architecture patterns in filename
   - **Generic**: Falls back to pattern matching for other formats
2. **Version Detection**: Supports complex versioning including:
   - Semantic versioning (1.2.3)
   - Pre-release tags (1.0.0-beta.1, 1.6.2-rc584)
   - Build metadata (1.0.0+build.123, 1.6.2~584+gd629c53)
   - Tildes for Debian epochs (1.6.2~584)
3. **Architecture Detection**: Automatically identifies CPU architectures
   - Common: arm64, amd64, aarch64, x86_64, i386, armhf
   - Target triples: aarch64-unknown-linux-musl, x86_64-unknown-linux-gnu
4. **Architecture Grouping**: Creates separate software items per architecture
   - Example: `tedge [arm64]` and `tedge [amd64]` are distinct software packages
   - Allows proper device targeting in Cumulocity
5. **Type Auto-Detection**: Automatically assigns software type based on extension
   - .deb → apt, .rpm → rpm, .apk → apk, .jar → java, etc.

## Example Output

```
🔍 Scanning directory: ./releases
📦 Found 27 files matching pattern(s): *.deb, *.rpm, *.apk

📋 Upload Plan:
  • tedge-flows [aarch64]: 2 version(s) [1.6.2_rc584+gd629c53, 1.6.2~584+gd629c53]
  • tedge-flows [arm64]: 1 version(s) [1.6.2~584+gd629c53]
  • tedge-agent [aarch64]: 2 version(s) [1.6.2_rc584+gd629c53, 1.6.2~584+gd629c53]
  • tedge-agent [arm64]: 1 version(s) [1.6.2~584+gd629c53]

📊 Summary: 18 software package(s), 27 version(s) total

✅ Successfully processed 27 version(s)
   📤 Newly uploaded: 15
   ♻️  Already existed: 12

⏱️  Total time: 12.3s

──────────────────────────────────────────────────
Total: 27 processed (15 new, 12 existing), 0 failed
```

The tool now tracks and displays:
- **Newly uploaded versions**: Versions that were uploaded to Cumulocity for the first time
- **Already existed**: Versions that were already present (skipped, idempotent operation)

This helps you understand on subsequent runs how many versions are new vs. already synchronized.

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

### Verifying Version Statistics

The "Newly uploaded" vs "Already existed" counts are determined by the `meta_found` field:
- **`meta_found=false`**: Version was newly created → counts as "Newly uploaded"
- **`meta_found=true`**: Version already existed → counts as "Already existed"

To verify the statistics are accurate, run with `--verbose` and check the `meta_found` values in the debug logs. This will show you exactly which versions were created vs found for each file.

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
