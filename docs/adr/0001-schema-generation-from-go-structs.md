# ADR 0001: Schema generation from Go structs

| Status  | Date       | Deciders         |
|---------|------------|------------------|
| Accepted | 2026-05-25 | @krisarmstrong   |

## Context

The three Mustard Seed Networks sibling repos (seed, stem, niac-go) all face the
same recurring class of bugs:

1. **TypeScript types drift from Go structs.** When a Go API DTO or YAML
   config struct changes, the corresponding `ui/src/types/*.ts` definitions
   silently fall out of sync. Tests pass, builds pass, and the bug only
   surfaces when a runtime cast (`as SomeType`) hits an unexpected shape in
   production.
2. **YAML/JSON validation is ad-hoc.** Each repo has a smattering of inline
   field checks (`if device.MAC == "" { ... }`) that catch some bad inputs
   but not all. There is no single place to express constraints like "VLAN
   must be 1–4094" or "device.ip and device.ips are mutually exclusive."
3. **Editors don't know the schema.** Engineers writing YAML configs by
   hand have no inline validation in their editor. Mistakes are caught at
   load time at best, runtime at worst.

NIAC already had a partial solution: `cmd/niac-schema` uses
[`invopop/jsonschema`](https://github.com/invopop/jsonschema) to reflect a
`converter.Config` Go struct into a JSON Schema committed at
`docs/schemas/niac.schema.json`. yaml-language-server picks it up via a
top-of-file `# yaml-language-server: $schema=...` comment, so YAML files
get inline editor validation.

This ADR codifies that pattern as the cross-repo standard and extends it
in two directions:

- **Up:** add a CI drift check so an engineer can't change a struct
  without regenerating the schema. (Done for NIAC in #682 / CI workflow
  lines 111–121.)
- **Down:** generate TypeScript types from the same schemas via
  `json-schema-to-typescript`, eliminating the hand-written
  `ui/src/types/` drift problem entirely.

## Decision

**One source of truth: the Go struct.** Everything else is generated.

```
                Go struct (source of truth)
                       │
                       ▼  invopop/jsonschema reflector
              docs/schemas/*.json (committed)
                       │
        ┌──────────────┴──────────────┐
        ▼                             ▼
  json-schema-to-typescript    yaml-language-server
        │                             │
        ▼                             ▼
  ui/src/types/generated/*.ts   editor inline validation
```

Concretely, every repo adopting this pattern ships:

| Component                                         | Purpose |
|---------------------------------------------------|---------|
| `cmd/<repo>-schema/main.go`                       | Reflects struct → JSON Schema |
| `make schema` Makefile target                     | Runs the generator |
| `docs/schemas/*.json` (committed)                 | Generated schemas |
| CI step: `make schema && git diff --exit-code`    | Drift gate |
| `ui/scripts/gen-types.ts`                         | Schema → TS via `json-schema-to-typescript` |
| `npm run gen-types` (prebuild hook)               | Runs codegen before build |
| CI step: TS codegen drift gate                    | Catches forgotten regenerations |

## Implementation reference

NIAC is the reference implementation. The generator lives at
[`cmd/niac-schema/main.go`](../../cmd/niac-schema/main.go); the
significant configuration choices are:

```go
reflector := &jsonschema.Reflector{
    ExpandedStruct:            false,
    Anonymous:                 true,
    AllowAdditionalProperties: false,
}
reflector.KeyNamer    = func(s string) string { return s }
reflector.FieldNameTag = "yaml"   // or "json" for API DTOs
```

- `AllowAdditionalProperties: false` — schemas reject unknown fields,
  matching the strict-decode posture on the backend (`DisallowUnknownFields`
  in HTTP handlers, struct-tag validation in YAML loaders).
- `FieldNameTag: "yaml"` for config schemas; use `"json"` for API DTO
  schemas so the generated schema reflects the wire format clients see.
- `Anonymous: true` keeps nested types inline rather than producing
  `$ref` indirection, which yaml-language-server resolves better.

The Makefile target:

```make
schema: ## Regenerate docs/schemas/niac.schema.json from converter.Config
	@go run ./cmd/niac-schema -o docs/schemas/niac.schema.json
```

The CI drift gate (`.github/workflows/ci.yml`, "Verify niac.schema.json is up to date" step):

```yaml
- name: Verify niac.schema.json is up to date
  run: |
    go run ./cmd/niac-schema -o /tmp/niac.schema.json
    if ! diff -u docs/schemas/niac.schema.json /tmp/niac.schema.json; then
      echo "::error::docs/schemas/niac.schema.json is stale. Run 'make schema' and commit the result."
      exit 1
    fi
```

## Consequences

**Positive:**

- One place to change a field — the Go struct. Everything else falls out.
- CI catches drift the same day it's introduced, not in production.
- Editors get inline validation for hand-edited YAML/JSON without any
  per-file configuration beyond a one-line `$schema` comment.
- Frontend `as SomeType` casts become rare — generated types match the
  backend exactly.

**Negative / trade-offs:**

- Adds a code-generation step to the build pipeline. The generator runs
  in <1s for NIAC's current Config; not a meaningful cost.
- Generated TS files appear in git diffs whenever Go structs change.
  This is intentional — the diff is the signal — but inflates PR review
  surface area slightly.
- Engineers must remember to run `make schema` before pushing. The CI
  drift gate is the safety net; local pre-commit hooks could enforce it
  but are out of scope here.
- `invopop/jsonschema` does **not** automatically pick up
  `go-playground/validator` `validate:` tags. Constraint export
  (translating `validate:"gte=1,lte=4094"` into JSON Schema's
  `minimum`/`maximum`) requires a small custom reflector. Filed as a
  follow-up; today the schemas describe shape, the validator enforces
  constraints.

## Alternatives considered

**OpenAPI codegen (oapi-codegen, swagger-codegen).** Better fit for
REST API contracts where the wire format is the source of truth.
Rejected because our pain is in the *other* direction: we want the Go
struct to be the source of truth and have the wire format / TS types /
YAML schemas fall out of it. OpenAPI codegen typically expects an
OpenAPI document as input.

**Hand-maintained JSON Schemas.** Tried briefly in older internal tools.
Two failure modes: (a) engineers forget to update them, (b) reviewers
can't tell what changed in a multi-thousand-line schema. The generator
makes the schema diff a function of the struct diff.

**`mfridman/tparse`-style runtime reflection.** Generating schemas at
program startup is appealing (no committed file to drift) but breaks
yaml-language-server's static lookup. Static committed schemas + CI
drift gate gives us the same correctness guarantee with editor support.

**`xeipuuv/gojsonschema` for validation instead of
`go-playground/validator`.** gojsonschema validates against committed
JSON Schemas at runtime. Considered but rejected: validator tags live
next to the struct field they constrain (more readable, lower
maintenance), and the failure messages with field paths are better.
JSON Schema constraints are still useful for the editor; the runtime
validator catches what the schema can't easily express (cross-field
rules, custom formats).

