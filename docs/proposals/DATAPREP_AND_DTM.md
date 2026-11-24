# Proposal: Data Preparation and Digital Twin Manager services

> **Status:** Draft for review
> **Specs:**
> - Data Preparation: <https://cumulocity.com/api/data-preparation/dist/c8y-dataprep-oas.json> (base path `/service/dataprep`)
> - Digital Twin Manager (DTM): <https://cumulocity.com/api/dtm/dist/c8y-dtm-oas.json> (base path `/service/dtm`)
> - Rule package format: <https://cumulocity.com/docs/data-preparation/external-development/#rule-package-format>
>
> **Related:** [API_DESIGN.md](../API_DESIGN.md), [IMPLEMENTATION_PATTERNS.md](../IMPLEMENTATION_PATTERNS.md), [DEFERRED_EXECUTION.md](../DEFERRED_EXECUTION.md)

## 1. Goals

1. Cover both APIs completely, using the same shapes as the other v2 services: `op.Result[T]`, options structs, `B`-suffix request builders, `ListAll` iterators, and resolver strings.
2. Make the common workflows one call each: deploy a smart function, test it, model an asset, link a device's measurements to an asset.
3. Make every write safe to re-run. Re-applying the same desired state returns `StatusSkipped` (or `StatusOK`) and sends no write.
4. Support dry run and deferred execution on every method, including the composite ones, so go-c8y-cli can show a plan and ask before it changes anything.

Non-goals: building or bundling TypeScript (the rules template repository does that), and a CLI command design (separate proposal).

## 2. Client layout

```text
client.DataPreparation                 (pkg/c8y/api/dataprep)
├── Rules        List, Get, GetDraft, Exists, Deploy, WaitDeployed, Delete,
│                Apply, TestAndDeploy, Export
├── Tests        Run
└── Diagnostics  Download

client.DigitalTwin                     (pkg/c8y/api/dtm)
├── Assets        Get, GetByExternalID, List, ListAll, Count, Create, Update, Delete,
│                 GetOrCreate, Upsert, ResolveID
│   ├── SubAssets      List, ListAll, Create, GetOrCreate, Assign, Unassign
│   ├── Devices        List, ListAll, Assign, Unassign
│   └── LinkedSeries   List, ListAll, Count, Set, Delete, Link, Opposites,
│                      GetSource, SetSource, DeleteSource, SetLabel, Reconcile
├── Definitions
│   ├── Properties     Get, List, ListAll, Count, Tags, Create, Update, Delete,
│   │                  GetOrCreate, Upsert, Compose
│   ├── Assets         (same set, minus Compose)
│   ├── Measurements   (same set)
│   ├── Events         (same set)
│   └── Alarms         (same set)
└── Apply / Plan   declarative model sync (section 5.6)
```

Service names follow the existing convention (`client.DataHub`, `client.RemoteAccess`). `DigitalTwin` is used instead of `DTM` so the name explains itself in autocomplete; the package is still `dtm`.

## 3. Idempotency rules (applies to both services)

| Method | Already in desired state | Not found | Conflict |
|---|---|---|---|
| `Delete`, `DeleteSource` | n/a | `StatusSkipped`, `Err == nil` | returned as error |
| `Unassign` | n/a | `StatusSkipped` (server returns 422 "is not a child of object", see 9.2) | returned as error |
| `Assign` | `StatusCreated` (server returns 201 again and adds no duplicate, see 9.2) | error (the parent or child is missing) | n/a |
| `GetOrCreate` | `StatusOK` (no write) | `StatusCreated` | 409 resolved by a follow-up GET (`StatusOK`) |
| `Upsert` | `StatusSkipped` (no write) | `StatusCreated` | `StatusUpdated` |
| `Create`, `Update` | plain calls, no extra lookups | error | error |
| `Deploy` (PUT) | `Idempotent: true`, always sends | n/a | n/a |
| `Rules.Apply` | `StatusSkipped` (deployed rule matches, no write) | `StatusCreated` | n/a (`StatusUpdated` on change) |

- "Desired state" is compared with `op.DesiredStateMatches`, the same subset match the repository upserts use. Platform fields (`id`, `creationTime`, `lastUpdated`, `owner`, `self`) are ignored.
- Annotations (`...model.Fragment`) work as in the repository services: they are written on create and on every real update, and never trigger an update on their own.
- Composite methods (`GetOrCreate`, `Upsert`, `Link`, `Apply`, `TestAndDeploy`) are built with `op.Result[T]{}.WithExecutor(...).ExecuteOrDefer(ctx)`. With deferred execution they return without sending anything, and `Execute` runs the whole flow later.
- Because `Delete` succeeds when the resource is gone, `defer client.X.Delete(...)` is safe in tests and setup scripts, even when the test already removed the resource.

## 4. Data Preparation

### 4.1 What the API offers

| Endpoint | In public spec | SDK method |
|---|---|---|
| `GET /v1/rules` | No | `Rules.List` |
| `GET /v1/rules/{ruleName}/deployed` | No | `Rules.Get`, `Rules.Exists` |
| `GET /v1/rules/{ruleName}/draft` | No | `Rules.GetDraft` |
| `PUT /v1/rules/{ruleName}/deployed` (tar.gz body) | Yes | `Rules.Deploy` |
| `DELETE /v1/rules/{ruleName}` | Yes | `Rules.Delete` |
| `POST /v1/run-tests` | Yes | `Tests.Run` |
| `GET /v1/diagnostics` (zip) | Yes | `Diagnostics.Download` |

