# Tenant Sync

A GitOps-style CLI tool for keeping Cumulocity IoT tenants in sync with a declarative manifest. Describe the desired tenant state in a YAML file — software, firmware and configuration repositories, tenant options, feature toggles, application subscriptions, device profiles — and apply it idempotently to one or more tenants.

It generalises the [software-uploader](../software-uploader/README.md) example: the same filename parsing and idempotent upload logic, extended to firmware and configuration, with artifact sources that can be local files **or GitHub releases**, all driven from a single configuration file.

## Concept

```
                     ┌──────────────────┐
  manifest.yaml ───▶ │   tenant-sync    │ ───▶ Tenant A (C8Y_* env)
                     │                  │
  sources:           │  resolve sources │      • software repository
   • local files     │  parse artifacts │      • firmware repository
   • GitHub releases │  diff + apply    │      • configuration repository
   • external URLs   │  (idempotent)    │      • device profiles
                     └──────────────────┘      • tenant options
                                               • feature toggles
                                               • application subscriptions
```

Because every operation is idempotent (get-or-create / upsert), the same manifest can be applied repeatedly and to multiple tenants — keep it in git, run it from CI, and your tenants stay in sync with the source of truth.

## Features

- 📄 **Declarative manifest**: one YAML file describes the desired tenant state
- 🔁 **Idempotent**: safe to run repeatedly; unchanged items are left alone
- 📦 **Software repository sync**: smart filename parsing for .deb, .rpm, .apk, .ipk, archives, etc. (same parser as software-uploader), grouped by name + architecture + type
- 🧱 **Firmware repository sync**: understands OS image naming from common build systems
  - **Yocto/OpenEmbedded**: `core-image-minimal-raspberrypi4-64-20240115103000.rootfs.wic.bz2`
  - **Buildroot**: `*.img`, `*.img.gz`, `sdcard.img`, `*.ext4`, `*.squashfs`
  - **Rugix Bakery**: `*.img.xz`, `*.rugixb`
  - **Update frameworks**: `*.swu` (SWUpdate), `*.raucb` (RAUC), `*.mender`
  - **Custom types**: any extension, with a `versionPattern` regex for custom naming schemes
- ⚙️ **Configuration repository sync**: upload configuration files with type and device type filters
- 🐙 **GitHub sources**: point any entry at a GitHub repository — assets are pulled from releases (`latest`, a specific tag, or `all`), with the release tag as the version fallback; `linkOnly` mode references download URLs without re-hosting binaries
- 🧩 **Device profiles**: declare firmware/software/configuration bundles; binary URLs are resolved from the tenant automatically
- 🎛️ **Tenant options & feature toggles**: set options, enable/disable features, subscribe/unsubscribe applications
- 🏢 **Target tenants**: apply one manifest to all child tenants, an explicit list, or tenants matching a selector — with per-tenant credentials resolved automatically
- 🔎 **Dry-run**: preview every change before applying
- 🔐 **Env var expansion**: `${VAR}` references in the manifest (e.g. GitHub tokens)

## Installation

```bash
git clone https://github.com/reubenmiller/go-c8y.git
cd go-c8y/examples/tenant-sync
go build -o tenant-sync
```

## Usage

The CLI follows the usual init → validate → run workflow:

```bash
# 1. Create a manifest (minimal skeleton; use --full for the annotated example)
./tenant-sync init

# 2. Validate it (schema only; --check-sources also resolves every source)
./tenant-sync validate tenant.yaml
./tenant-sync validate tenant.yaml --check-sources

# 3. Preview, then apply
./tenant-sync run -f tenant.yaml --dry-run
./tenant-sync run -f tenant.yaml
```