## Migration guide

For seed and stem to adopt this pattern, each repo needs:

1. **Add the generator.** Copy `cmd/niac-schema/main.go` to
   `cmd/<repo>-schema/main.go`. Update:
   - `Reflect(&converter.Config{})` → the repo's top-level config or DTO
     types
   - `schema.Title`, `schema.Description`, `schema.ID` (point at the
     repo's GitHub raw URL)
   - `FieldNameTag` — `"yaml"` for config schemas, `"json"` for API DTOs.
     Generate one file per logical group rather than one giant schema.

2. **Add `make schema`.** In `mk/build.mk` (or equivalent):

   ```make
   schema: ## Regenerate docs/schemas/*.json from Go structs
   	@go run ./cmd/<repo>-schema -o docs/schemas/<name>.schema.json
   ```

3. **Commit `docs/schemas/.gitkeep`.** Then run `make schema` and commit
   the generated files.

4. **Add the CI drift gate.** Copy the NIAC step above into the repo's
   `.github/workflows/ci.yml`. Add a path filter so the job only runs
   when schemas or generators change:

   ```yaml
   schema:
     - 'docs/schemas/**'
     - 'cmd/<repo>-schema/**'
   ```

5. **Wire frontend codegen.** Add `json-schema-to-typescript` to
   `ui/package.json` devDependencies (pin exact version), add
   `ui/scripts/gen-types.ts` to invoke it, and add `npm run gen-types`
   to the `prebuild` script. Add a CI drift gate for the TS side too.

6. **Migrate hand-written types.** For each schema, replace the
   corresponding hand-written TS type with the generated one. Mark
   legacy types `@deprecated` if not yet migrated.

The migration is **incremental**. Per-schema, per-type. There is no big
bang.

## Related issues and PRs

- niac-go#669 — `validate:` tags + `go-playground/validator` (implemented;
  Phase 2 of the boundary-validation epic)
- niac-go#682 — CI schema drift gate (already shipped)
- krisarmstrong/seed#1099 — boundary validation epic; will adopt this pattern
- krisarmstrong/stem#267 — boundary validation epic; will adopt this pattern
