# NIAC YAML schemas

This directory holds the published JSON Schemas for the formats NIAC reads
and writes.

| File                  | Source-of-truth Go type        | Regenerate with |
|-----------------------|--------------------------------|-----------------|
| `niac.schema.json`    | `internal/converter.Config`    | `make schema`   |

## Editor integration

### VS Code (yaml-language-server)

Install the **YAML** extension (Red Hat). Then add a modeline to the top of
any NIAC config file:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/krisarmstrong/niac-go/main/docs/schemas/niac.schema.json
include_path: "."
devices:
  - name: my-router
    ...
```

…or set it globally in `settings.json`:

```jsonc
{
  "yaml.schemas": {
    "https://raw.githubusercontent.com/krisarmstrong/niac-go/main/docs/schemas/niac.schema.json":
      ["niac-*.yaml", "examples/**/*.yaml"]
  }
}
```

### JetBrains IDEs

Settings → Languages & Frameworks → Schemas and DTDs → JSON Schema Mappings
→ Add. Use the same raw GitHub URL.

### CI / pre-commit

Run `ajv validate` (or any draft-2020-12 validator) against the generated
schema for each YAML config before merging:

```sh
ajv validate -s docs/schemas/niac.schema.json -d "examples/**/*.yaml"
```

## Regenerating after struct changes

The schema is auto-derived from the Go type so it can't drift. If you add or
remove fields on `internal/converter.Config` (or any of its referenced
structs), run `make schema` and commit the resulting diff alongside the Go
change. CI may eventually enforce this with a "schema is up to date" check.
