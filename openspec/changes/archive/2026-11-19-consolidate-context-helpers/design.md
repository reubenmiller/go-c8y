## Context

- `contexthelpers` defines the context keys and all setters/getters. Core, `op`, and services import it.
- `api` re-exports `WithDryRun`, `IsDryRun`, `WithMockResponses`, `IsMockResponses`, `WithRedactHeaders`, `ShouldRedactHeaders`, `WithDeferredExecution`, `IsDeferredExecution`, `WithTenant`, and adds `WithAuth` / `WithServiceUser` (forwarding to `authentication.WithAuth`).
- `context` re-exports the five processing-mode helpers. No package in the repo imports it.

## Goals / Non-Goals

**Goals:**
- One documented public package for request-scoped options.
- No public package named `context`.

**Non-Goals:**
- Changing the behavior of any option.
- Moving `authentication.WithAuth` (it is used below the `api` package).

## Decisions

**`api` is the public home.** Users already use `api.WithDryRun`, `api.WithAuth`, and `api.WithTenant` in docs and examples, and they already import `api` to create a client. Alternative: keep `contexthelpers` public and remove the `api` re-exports. Rejected because it adds an import for every user and its name describes the implementation, not the purpose.

**`contexthelpers` moves to `pkg/c8y/internal/contexthelpers`.** Placing it under `pkg/c8y/internal` lets both `pkg/c8y/op` and `pkg/c8y/api/...` import it, which `pkg/c8y/api/internal` would not allow for `op`.

**Getters stay public only where users need them.** `IsDryRun`, `IsDeferredExecution`, `IsMockResponses`, `ShouldRedactHeaders`, `TenantFromContext`, and `ProcessingMode` are useful in user middleware. Internal flags (`IsSkipTokenSource`, `IsSkipContextTokenExchange`, `ResolutionContext`) stay internal.

## Risks / Trade-offs

- [go-c8y-cli imports `contexthelpers` directly] → Check before moving; add any missing `api` re-exports it needs.
- [`api` package grows] → Group the context helpers in one file (`api/context_options.go`) with package-level godoc.

## Open Questions

- Should `api` also re-export `WithSkipTokenSource` for advanced users, or keep it internal?