Authentication (for `run` only — `init` and `validate` work offline) uses the standard `C8Y_*` environment variables (`C8Y_BASEURL` / `C8Y_HOST`, `C8Y_TENANT`, `C8Y_USERNAME` / `C8Y_USER`, `C8Y_PASSWORD`), e.g. via a [go-c8y-cli](https://goc8ycli.netlify.app/) session:

```bash
# Only sync specific sections
./tenant-sync run -f tenant.yaml --only firmware,deviceProfiles

# Replace existing version binaries (instead of skipping existing versions)
./tenant-sync run -f tenant.yaml --force

# Sync multiple tenants from the same manifest (one session per tenant) ...
set-session tenant-a && ./tenant-sync run -f tenant.yaml
set-session tenant-b && ./tenant-sync run -f tenant.yaml

# ... or apply to other tenants in one run from the parent tenant
# (see "Target tenants" below)
./tenant-sync run -f tenant.yaml --all-children
```

### Commands

| Command | Description |
|---------|-------------|
| `init [path]` | Create a new manifest (default: `tenant.yaml`). `--full` writes the fully annotated example, `--force` overwrites |
| `validate [path]` | Validate the manifest schema (strict — unknown fields are errors). `--check-sources` also resolves local paths and GitHub releases |
| `run [path]` | Apply the manifest to the tenant (alias: `apply`) |
| `schema` | Generate the JSON schema for the manifest (`-o file` to write to a file) |

### Run flags

| Flag | Default | Description |
|------|---------|-------------|
| `-f`, `--manifest` | `tenant.yaml` | Path to the manifest file (also accepted as a positional argument) |
| `--dry-run`, `--dry` | | Preview changes without applying them |
| `--only` | *(all)* | Comma-separated sections to apply: `tenantOptions`, `features`, `applications`, `software`, `firmware`, `configuration`, `deviceProfiles`, `commands` |
| `--force` | | Replace existing version binaries / configuration files |
| `--concurrency` | `5` | Concurrent software version uploads (1–20) |
| `--target` | | Apply to a tenant referenced by ID or domain; repeatable or comma-separated (overrides the manifest `targets` section) |
| `--all-children` | | Apply to all child tenants of the current tenant (overrides the manifest `targets` section) |
| `--target-selector` | | Apply to tenants matching criteria, e.g. `domain=*.example.com,company=ACME` (overrides the manifest `targets` section) |
| `--include-current` | | Also include the current tenant when other target flags are used |
| `--credentials-mode` | | How credentials for other tenants are obtained: `serviceUser` or `sessions` (overrides the manifest) |
| `--verbose` | | Detailed logging |
| `--debug` | | Verbose logging + HTTP request/response details |

Sections are applied in a fixed order — tenant options, features, applications, software, firmware, configuration, device profiles, then commands — so profiles can reference repository items synced in the same run, and custom commands can build on everything the manifest declares.

## The Manifest

See [tenant.example.yaml](tenant.example.yaml) for a complete annotated example.

### IDE completion (JSON schema)

A JSON schema for the manifest is published at [tenant.schema.json](tenant.schema.json) and is generated from the Go types (`tenant-sync schema`). Reference it from the first line of your manifest to get code completion, inline documentation and validation in editors with YAML language server support (VS Code [YAML extension](https://marketplace.visualstudio.com/items?itemName=redhat.vscode-yaml), JetBrains, Neovim, ...):

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/reubenmiller/go-c8y/main/examples/tenant-sync/tenant.schema.json
```

Manifests created with `tenant-sync init` include this line already. To use a local copy instead (e.g. offline or pinned to your binary's version):

```bash
tenant-sync schema -o tenant.schema.json
```

```yaml
# yaml-language-server: $schema=./tenant.schema.json
```

The schema encodes the same rules as `tenant-sync validate`: unknown fields are rejected, required fields are enforced, and each `source` must set exactly one of `path`, `url` or `github`. A test keeps the committed schema file in sync with the Go types.

### Sources

Every repository entry (`software`, `firmware`, `configuration`) takes a `source` with exactly one of:

```yaml
source:
  # 1. Local file or directory (directories searched recursively with patterns)
  path: ./packages
  patterns: ["*.deb", "*.rpm"]
```

Relative paths are resolved against the manifest file's directory, so a manifest checked into a repository works no matter where the tool is invoked from.

```yaml
source:
  # 2. GitHub releases
  github:
    repo: owner/repo
    release: latest        # "latest" (default) | "latest-N" e.g. "latest-5" | "all" | a tag e.g. "v1.2.0"
    assets: ["*.wic.xz"]   # glob patterns against asset names
    includePrereleases: false
    linkOnly: false        # true: reference the download URL, don't upload the binary
    token: ${GITHUB_TOKEN} # optional; falls back to GITHUB_TOKEN / GH_TOKEN env vars
```

```yaml
source:
  # 3. External URL (referenced, not uploaded)
  url: https://example.com/files/config.toml
```

Any source can be marked `optional: true`: when the local path does not exist or nothing matches (no files, no release assets), the entry is reported as `skipped` instead of failing the run — useful when a build directory only exists on some pipelines.

With a GitHub source, `release: all` mirrors every release into the tenant, and `release: latest-5` keeps just the most recent five — pointing a firmware entry at an image repository is enough to populate the firmware version history:

```yaml
firmware:
  - name: tedge-rugix-pi
    deviceType: rugix-pi
    source:
      github:
        repo: thin-edge/tedge-rugix-image
        release: latest-5
        assets: ["*.img.xz"]
```

### Version resolution

For each artifact the version is determined by (first match wins):

1. `version` set explicitly on the manifest entry
2. Version parsed from the filename (package conventions for software; build-system conventions or `versionPattern` for firmware)
3. The GitHub release tag (with a leading `v` stripped)

Firmware entries with non-standard naming can extract versions with a regex:

```yaml
firmware:
  - deviceType: model7
    versionPattern: 'BUILD(\d+)'   # single capture group = version
    source:
      path: ./firmware
      patterns: ["FW_MODEL7_*.custom"]
```

The firmware `deviceType` supports placeholders derived from each parsed artifact — `{name}`, `{version}` and `{filename}` — so the device type can follow the artifact naming without listing every image explicitly:

```yaml
firmware:
  - deviceType: "linux-{name}"   # e.g. linux-core-image-tedge-rpi4
    source:
      path: ./tmp/deploy/images
      patterns: ["*.wic.bz2"]
```

### Tenant option lookups

Tenant option values can be resolved by reference at apply time instead of hardcoding IDs, using `valueFrom` with exactly one reference (`application` resolves a name to the application ID, `device` to a device managed object ID):

```yaml
tenantOptions:
  - category: application
    key: default.application
    valueFrom:
      application: devicemanagement
```

### Application subscriptions

Application entries normally just subscribe an existing application to the target tenant. `subscribed: false` declares the opposite — the application is unsubscribed (a missing application or subscription counts as the desired state, so the entry stays idempotent):

```yaml
applications:
  - name: advanced-software-mgmt
  - name: cloud-remote-access
    subscribed: false
```

> **Note**: (un)subscribing an application that the tenant does not own fails with `403 security/Forbidden` — such subscriptions are managed by the application's owner tenant. Run tenant-sync from the owner (parent) tenant instead, typically with a [targets section](#target-tenants) selecting the tenants to maintain.

With a `source`, the application is also created when missing (type defaults to `MICROSERVICE`, `contextPath` to the name) and its binary (a single zip) uploaded. Since binary content cannot be compared, the upload only happens on creation — or on every run with `--force`:

```yaml
applications:
  - name: my-microservice
    type: MICROSERVICE
    source:
      path: ./build/my-microservice.zip
```

### Target tenants

By default the manifest applies to the current tenant (the `C8Y_*` session). A `targets` section applies it to other tenants instead — run once from the parent/management tenant to keep a whole fleet of tenants in sync. The selection modes are additive (the union of all matches):

```yaml
targets:
  allChildren: true                    # every child tenant of the current tenant
  tenants: [t12345, child.example.com] # explicit, by ID or domain
  selector:
    domain: "*.iot.example.com"        # glob matched against the tenant domain
    company: ACME                      # exact company name
  current: true                        # also include the current tenant
                                       # (defaults to true only when nothing else is selected)
```

The selection can be overridden from the command line (any target flag replaces the manifest's selection entirely):

```bash
./tenant-sync run -f tenant.yaml --all-children
./tenant-sync run -f tenant.yaml --target t12345 --target child.example.com
./tenant-sync run -f tenant.yaml --target-selector "domain=*.iot.example.com" --include-current
```

Application subscriptions and feature toggles are managed directly from the parent (their APIs address the target tenant by ID). Everything else — tenant options, software/firmware/configuration repositories, device profiles, hooks — runs *inside* each target tenant and therefore needs per-tenant credentials, configured via `targets.credentials`:

```yaml
targets:
  allChildren: true
  credentials:
    mode: serviceUser        # serviceUser (default) | sessions
    application: tenant-sync # serviceUser mode: placeholder microservice name
    sessionHome: ""          # sessions mode: session dir (default $C8Y_SESSION_HOME, then ~/.cumulocity)
```

- **`serviceUser`** (default) uses the [microservice service user](https://goc8ycli.netlify.app/docs/tutorials/microservice-serviceuser/) pattern: a placeholder `MICROSERVICE` application (default name `tenant-sync`, no binary, no deployment) is created in the current tenant, subscribed to every target tenant, and its per-tenant service users are fetched via the application bootstrap user. The application and subscriptions are idempotent and left in place between runs. Service users for freshly subscribed tenants can take a few seconds to appear; the tool retries briefly.
- **`sessions`** reads [go-c8y-cli session files](https://goc8ycli.netlify.app/docs/gettingstarted/#sessions) and matches them to each target by tenant ID and then by domain. Encrypted sessions are not supported — the run fails with a clear error listing tenants without a matching session.

In multi-tenant runs every result line and the hook environment (`C8Y_TENANT`, `C8Y_USERNAME`, `C8Y_PASSWORD`) are scoped to the tenant being applied:

```
🏢 Tenant: t12345
  ✓ [applications] t12345: cloud-remote-access (unchanged): not subscribed
  ↻ [features] t12345: mqtt-service.smartrest (updated): enabled
```

### Hooks

Pre and post hooks run arbitrary commands around the sync — for example go-c8y-cli calls for anything outside the manifest's scope. Hooks execute via `sh -c` from the manifest directory with the current environment (including the `C8Y_*` session variables) passed through. A failing pre hook aborts the run; post hooks always run (also after section failures) and their failures are reported without aborting. Hooks run regardless of `--only`, and are only printed (not executed) in dry-run mode. In multi-tenant runs hooks execute once per target tenant, with `C8Y_TENANT`, `C8Y_USERNAME` and `C8Y_PASSWORD` overridden to that tenant's resolved credentials (and `C8Y_TOKEN` cleared).

```yaml
hooks:
  pre:
    - name: show session
      run: c8y currenttenant get --select name -o csv
  post:
    - run: c8y inventory list --query "has(c8y_TenantSync)" --select id,name -o csv
```

Hooks can also be scoped to a single section under `hooks.sections.<sectionName>` — they run immediately before/after that section, and (unlike the global hooks) are skipped when the section is excluded by `--only`. The same failure semantics apply: a failing section pre hook aborts the run, post hook failures are reported without aborting.

```yaml
hooks:
  sections:
    software:
      pre:
        - run: ./scripts/build-packages.sh
      post:
        - run: c8y software list --pageSize 100 -o csv
```

### Commands

The `commands` section runs named groups of arbitrary shell commands as the last section — for anything declarative sections don't cover (creating devices, registering external IDs, smoke tests, ...). The **actions within a group run in sequence** (a failing action skips the rest of its group, reported as `skipped`), while **separate groups run concurrently** with each other — so order-dependent steps go in one group, and independent work goes in separate groups:

```yaml
commands:
  - name: seed-devices
    actions:
      - c8y devices create --name "foo"
      - c8y devices create --name "foo2"

  # runs at the same time as seed-devices
  - name: smoke-test
    actions:
      - c8y currenttenant get --select name -o csv
```

Commands execute like hooks: via `sh -c` from the manifest directory, with the `C8Y_*` session variables passed through (scoped to the target tenant in multi-tenant runs), printed but not executed in dry-run mode. Unlike a failing pre hook, a failing action does not abort the run — it fails its group and is reflected in the summary and exit code.

### Device profiles

Device profiles bundle firmware, software and configuration. References are looked up in the tenant and resolved to binary URLs, so list the referenced items in the same manifest (or make sure they already exist):

```yaml
deviceProfiles:
  - name: rpi4-base
    deviceType: raspberrypi4-64
    firmware:
      name: core-image-tedge-raspberrypi4-64
      version: "20240115103000"
    software:
      - name: tedge
        version: 1.6.2
        action: install
    configuration:
      - name: mosquitto
        type: mosquitto.conf
```

## Example Output

```
📄 Manifest: tenant.yaml

  ✓ [tenantOptions] configuration/device.bootstrap.enabled (unchanged)
  ↻ [features] feature-branding (updated): enabled
  ✓ [applications] advanced-software-mgmt (unchanged): already subscribed
  ✚ [software] tedge [arm64/apt] (created)
  ✚ [software] tedge [arm64/apt] 1.6.2~584+gd629c53 (created): tedge_1.6.2~584+gd629c53_arm64.deb
  ✓ [firmware] core-image-tedge-rpi4 (unchanged)
  ✚ [firmware] core-image-tedge-rpi4 20240115103000 (created): core-image-tedge-rpi4-20240115103000.rootfs.wic.bz2
  ✚ [configuration] mosquitto (mosquitto.conf) (created): mosquitto.conf
  ↻ [deviceProfiles] rpi4-base (updated)

──────────────────────────────────────────────────
Total: 9 item(s) (5 created, 2 unchanged, 2 updated)
⏱️  Total time: 14.2s
```

## Identifying Synced Items

Managed objects created by this tool carry a `c8y_TenantSync` fragment:

```json
{
  "c8y_TenantSync": {
    "tool": "tenant-sync",
    "syncedAt": "2026-06-10T10:30:00Z"
  }
}
```

The fragment is passed to the SDK upserts as an **annotation**: it is written on create and on every real update, but it is excluded from change detection and never triggers an update by itself. Re-applying an unchanged manifest therefore performs no writes — `syncedAt` records when the desired state last actually changed, not when the tool last ran.

```bash
# List everything managed by tenant-sync
c8y inventory list --query "has(c8y_TenantSync)"
```

## CI/CD Integration

The manifest is designed to live in git next to your artifacts and run from CI:

```yaml
# GitHub Actions example
jobs:
  sync-tenant:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: go run github.com/reubenmiller/go-c8y/v2/examples/tenant-sync@latest run -f tenant.yaml
        env:
          C8Y_BASEURL: ${{ secrets.C8Y_BASEURL }}
          C8Y_TENANT: ${{ secrets.C8Y_TENANT }}
          C8Y_USERNAME: ${{ secrets.C8Y_USERNAME }}
          C8Y_PASSWORD: ${{ secrets.C8Y_PASSWORD }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Run the job per tenant (e.g. a matrix over credential sets) to maintain multiple tenants from the same source.

## Architecture

```
┌──────────────────────────────┐
│   main.go + cmd_{run,init,   │  CLI: subcommand dispatch
│   validate}.go               │
└──────┬───────────────────────┘
       │
┌──────────────┐    ┌──────────────┐
│ manifest.go  │    │  sources.go  │  local dirs, URLs, GitHub releases
└──────┬───────┘    └──────┬───────┘
       │                   │
┌──────────────────────────────────┐
│             sync.go              │  orchestrator, ordering, reporting
│  sync_tenant.go    (options,     │
│                     features,    │
│                     apps)        │
│  sync_software.go  + parser.go   │  package filename parsing
│  sync_firmware.go  + firmware.go │  OS image filename parsing
│  sync_configuration.go           │
│  sync_profiles.go                │  resolves references to binary URLs
└──────────────┬───────────────────┘
               │
        ┌────────────┐
        │  go-c8y SDK│  Repository.{Software,Firmware,Configuration},
        └────────────┘  Tenants.Options, Features, Applications, ManagedObjects
```

## Future Enhancements

- Users and groups: create users if missing, assign to groups/roles
- Prune mode: remove items present in the tenant but absent from the manifest
- Checksums/signatures for artifact integrity
- Additional sources (S3, OCI registries, GitLab releases)
- Watch mode for continuous reconciliation
