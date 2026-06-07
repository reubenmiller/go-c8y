# c8ygen — OpenAPI → SDK generator

`c8ygen` generates the **Layer-0 substrate** of the go-c8y v2 SDK from the Cumulocity
OpenAPI specification, and reports drift between the spec and the hand-written SDK.

It implements Phases 0–1 of [docs/API_GEN.md](../../docs/API_GEN.md): generate the
mechanical, 1:1-with-OAS parts; leave the ergonomic API surface hand-written.

## Design in one paragraph

The SDK is intentionally **not** a 1:1 mirror of the API (resolver strings, ergonomic
option structs, `Upsert`, iterators, realtime). So the generator owns only what is
provably derivable from the spec — path constants and enums — emitting them into
`pkg/c8y/api/spec` as `zz_generated_*.go` files marked `DO NOT EDIT`. The hand-written
per-resource packages compose those constants. See [docs/API_GEN.md](../../docs/API_GEN.md)
for the full two-layer design and the planned later phases (option structs, façade models).

## Commands

```bash
# Generate path + enum constants from the vendored spec
task generate
#   └─ go run ./tools/c8ygen generate --spec docs/c8y-oas.yml --out pkg/c8y/api/spec

# Report drift between the OAS and the SDK source
task lint-api
#   └─ go run ./tools/c8ygen lint
# add --strict to exit non-zero on drift (for CI gating)
task lint-api -- --strict

# Download the latest spec into docs/c8y-oas.yml
task fetch-spec

# Generate directly from the latest upstream spec, without vendoring it first
go run ./tools/c8ygen generate --fetch
```

`go generate ./...` also works (directive lives in `pkg/c8y/api/spec/doc.go`).

### Flags

| Flag | Commands | Default | Meaning |
|---|---|---|---|
| `--spec` | generate, lint | `docs/c8y-oas.yml` | Local spec file |
| `--fetch` | generate, lint | `false` | Download the latest spec instead of `--spec` |
| `--url` | generate, lint, fetch | `https://cumulocity.com/api/core/dist/c8y-oas.yml` | Remote spec URL |
| `--out` | generate | `pkg/c8y/api/spec` | Output directory |
| `--src` | lint | `pkg/c8y/api` | SDK tree to scan for path literals |
| `--strict` | lint | `false` | Exit non-zero when drift is found |

## Layout

| File | Role |
|---|---|
| `main.go` | CLI dispatch and flags |
| `spec.go` | Spec loading (file/URL) + minimal OAS model + YAML parsing |
| `ident.go` | Path/word → Go identifier naming (initialisms, camel/underscore split) |
| `resolve.go` | `$ref` resolution + schema → Go type mapping |
| `generate.go` | Central path/enum extraction + emission (→ `pkg/c8y/api/spec`) |
| `resources.go` | Per-resource registry (the in-code precursor to the `x-c8y-sdk-*` overlay) |
| `resources_gen.go` | Per-resource option-struct + façade-model emission |
| `lint.go` | Drift detection (scan SDK literals, compare to OAS) |

Dependencies are deliberately minimal: standard library + `gopkg.in/yaml.v3` (already a
SDK dependency). No heavy OpenAPI toolkit is pulled into the module.

## Per-resource generation (Phases 2–3)

`c8ygen resources` reads the **SDK overlay** ([docs/c8y-oas.overlay.yml](../../docs/c8y-oas.overlay.yml))
and generates, for each resource declared there:

- the full option struct (query params, with type/doc overrides for divergences like the
  `Source` resolver field) into `pkg/c8y/api/<pkg>/zz_generated_options.go`, and
- façade accessors for the response schema into `pkg/c8y/jsonmodels/zz_generated_<schema>.go`.

`task generate` runs both `generate` and `resources`. **Adding a resource is a docs
change**: append an entry to the overlay, run `task generate`, and delete the superseded
hand-written struct/accessors (keep the resolver field, nested-object accessors like
`SourceID`, and constructors). The overlay is kept separate from `c8y-oas.yml` so it
survives `task fetch-spec`.

Currently migrated: `alarms`, `events`. See [docs/API_GEN.md](../../docs/API_GEN.md) §8
for the design finding on why option structs are generated whole rather than embedded.

## Adding later phases

Request-body structs (`CreateOptions`) need `allOf` flattening; grow `resolve.go` or
switch parsing to a dedicated library. The emission templates and the
`zz_generated_*.go` / DO-NOT-EDIT contract stay the same.
