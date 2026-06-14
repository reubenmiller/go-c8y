# Proposal: invert CLI code generation — the command tree is the source of truth

Status: accepted / in progress
Supersedes the forward half of [`CLI_CODE_GENERATION.md`](./CLI_CODE_GENERATION.md) (Step 3, "thin-command emitter")
Related: [`go-c8y-cli/docs/proposals/v2-integration.md`](../../../go-c8y-cli/docs/proposals/v2-integration.md), the `pkg/c8ystream` bridge, go-c8y v2 services

## TL;DR

Today both the Go CLI commands **and** the PowerShell module are emitted from one
REST-endpoint spec (`api/spec/json/*.json`). That spec was the source of truth
because every command *was* a REST request. It no longer is: commands are being
hand-written against go-c8y **v2 services** (`client.Devices.Get(ctx, ref, opt)`),
and the REST details (method, path, body property-mapping, `accept`) have moved
**into the SDK**. The spec now describes something the command no longer does.

We invert the pipeline:

- **The hand-written Cobra command tree becomes the single source of truth** for
  the CLI surface (flags, pipeline, validation, examples, output type).
- **Everything downstream is a projection of that tree** — PowerShell cmdlets,
  Pester tests, markdown docs, man pages, shell completions — produced by walking
  the live command tree in-process, never by reading the REST spec.

This is not speculative: `cmd/gen-docs` and shell completions **already** work this
way. The work is to bring `gen-powershell` (and `gen-tests`) onto the same model,
and to add a small, closed set of command/flag annotations so the tree is a
*complete* description of the CLI surface.

## Current state (what we are changing)

The CLI has four generators, all historically fed by the REST spec:

| Generator | Input today | Output | Target input |
|---|---|---|---|
| `cmd/gen-cli` | REST spec | `pkg/cmd/**/*.auto.go` (360 files) | — (retire as commands are hand-written) |
| `cmd/gen-powershell` | REST spec | PSc8y cmdlets + Pester tests (281) | **command tree** |
| `cmd/gen-tests` | REST spec | command tests | **command tree** |
| `cmd/gen-docs` | **command tree** ✅ | markdown + man pages | (already correct) |

`gen-docs` ([`cmd/gen-docs/main.go`](../../../go-c8y-cli/cmd/gen-docs/main.go)) builds the
real root command via `root.NewCmdRoot(factory, …)` and walks it
(`internal/docs/markdown.go` recurses `cmd.Commands()`, reads `cmd.Annotations`,
`cmd.NonInheritedFlags()`, `cmd.Example`). It never touches the spec. That is the
pattern we generalise.

### Why the spec is now the wrong source

When `devices` was promoted onto the `c8ystream.Runner`, the old `.auto.go` files
were deleted and the commands became ~30–230 line hand-written files
(`pkg/cmd/devices/{get,list,create,update,delete}`). A command now:

1. declares Cobra flags (name, type, default, description),
2. sets up iteration (`Runner.Input` / `InputFlag`),
3. builds a typed SDK options struct from the flags,
4. returns a `Call` closure that invokes a v2 service.

The REST request is gone from the command — it lives in the SDK. So a parallel
spec entry for `devices list` is now (a) redundant and (b) **stale**: the
hand-written `list` has flags (`--queryTemplate`, `--withLatestValues`,
`--paginationStrategy`, group resolution, …) that no longer match `devices.json`.
The spec-driven `Get-DeviceCollection.ps1` is already wrong; only a tree-driven
generator can produce the current cmdlet.

Crucially, the PowerShell layer **already** just shells out to the binary
(`… | c8y devices get $c8yargs | ConvertFrom-ClientOutput`), so it never needed
REST knowledge — only the CLI *surface*. That surface now lives in the command.

## Target architecture

Three layers; code generation flips from a *frontend compiler* (spec→command) to
*backend projections* (command→bindings):