The three GET endpoints are not in the published OpenAPI spec (`/service/dataprep/v3/api-docs` doesn't list them either), but the Data Preparation application uses them, and they were checked on a tenant on 2026-11-24:

- `GET /v1/rules` returns an object keyed by rule name (not a paged array):

  ```json
  {
    "tiny-edge-events": {
      "config": {
        "description": "tiny-edge events to Cumulocity events",
        "disabled": false,
        "source": {"topicPattern": "te/device/*///e/*", "transport": "mqtt"},
        "tags": ["tiny-edge", "event"]
      },
      "deploymentStatus": {
        "deploymentDisabled": false,
        "deploymentInProgress": false,
        "deploymentVersion": "4e196f69…48ccba",
        "error": "",
        "time": "2026-11-24T00:11:25.400Z",
        "user": "…"
      }
    }
  }
  ```

- `GET /v1/rules/{ruleName}/deployed` returns the same `config` and `deploymentStatus`, plus `jsCode` (the bundled smart function) and `tests` (keyed by test name, with `inputs` and `expectedOutput`). A missing rule returns 404 with `Rule '<name>' not found`.
- `GET /v1/rules/{ruleName}/draft` has the same shape and holds the version being edited in the Data Preparation application.
- `GET /v1/rules/{ruleName}` returns 405, so `Get` uses `/deployed`.

Notes for the implementation:

- The server stores the package's `input` block as `config.source` and drops `smartFunctionFile`. The SDK maps between the two so users only see the package names (`Input`).
- Because these endpoints are undocumented, their doc comments say so, and the methods live behind the same service so they can be marked experimental without moving.

### 4.2 Rule packages

Users should not have to build a tar.gz by hand. `dataprep.RulePackage` is an in-memory package with three constructors:

```go
// From a folder laid out like the rules template repository's build output
// (data-prep.yaml, the bundled .js file, tests/*.yaml).
pkg, err := dataprep.PackageFromDir("./rules/temperature/dist")

// From an archive exported by the Data Preparation application.
pkg, err := dataprep.PackageFromArchive(file) // io.Reader

// From Go values.
pkg := dataprep.NewPackage(dataprep.RuleConfig{
    SmartFunctionFile: "transform.js",
    Description:       "Converts temperature readings to measurements",
    Tags:              []string{"sensor", "temperature"},
    Input: dataprep.Input{
        Transport:       dataprep.TransportMQTT,
        TopicPattern:    "sensors/temperature/*",
        ClientIDPattern: "*",
    },
}).
    WithSmartFunction(jsCode).
    WithTest("reading", dataprep.TestCase{
        Inputs: []dataprep.TestInput{
            dataprep.JSONInput("sensors/temperature/s01", "s01", `{"temperature":25.5}`),
        },
        ExpectedOutput: []map[string]any{ /* optional */ },
    })
```

`RulePackage.Validate()` runs before any request is sent (also in dry run), and checks the documented limits locally:

- `data-prep.yaml` and the smart function file are present
- at most 100 entries, 1 MB per file, 10 MB in total
- no `..` path segments, no duplicate test names
- `topicPattern` does not contain `**`, `+` or `#`
- test names are percent-encoded in filenames (`inlet/outlet` becomes `tests/inlet%2Foutlet.yaml`)
- YAML is written in the strict form the server expects (lowercase `true`/`false`, plain decimals)

So a bad package fails with a clear Go error instead of an HTTP 400 or 422.

### 4.3 Scenario: deploy a rule from CI

```go
pkg, err := dataprep.PackageFromDir("./dist/temperature")
if err != nil {
    return err
}

result := client.DataPreparation.Rules.Deploy(ctx, "temperature", dataprep.DeployOptions{
    Package: pkg,
    // Poll GET .../deployed until deploymentInProgress is false, then fail
    // if deploymentStatus.error is set. 0 = return right after the PUT.
    WaitTimeout: 30 * time.Second,
})
var depErr *dataprep.DeploymentError
switch {
case errors.As(result.Err, &depErr):
    // The upload was accepted but the engine could not deploy the rule
    fmt.Println("deployment failed:", depErr.Message, depErr.Version)
    return result.Err
case result.Err != nil:
    // 400/413/422 come back as *core.Error with the server message
    return result.Err
}
rule := result.Data // jsonmodels.DataPrepRule, read back after the wait
fmt.Println(result.Status, rule.DeploymentVersion())
// Status: Created (rule did not exist) or Updated (overwritten)
```

`DeployOptions` also accepts `Archive io.Reader` for a prebuilt tar.gz. Exactly one of `Package` or `Archive` must be set.

`Created` versus `Updated` is known because `Deploy` checks `Exists` first. That check is one extra GET; set `DeployOptions.SkipExistsCheck: true` to drop it, and the status is then always `Updated`.

### 4.4 Scenario: run the package's tests, then deploy only if they pass

```go
result := client.DataPreparation.Rules.TestAndDeploy(ctx, "temperature", dataprep.TestAndDeployOptions{
    Package: pkg,
    // Compare outputs with each test's expectedOutput (client side).
    // Without expectedOutput a test passes when onMessage does not throw.
    CompareExpected: true,
})

var testErr *dataprep.TestFailedError
switch {
case errors.As(result.Err, &testErr):
    for _, f := range testErr.Failures {
        fmt.Printf("%s[%d]: %s\n", f.Test, f.InputIndex, f.Reason)
        for _, line := range f.Logs {
            fmt.Println("  log:", line)
        }
    }
    return result.Err // nothing was deployed
case result.Err != nil:
    return result.Err
}
fmt.Println("deployed, all tests passed")
```

### 4.5 Scenario: test a smart function while developing it

No package is needed. The input helpers set `payloadFormat` and default `time` to now.

```go
res := client.DataPreparation.Tests.Run(ctx, dataprep.RunTestsOptions{
    JSCode: `export function onMessage(msg, ctx) { /* ... */ }`,
    Config: map[string]any{"unit": "°C"},
    Tests: map[string]dataprep.TestCase{
        "json": {Inputs: []dataprep.TestInput{
            dataprep.JSONInput("dev/d1/data", "d1", `{"temp":22.5}`),
        }},
        "csv": {Inputs: []dataprep.TestInput{
            dataprep.TextInput("dev/d1/csv", "d1", "temp|hum\n22.5|45\n"),
        }},
        "cbor": {Inputs: []dataprep.TestInput{
            dataprep.BinaryInput("dev/d1/bin", "d1", cborBytes), // base64-encoded for you
        }},
    },
})
if res.Err != nil {
    return res.Err
}

report := res.Data // jsonmodels.DataPrepTestResults
for name, inputs := range report.All() {
    for i, r := range inputs {
        if r.Error() != nil {
            fmt.Printf("%s[%d] threw: %s\n%s\n", name, i, r.Error().Message, r.Error().Stack)
            continue
        }
        for out := range r.Outputs() {
            fmt.Println(out.CumulocityType(), out.Payload().String())
        }
    }
}

// Shortcuts
report.Passed()            // true when no input threw
report.Failures()          // iter.Seq of failing (test, index, result)
report.Compare(expected)   // client-side diff against expected outputs
```

A package can be tested without deploying it: `Tests.Run(ctx, dataprep.RunTestsOptions{Package: pkg})` reads the code, config and tests from the package.

### 4.6 Scenario: remove a rule (idempotent, safe in `defer`)

```go
result := client.DataPreparation.Rules.Delete(ctx, "temperature")
// result.Status: NoContent (deleted) or Skipped (did not exist)
```

In integration tests:

```go
func TestTemperatureRule(t *testing.T) {
    name := "test-temperature-" + testutil.RandomSuffix()
    require.NoError(t, client.DataPreparation.Rules.Deploy(ctx, name, dataprep.DeployOptions{Package: pkg}).Err)
    defer client.DataPreparation.Rules.Delete(context.Background(), name)

    // publish MQTT messages and assert on measurements...
}
```

### 4.7 Scenario: deploy only when something changed (CI, GitOps)

`Deploy` always sends the PUT (safe, it overwrites). `Apply` reads the deployed rule first and only deploys when the package differs from it:

```go
result := client.DataPreparation.Rules.Apply(ctx, "temperature", dataprep.ApplyOptions{
    Package:     pkg,
    WaitTimeout: 30 * time.Second,
})
switch result.Status {
case op.StatusCreated: // rule did not exist
case op.StatusUpdated: // deployed, Meta["changed"] lists what differed
case op.StatusSkipped: // deployed rule already matches, nothing sent
case op.StatusFailed:
    return result.Err
}
fmt.Println(result.Meta["changed"]) // e.g. [jsCode tests.identify config.tags]
```

The comparison uses `GET .../deployed` and checks:

| Package | Deployed rule |
|---|---|
| smart function file contents | `jsCode` (exact string match) |
| `data-prep.yaml` `description`, `tags`, `disabled` | `config.description`, `config.tags`, `config.disabled` |
| `data-prep.yaml` `input` | `config.source` |
| `tests/*.yaml` | `tests` (by name, JSON-normalized) |

`ai-chat.yaml` is not compared: the server keeps the existing chat history when the package has none, so it is not part of the desired state. Optional fields left out of the package are treated as their server defaults (`disabled: false`, `transportID` = `transport`) so they don't count as a change.

`deploymentVersion` looks like a SHA-256, but it isn't documented what it is computed from, so the SDK does not try to reproduce it (open question 1).

A failed earlier deployment is handled as well: if the content matches but `deploymentStatus.error` is set, `Apply` deploys again instead of returning `Skipped`.

### 4.8 Scenario: list rules and check their deployment state

```go
rules := client.DataPreparation.Rules.List(ctx, dataprep.ListOptions{
    // Client-side filters; the endpoint returns all rules in one response
    Tags:   []string{"tiny-edge"},
    Name:   "tiny-edge-*", // wildcard match on the rule name
})
if rules.Err != nil {
    return rules.Err
}
for rule := range rules.Data.Items() {
    st := rule.DeploymentStatus()
    fmt.Printf("%-30s %-40s disabled=%v inProgress=%v error=%q\n",
        rule.Name(), rule.TopicPattern(), rule.Disabled(), st.InProgress(), st.Error())
}

// Rules that failed to deploy
for rule := range rules.Data.Items() {
    if rule.DeploymentStatus().Error() != "" {
        fmt.Println("broken:", rule.Name())
    }
}
```

`List` returns `op.Result[jsonmodels.DataPrepRuleCollection]`. The SDK turns the name-keyed object into an ordered list (sorted by name) with `Name()` filled in, so it reads like the other collections and works with `IterAs[T]()`. It does not page, because the endpoint does not page.

Read a single rule, including its code and tests:

```go
rule := client.DataPreparation.Rules.Get(ctx, "tiny-edge-events")
if api.IsNotFound(rule.Err) {
    fmt.Println("not deployed")
}
fmt.Println(rule.Data.JSCode())
for name, tc := range rule.Data.Tests() {
    fmt.Println(name, len(tc.Inputs))
}

// Just check it is there
exists, err := client.DataPreparation.Rules.Exists(ctx, "tiny-edge-events")

// What is being edited in the UI but not yet deployed
draft := client.DataPreparation.Rules.GetDraft(ctx, "tiny-edge-events")
```

### 4.9 Scenario: export a deployed rule into a package

Because `Get` returns the code, config and tests, a deployed rule can be turned back into a package. This lets you copy rules between tenants or bring a UI-authored rule into Git:

```go
pkg := client.DataPreparation.Rules.Export(ctx, "tiny-edge-events", dataprep.ExportOptions{
    Draft: false, // true exports the draft instead of the deployed rule
})
if pkg.Err != nil {
    return pkg.Err
}

// Write it out in the template repository layout...
err := pkg.Data.WriteDir("./rules/tiny-edge-events")

// ...or deploy it to another tenant, only if it differs there
other.DataPreparation.Rules.Apply(ctx, "tiny-edge-events", dataprep.ApplyOptions{Package: pkg.Data})
```

The exported smart function is written as `smartFunction.js`, the same name the Data Preparation application uses for its own exports.

### 4.10 Scenario: deferred execution before deleting

```go
prepared := client.DataPreparation.Rules.Delete(api.WithDeferredExecution(ctx, true), "temperature")
fmt.Println(prepared.Request.Method, prepared.Request.URL) // DELETE .../service/dataprep/v1/rules/temperature
if confirm("Delete rule temperature?") {
    if res := prepared.Execute(ctx); res.Err != nil {
        return res.Err
    }
}
```

For `Deploy`, `prepared.Request` has `Content-Type: application/gzip` and `Meta["package"]` lists the entries, so a CLI can show what will be uploaded without reading the archive.

With `Apply`, deferred execution still does the GET and the comparison, so the prepared result already says `Skipped` or lists `Meta["changed"]` before anything is written. A CLI can show "3 of 12 rules will change" and ask once.

### 4.11 Scenario: download diagnostics

```go
res := client.DataPreparation.Diagnostics.Download(ctx, dataprep.DiagnosticsOptions{
    // 429 means an archive is already being prepared; wait for Retry-After
    // and try again, up to this many times. 0 = no retry.
    RetryOnBusy: 3,
})
if res.Err != nil {
    return res.Err
}
defer res.Data.Close()
_, err := io.Copy(outFile, res.Data) // core.BinaryResponse, streamed
```

## 5. Digital Twin Manager

### 5.1 Asset references

Every method that takes an asset accepts a resolver string, like the device resolver:

| Reference | Lookup |
|---|---|
| `"12345"` | asset ID, no lookup |
| `"ext:pump-07"` | `GET /assets/externalIds/pump-07` (the DTM `c8y_ExternalId`, identity type `c8y_Asset`) |
| `"name:Pump 07"` | `GET /assets?query=$filter=(name eq 'Pump 07')`, must match exactly one asset |
| `"query:type eq 'Pump'"` | DTM `$filter` query, must match exactly one asset |

DTM external IDs have one value and a fixed identity type (`c8y_Asset`), so `ext:` takes one value here. Device arguments (`Devices.Assign`, `LinkedSeries.Link`) use the existing device resolver, so `ext:c8y_Serial:ABC123` still works for devices.

### 5.2 Scenario: create assets from an external system (idempotent sync)

The DTM API has a server-side upsert keyed by `c8y_ExternalId` (`POST /assets` with `X-Upsert-Mode: true`). The SDK wraps it and adds the desired-state check, so re-running a sync performs no writes:

```go
result := client.DigitalTwin.Assets.Upsert(ctx, dtm.AssetUpsertOptions{
    ExternalID: "pump-07",
    Body: map[string]any{
        "name":     "Pump 07",
        "type":     "Pump",
        "c8y_Notes": "North wing, level 2",
        "maxFlow":  120,
    },
}, model.Frag("c8y_ExternalAsset", map[string]any{
    "system":   "SAP",
    "syncedAt": time.Now().Format(time.RFC3339), // annotation, never triggers an update
}))

switch result.Status {
case op.StatusCreated: // 201
case op.StatusUpdated: // 200, something changed
case op.StatusSkipped: // already matched, no write sent
case op.StatusFailed:
    return result.Err
}
```

How it works: GET `/assets/externalIds/{id}`; if found and `DesiredStateMatches`, return `Skipped`; otherwise POST with `X-Upsert-Mode: true`. Set `AssetUpsertOptions.SkipCompare: true` to send the upsert directly (one request, never `Skipped`).

The client-side comparison is needed: on the tenant, a server upsert with an identical body still returned 200 and changed `lastUpdated`, so it always writes (see 9.2).

`GetOrCreate` uses the same key but never updates:

```go
asset := client.DigitalTwin.Assets.GetOrCreate(ctx, dtm.AssetCreateOptions{
    ExternalID: "site-berlin",
    Body:       map[string]any{"name": "Berlin site", "type": "Site"},
})
// StatusOK = existed, StatusCreated = new
```

### 5.3 Scenario: build an asset hierarchy

```go
site := client.DigitalTwin.Assets.GetOrCreate(ctx, dtm.AssetCreateOptions{
    ExternalID: "site-berlin",
    Body:       map[string]any{"name": "Berlin site", "type": "Site"},
})
if site.Err != nil {
    return site.Err
}

// Create a child and assign it in one call (POST /assets/{id}/subAssets).
// Idempotent through ExternalID: an existing child is assigned, not duplicated.
hall := client.DigitalTwin.Assets.SubAssets.GetOrCreate(ctx, "ext:site-berlin", dtm.AssetCreateOptions{
    ExternalID: "hall-1",
    Body:       map[string]any{"name": "Hall 1", "type": "Building"},
})

// Assign an existing asset. Safe to repeat: the server ignores a repeat.
client.DigitalTwin.Assets.SubAssets.Assign(ctx, "ext:hall-1", "ext:pump-07")

// Assign a device by name. Safe to repeat.
client.DigitalTwin.Assets.Devices.Assign(ctx, "ext:pump-07", "name:pump-07-controller")

// Unassign. Not assigned (server 422 "is not a child of object") -> StatusSkipped.
client.DigitalTwin.Assets.Devices.Unassign(ctx, "ext:pump-07", "name:pump-07-controller")

// Read the tree back
tree := client.DigitalTwin.Assets.Get(ctx, "ext:site-berlin", dtm.GetOptions{
    WithSubAssets:     true,
    WithChildrenCount: true,
})
for child := range tree.Data.SubAssets() {
    fmt.Println(child.ID(), child.Name())
}
```

### 5.4 Scenario: link device measurements to an asset

Linked series are keyed by `fragment` + `series`. `Link` resolves the device, sets the source, and (optionally) fetches the measurement type, in one call:

```go
res := client.DigitalTwin.Assets.LinkedSeries.Link(ctx, "ext:pump-07", dtm.LinkOptions{
    Fragment: "c8y_Flow",
    Series:   "F",
    Label:    "Flow rate",
    Device:   "ext:c8y_Serial:CTRL-0007", // device resolver
    // Asset series can have a different name from the device series:
    SourceFragment:      "c8y_FlowMeter",
    SourceSeries:        "rate",
    FetchTypeFromSource: true,
})
// StatusSkipped when the link already points to the same source with the same label
```

Several at once (one `POST /linkedSeries` request, add or update by key):

```go
client.DigitalTwin.Assets.LinkedSeries.Set(ctx, "ext:pump-07", []dtm.LinkedSeries{
    {Fragment: "c8y_Flow", Series: "F", Label: "Flow rate"},
    {Fragment: "c8y_Pressure", Series: "P", Label: "Outlet pressure"},
}, dtm.SetLinkedSeriesOptions{})
```

Find which assets a device feeds (for example before deleting the device):

```go
for link, err := range client.DigitalTwin.Assets.LinkedSeries.Opposites(ctx, "name:pump-07-controller", dtm.OppositesOptions{}).Items() {
    if err != nil {
        return err
    }
    fmt.Println(link.Asset().ID(), link.Fragment(), link.Series())
}
```

### 5.5 Scenario: define the data model (definitions)

The five definition types share one method set, keyed by `identifier`. Property definitions are keyed by `identifier` plus `contexts` (the API requires both for update and delete), so their options carry `Contexts`.

```go
props := client.DigitalTwin.Definitions.Properties

// Create or update a property definition. No write when it already matches.
res := props.Upsert(ctx, dtm.PropertyDefinition{
    Identifier: "maxFlow",
    Contexts:   []string{dtm.ContextAsset},
    Tags:       []string{"pump"},
    JSONSchema: map[string]any{
        "title":   "Maximum flow",
        "type":    "number",
        "minimum": 0,
    },
})

// Asset definition that uses it
assetDefs := client.DigitalTwin.Definitions.Assets
res = assetDefs.Upsert(ctx, dtm.AssetDefinition{
    Identifier: "Pump",
    Tags:       []string{"pump"},
    JSONSchema: map[string]any{"title": "Pump"},
    Composition: dtm.Composition{
        AllowedProperties: []dtm.AllowedProperty{{Identifier: "maxFlow", MinOccurs: "1"}},
        AllowedSubAssets:  []dtm.AllowedSubAsset{{Identifier: "Motor", MaxOccurs: "2"}},
    },
})
```

**Server-generated schema fields.** The server adds to the `jsonSchema` you send: `$schema`, `additionalProperties: false`, and for asset definitions `definitions`, `properties` and `required` built from `composition` (for measurement definitions, a required `source`). So:

- `Upsert` compares with a subset match (`op.DesiredStateMatches`): the desired schema only has to be contained in the stored one. Sending `{title: "Pump"}` again returns `Skipped`.
- For asset definitions, the structure belongs in `composition`, not in `jsonSchema.properties`. `Upsert` compares `title`, `description`, `tags` and `composition`.
- `PUT` replaces the whole `jsonSchema`. On the tenant this dropped the server-generated `source` from a measurement definition. `Update` and `Upsert` therefore send the stored schema merged with the desired one by default; `UpdateOptions.Replace: true` sends only what you give.

**Asset definition immutability.** The API description says an asset definition's schema can't change after creation, but on the tenant a PUT changed the title, description and composition, and the server rebuilt `properties` and `required` to match (see 9.2). The SDK does not add its own immutability check. Open question 6 asks the DTM team which behavior is intended.

**Unique titles.** Creating a property definition with a new identifier but an existing `title` fails with 409 ("Property Definition with title '…' already exists"). `Upsert` keys on `identifier`, so this case comes back as `StatusFailed` with `dtm.ErrDuplicateTitle` and the conflicting title, instead of a raw 409.

**Shared and library definitions.** Definitions that come from the enterprise tenant (`c8y_SharedDefinition`) or a library (`c8y_Origin`) are read-only. `Upsert` returns `StatusSkipped` if they already match, and `dtm.ErrReadOnlyDefinition` otherwise, instead of a raw 403.

Read helpers:

```go
// One definition (404 -> StatusNoMatch with IgnoreNotFound)
def := props.Get(ctx, "maxFlow", dtm.GetPropertyOptions{ApplicableTo: dtm.ContextAsset})

// Filtered list, all pages
for d, err := range props.ListAll(ctx, dtm.ListDefinitionsOptions{Tags: []string{"pump"}}).Items() { /* ... */ }

// Counts and tags
n := props.Count(ctx, dtm.ListDefinitionsOptions{Tags: []string{"pump"}})
tags := props.Tags(ctx, dtm.TagsOptions{Identifiers: []string{"maxFlow", "minFlow"}})

// JSON schema built from several property definitions
schema := props.Compose(ctx, []string{"maxFlow", "minFlow"})
```

### 5.6 Scenario: apply a whole model from files (declarative)

For GitOps-style workflows, `dtm.Model` holds definitions and, optionally, assets. `Apply` makes the tenant match it, in dependency order:

1. property, measurement, event and alarm definitions
2. asset definitions, each after the asset definitions its composition allows as sub-assets
3. assets, each after its parent when the parent is in the model
4. sub-asset assignments, device assignments, linked series
5. with `Prune`, deletes of definitions that are no longer in the model

A cycle (in compositions or in parents) is an error before anything is sent.

Model files are YAML or JSON. A file holds one or more objects (YAML documents separated by `---`, or a list), each with a `kind`:

```yaml
kind: PropertyDefinition
identifier: maxFlow
contexts: [asset]
jsonSchema: {title: Maximum flow, type: number}
---
kind: AssetDefinition
identifier: Pump
jsonSchema: {title: Pump}
composition:
  allowedProperties: [{identifier: maxFlow}]
---
kind: Asset
externalId: pump-07
parent: ext:site-berlin
asset: {name: Pump 07, type: Pump, maxFlow: 120}
devices: ["name:pump-07-controller"]
linkedSeries:
  - {fragment: c8y_Flow, series: F, label: Flow, device: "name:pump-07-controller", sourceType: c8y_FlowMeasurement}
```

The kinds are `PropertyDefinition`, `AssetDefinition`, `MeasurementDefinition`, `EventDefinition`, `AlarmDefinition` and `Asset`. Asset fields go under `asset:`; an unknown field at the top level of an `Asset` is an error, so a typo is not silently dropped.

```go
twin, err := dtm.LoadModel("./twin-model") // files or folders; *.yaml, *.yml, *.json
if err != nil {
    return err
}

report := client.DigitalTwin.Apply(ctx, twin, dtm.ApplyOptions{
    ManagedBy:       "plant-model", // tag added to every definition Apply writes
    Prune:           true,          // delete definitions tagged plant-model that are not in the model
    ContinueOnError: true,          // default: stop at the first failure
})
for _, item := range report.Data.Items {
    fmt.Printf("%-7s %-19s %-30s %v %v\n", item.Action, item.Kind, item.Key, item.Changed, item.Err)
}
// create  PropertyDefinition  maxFlow[asset]                 [] <nil>
// none    AssetDefinition     Motor                          [] <nil>
// update  AssetDefinition     Pump                           [composition] <nil>
// assign  SubAssetAssignment  ext:site-berlin > ext:pump-07  [] <nil>
// delete  PropertyDefinition  rpm[asset]                     [] <nil>
```

Every item is idempotent: objects that already match get `ActionNone` and are not written, so applying the same model twice writes nothing the second time. `result.Err` joins the errors of the failed items, and `result.Data` always holds the report.

`Plan` does all lookups and comparisons with the same helpers as `Apply`, sends no writes, and reports for each item the action `Apply` would take, the fields that differ, and the request it would send (`Method`, `Path`). Executing the plan runs `Apply` with the same model and options; it reads the tenant again, so changes made in between are taken into account:

```go
plan := client.DigitalTwin.Plan(ctx, twin, opt)
for _, it := range plan.Data.Items {
    if it.Action != dtm.ActionNone {
        fmt.Println(it.Action, it.Kind, it.Key, it.Method, it.Path)
    }
}
if confirm(fmt.Sprintf("Apply %d changes?", plan.Data.Changes())) {
    report := plan.Execute(ctx)
    _ = report
}
```

Prune covers definitions only (open question 3): deleting assets can cascade to devices. Asset definitions are deleted before the definitions their composition uses, and property definitions last, so the "still referenced" 409 is avoided.

### 5.7 Scenario: delete with confirmation, and cascade

```go
ctx := api.WithDeferredExecution(context.Background(), true)
prepared := client.DigitalTwin.Assets.Delete(ctx, "ext:site-berlin", dtm.DeleteOptions{
    Cascade: dtm.CascadeSubAssets, // or CascadeSubAssets | CascadeDevices
})
// Meta["resolvedId"], Meta["cascade"] let a CLI print what will be removed
fmt.Println(prepared.Request.URL) // .../assets/12345?deleteSubAssets=true
if confirm("Delete Berlin site and its sub-assets?") {
    res := prepared.Execute(context.Background())
    // StatusNoContent, or StatusSkipped if it was already gone
    _ = res
}
```

`Cascade` is an explicit option, with no default, so the destructive `deleteDevices=true` is never sent by accident. The API leaves assets and devices that also belong to another parent in place; the SDK does not change that.

### 5.8 Scenario: test fixtures with `defer`

```go
func TestPumpDashboard(t *testing.T) {
    defs := client.DigitalTwin.Definitions
    require.NoError(t, defs.Properties.Upsert(ctx, maxFlowDef).Err)
    defer defs.Properties.Delete(context.Background(), "maxFlow", dtm.DeletePropertyOptions{Contexts: []string{dtm.ContextAsset}})

    pump := client.DigitalTwin.Assets.GetOrCreate(ctx, dtm.AssetCreateOptions{
        ExternalID: "test-pump-" + testutil.RandomSuffix(),
        Body:       map[string]any{"name": "Test pump", "type": "Pump", "maxFlow": 10},
    })
    require.NoError(t, pump.Err)
    defer client.DigitalTwin.Assets.Delete(context.Background(), pump.Data.ID(), dtm.DeleteOptions{
        Cascade: dtm.CascadeSubAssets,
    })

    // ...
}
```

Deletes run in reverse order (asset before definition), which is also the order the API needs. A property definition still used by an asset definition returns 409 ("is still referenced in Asset Definition '…'"); the SDK returns it as `dtm.ErrDefinitionInUse` with the referencing identifiers, and `Apply` with `Prune` deletes in reverse dependency order so it doesn't hit this.

### 5.9 Scenario: dry run in unit tests

```go
ctx := api.WithDryRun(context.Background(), true)
res := client.DigitalTwin.Assets.Upsert(ctx, dtm.AssetUpsertOptions{ExternalID: "p1", Body: body})
assert.Equal(t, "POST", res.Request.Method)
assert.Equal(t, "/service/dtm/assets", res.Request.URL.Path)
assert.Equal(t, "true", res.Request.Header.Get("X-Upsert-Mode"))
```

## 6. Models

New `jsonmodels` facades, each with typed accessors and full `jsondoc` access for custom fragments:

| Model | Main accessors |
|---|---|
| `DigitalTwinAsset` | `ID`, `Name`, `Type`, `ExternalID`, `Notes`, `SubAssets()`, `Parents()`, `LinkedSeries()`, `LatestMeasurements()`, `ChildrenCount` |
| `DigitalTwinLinkedSeries` | `Fragment`, `Series`, `Label`, `Source()` (`ID`, `Name`, `Type`, `Fragment`, `Series`) |
| `DigitalTwinLinkedAsset` | `Fragment`, `Series`, `Type`, `Name`, `Asset()` |
| `DigitalTwinDefinition` | `Identifier`, `Title`, `JSONSchema()`, `Tags`, `Contexts`, `Origin`, `IsShared`, `AvailableActions`, `Composition()` |
| `DataPrepRule` | `Name`, `Description`, `Tags`, `Disabled`, `Transport`, `TopicPattern`, `ClientIDPattern`, `JSCode`, `Tests()`, `DeploymentStatus()` (`InProgress`, `Disabled`, `Version`, `Error`, `Time`, `User`) |
| `DataPrepRuleCollection` | `Items()`, `Names()`, `Get(name)` |
| `DataPrepTestResults` | `All()`, `Test(name)`, `Passed()`, `Failures()`, `Compare()` |

Request types (`dtm.PropertyDefinition`, `dtm.AssetDefinition`, `dtm.LinkedSeries`, `dataprep.RuleConfig`, `dataprep.TestCase`) are plain structs with JSON tags and an `Extra map[string]any` for custom fragments. Every method that takes one of these also accepts `any` (a map or the user's own struct) through a `...Raw` variant, as `alarms.UpsertRaw` does today.

Collections use `httpcore.ExecuteCollection` with the DTM array names (`assets`, `definitions`, `linkedSeries`, `devices`) and `statistics` for paging, so `ListAll` and `IterAs[T]()` work without new pagination code.

## 7. Implementation plan

1. `dtm` read-only: models, `Assets.Get/List/ListAll/Count`, asset resolver, definition `Get/List/Count/Tags`. Offline tests with recorded responses.
2. `dtm` writes: CRUD, `Assign/Unassign`, linked series, idempotent `Delete`.
3. `dtm` composites: `GetOrCreate`, `Upsert` (assets and definitions), `Link`, schema merge on update, read-only and in-use checks, and the 422 "is not a child" mapping for `Unassign`.
4. `dataprep`: `RulePackage` with local validation, `List`, `Get`, `GetDraft`, `Exists`, `Deploy` (with wait), `Delete`, `Tests.Run`, `TestAndDeploy`, `Apply`, `Export`, `Diagnostics.Download`.
5. `dtm.Apply` / `Plan` (after open question 3 is settled).
6. Integration tests in `test/` against a tenant with DTM and Data Preparation subscribed; skip when `GET /service/dtm/assets/count` or `/service/dataprep/...` returns 404.
7. Update `api-coverage` spec and `V2.md` status tables.

Steps 1 to 3 and step 4 do not depend on each other and can be built in parallel.

## 8. Open questions

1. **Undocumented rule endpoints.** `GET /v1/rules`, `GET /v1/rules/{name}/deployed` and `GET /v1/rules/{name}/draft` work but aren't in the OpenAPI spec. Can the Data Preparation team confirm they are stable and add them to the spec? Related:
   - Will `GET /v1/rules` stay unpaged? A tenant with hundreds of rules returns one large object.
   - What is `deploymentVersion` computed from? If it is a hash of the normalized package, `Apply` could compare one value instead of the code, config and tests.
   - Is `PUT .../deployed` synchronous, or can `deploymentInProgress` still be `true` after the 200? This decides whether `WaitTimeout` defaults to 0 or to a few seconds.
   - Is there a write endpoint for drafts (used by the application editor), and should the SDK expose it?
2. **Data Preparation deploy behavior (not yet verified).** go-c8y-cli's `api --file` sends multipart, so the deploy test could not run through it. Still to check with a raw `PUT`: whether `deploymentInProgress` can be `true` right after the 200, and whether redeploying an identical package changes `deploymentVersion`.
3. **Prune scope.** Resolved: `Prune` covers definitions only, because pruning assets can cascade to devices. Assets can be added later as a separate opt-in if needed.
4. **Naming.** `client.DigitalTwin` and `client.DataPreparation`, or `client.DTM` and `client.DataPrep` to match the URL paths?
5. **Expected-output comparison.** The run-tests endpoint returns outputs only. Should `Compare` treat extra output fields as a pass (subset match, same as `DesiredStateMatches`) or a fail? The proposal uses subset match.
6. **DTM definition updates (for the DTM team).**
   - The asset definition docs say the schema can't change after creation, but PUT changed title, description and composition. Which is intended?
   - PUT on a measurement definition dropped the server-generated `source` property and `required`. Is that a bug, or should clients always send the full schema?
   - Unassigning a child that isn't assigned returns 422. Could it return 204 or 404, so clients don't need to match the message text?

## 9. Verified behavior

Checked on a Cumulocity tenant on 2026-11-24 with go-c8y-cli. All test objects used a `gocy-proposal` prefix and were deleted afterwards.

### 9.1 Data Preparation (read only)

| Request | Result |
|---|---|
| `GET /v1/rules` | 200, object keyed by rule name with `config` and `deploymentStatus`, not paged |
| `GET /v1/rules/{name}/deployed` | 200, adds `jsCode` and `tests` |
| `GET /v1/rules/{name}/draft` | 200, same shape as `deployed` |
| `GET /v1/rules/{name}` | 405 |
| `GET /v1/rules/{missing}/deployed` | 404, `Rule '<name>' not found` |

### 9.2 Digital Twin Manager

| Request | Result | SDK consequence |
|---|---|---|
| `POST /assets`, external ID exists, no header | 409, `externalId '…' is already used by ManagedObject '<id>'` | `GetOrCreate` follows up with a GET by external ID |
| `POST /assets`, `X-Upsert-Mode: true`, identical body | 200, `lastUpdated` changes | client-side compare needed to return `Skipped` |
| `POST /assets/{id}/subAssets/{subId}` twice | 201 both times, no duplicate | `Assign` is safe to repeat without a pre-check |
| `POST /assets/{id}/devices/{deviceId}` twice | 201 both times, one device listed | same |
| `DELETE /assets/{id}/subAssets/{subId}` when not assigned | 422, `is not a child of object` | `Unassign` maps this 422 (and only this message) to `Skipped` |
| `DELETE /assets/{id}/devices/{deviceId}` when not assigned | 422, same message | same |
| `POST /definitions/properties`, same identifier and contexts | 409, lists both the title and the identifier conflict | `GetOrCreate` follows up with a GET; `ErrDuplicateTitle` for a title-only conflict |
| `POST /definitions/assets`, `/measurements` | 201; server adds `$schema`, `definitions`, `properties`, `required`, `additionalProperties` | subset comparison in `Upsert` |
| `PUT /definitions/assets` with new title, description, composition | 200, all applied, schema rebuilt from composition | no SDK immutability check (open question 6) |
| `PUT /definitions/measurements` with partial `jsonSchema` | 200, schema replaced, server-generated `source` dropped | `Update` merges with the stored schema by default |
| `DELETE /definitions/properties/{id}` while referenced | 409, `still referenced in Asset Definition '…'` | `ErrDefinitionInUse`, reverse-order prune |
| `DELETE /definitions/assets/{id}` twice | 204, then 404 | `Delete` uses `IgnoreNotFound` |
| `DELETE /assets/{id}` twice | 204, then 404 | same |