```
  OpenAPI spec ──► tools/c8ygen ──► go-c8y-v2: Layer-0 substrate (zz_generated_* options/paths/enums)
   (REST lives                              +  hand-written ergonomic services   ◄── drift-gated (lint-api)
    HERE now)                                        │
                                                     ▼
  go-c8y-cli COMMAND TREE  ◄─── hand-written, thin (~30 lines via c8ystream.Runner)
   = SINGLE SOURCE OF TRUTH      flags → typed SDK options → service call
    for the CLI surface          self-describing: Cobra flags + a small annotation vocabulary
                                                     │
            ┌────────────────────────┬───────────────┴──────────┬─────────────────────┐
            ▼                        ▼                          ▼                     ▼
      gen-powershell            gen-tests                   gen-docs            shell completions
     (walk tree, emit PS)    (walk tree, Pester)         (already tree)        (Cobra, already tree)
```

- **Layer 1 — SDK (go-c8y-v2):** unchanged. OAS → generated Layer-0 substrate +
  hand-written services, with its own gating drift check (`tools/c8ygen lint`).
  This is the correct home for REST coupling. The CLI's `replace` directive
  already points at the local SDK.
- **Layer 2 — CLI commands (go-c8y-cli):** hand-written, thin, mapping flags →
  SDK options structs (whose `url`-tagged fields are themselves OAS-generated, so
  "map to go-c8y services" is literal). Each command carries its full surface as
  introspectable metadata.
- **Layer 3 — projections:** every binding walks the live tree in-process (the
  `gen-docs` bootstrap). None read the REST spec.

### Key consequence: projectors can flip *now*

`gen-docs` already walks `.auto.go`, `.manual.go`, and hand-written commands
uniformly — the tree does not care how a command was authored. So **rewriting
`gen-powershell` to walk the tree is independent of how fast commands are
migrated.** We do not have to finish the command migration to gain a correct,
single-source PowerShell module.

## The annotation vocabulary (enabling work)

For the tree to be a *complete* source of truth, three things that currently live
only in the spec (or are lost) need an introspectable home on the command. The
`flags`/`completion` helpers already set most of the surface (pipeline, aliases,
collection property, examples, help); we add three:

| Concern | Where today | New home | Helper |
|---|---|---|---|
| Validation set (enum) | completion *closure* only — not readable | flag annotation `c8y:validateSet` (`[]string`) | `completion.WithValidateSet` also sets the annotation |
| Output media-type / item-type (for `ConvertFrom-ClientOutput`) | spec `accept`/`collectionType` — dropped from new cmds | command annotations `c8y:output.accept` / `c8y:output.itemType` | new `flags.WithOutputType(accept, itemType)` |
| PowerShell cmdlet name | spec `alias.powershell` | command annotation `c8y:powershell.name` | new `flags.WithPowershellName(name)` |

Everything else the projectors need is already present:

- flag name / type / default / description — Cobra
- pipeline target + property + aliases — `valueFromPipeline` / `valueFromPipeline.data`
  (`flags.WithExtendedPipelineSupport`, `WithPipelineAliases`)
- collection property — `collectionProperty` (`flags.WithCollectionProperty`)
- semantic method (for confirm prompts / common param sets) — `semanticMethod`
- examples — `cmd.Example`; help — `cmd.Short` / `cmd.Long`

The HTTP **method** (used only to pick the PowerShell common parameter set
Get/Create/Update/Delete/Collection) is derived from the verb
(`get`/`list`→GET, `create`→POST, `update`→PUT, `delete`→DELETE), overridable by
the `semanticMethod` annotation. The "Collection" set is added when the command
has a `collectionProperty` annotation or a collection `accept` type.

The vocabulary is deliberately small and closed. Anything not expressible in it is
a signal that the command is doing something the projection cannot represent —
which is exactly when we want a human to notice.

## Implementation

A shared tree walk feeds every projection; PowerShell reuses the existing renderer:

1. `internal/clibuild` — extract the `gen-docs` factory bootstrap into one shared
   helper (`NewRootCommand()`), so every projector builds the tree the same way.
2. `internal/clisurface` — the **single, projection-neutral tree walk**
   (`Walk(root, filter)`) that extracts the CLI surface: per command its path,
   verb, examples, method, accept/item-type, collection property, powershell
   name, and flags (raw pflag type, pipeline, required, validate set). Every
   projector and the surface manifest consume this one walker, so they cannot
   diverge on what the tree contains.
3. `internal/codegen/powershell/fromtree.go` — `GenerateFromTree(root, filter)`
   maps each `clisurface.Command` to a **spec-shaped value** and feeds the
   **existing** `GenerateSpec`/`renderCmdlet`/`renderTest`. PowerShell-specific
   conventions (pflag-type → PS type, cmdlet-name derivation, usage cleanup) live
   here; the neutral extraction lives in `clisurface`.
4. `cmd/gen-powershell` — `--from-tree`, `--command <path>` (subtree filter).
5. `cmd/gen-surface` — dumps `clisurface.Walk` as JSON: the stable surface
   manifest, the seam projectors and CI diffs consume.

Reusing the renderer via a synthetic spec value means the PowerShell output format,
the `New-ClientArgument`/`ConvertFrom-ClientOutput` plumbing, and the test
templates are all unchanged — only the *source of the metadata* changes.

### Note on byte-for-byte parity

The Go port of the PowerShell generator (PR #670) reproduces the legacy
pwsh output byte-for-byte via golden tests. That is a **transitional** artifact.
Tree-sourced output will differ (different metadata source, different ordering,
no stale entries) — intentionally. Do not invest further in spec-based PS parity;
the spec golden tests stay green only until the spec path is retired.

## The 360 `.auto.go` commands

Counts today: **360** `.auto.go`, **98** `.manual.go`, **30** hand-written. Do not
hand-write 360 commands blindly:

- Hand-write the rich resources (devices, alarms, events, measurements, inventory,
  operations) on the `c8ystream.Runner` (~30 lines each).
- For the mechanical long-tail CRUD, the right leverage is a **new** scaffold
  generator whose input is the **SDK's generated options structs** (not the REST
  spec): `devices.ListOptions` fields with `url` tags + docs + enum types *are*
  the flag list. This keeps the "map to go-c8y services" principle — the SDK is
  the input — and emits a constructor + flag declarations + a CRUD `Run` body for
  the standard shapes.
- Retire `gen-cli` + the per-resource spec as parity lands. During migration the
  spec-generated and hand-written commands coexist; the projectors already handle
  both.

## Migration path

1. Lock the annotation vocabulary; apply it to the migrated `devices` commands. ✅ (this change)
2. Extract `internal/clibuild`; rewrite `gen-powershell` to walk the tree. ✅ (this change)
3. Validate: generate the `devices` subtree from the tree, build, and diff against
   the (stale) spec-generated cmdlets to confirm the tree version is current. ✅ (this change)
4. Add the surface manifest seam (`internal/clisurface` + `cmd/gen-surface`) — a
   single neutral tree walk emitting JSON, shared by every projector. ✅ (this change)
5. (next) Point `gen-tests` at `clisurface`; flip `task build-powershell` to the
   tree path once enough commands carry annotations; retire the spec path
   per-resource as commands migrate.
6. (next) Scaffold the long-tail CRUD commands from the SDK's generated options
   structs (input = SDK, not REST spec), then retire `gen-cli` + the spec.

## Risks / things to preserve

- **Dynamic completions** (device/group lookups via `WithDevice`) cannot become
  static PowerShell `ValidateSet`s — they were not static before either; those
  params stay `[object[]]`. No regression.
- **Output-type simplification**: in the v2 world the CLI emits JSON/NDJSON, so the
  vnd media-type matters less than the `.ps1xml` view name + `collectionProperty`.
  The annotation captures it verbatim for now; simplifying `ConvertFrom-ClientOutput`
  is a separate, isolated follow-up.
- **`.ps1xml` format-data** is hand-maintained per resource and copied by
  `module.go`; it is orthogonal to this change.
