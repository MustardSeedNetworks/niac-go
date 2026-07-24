# ADR 0001: Generate the YAML schema from Go structs

| Status | Date | Decider |
| --- | --- | --- |
| Partially implemented | 2026-05-25 | @krisarmstrong |

## Context

NIAC configuration structs are the source for the YAML authoring contract.
Maintaining a separate JSON Schema by hand caused drift and made editor
validation unreliable.

The repository already has `cmd/niac-schema`, which reflects the authoring
structs into `docs/schemas/niac.schema.json`. Example YAML files reference that
schema for editor validation.

The original proposal also treated that authoring schema as a source for REST
TypeScript types. That part is not implemented and is not valid: the YAML
authoring contract and REST response contracts are different surfaces.

## Decision

Generate and commit the YAML authoring schema from the Go authoring structs.
CI regenerates the schema and rejects drift.

```text
Go authoring structs
        |
        v
cmd/niac-schema
        |
        v
docs/schemas/niac.schema.json
        |
        v
YAML editor validation
```

The implementation contract is:

- `cmd/niac-schema` contains the generator;
- `make schema` runs it;
- `docs/schemas/niac.schema.json` is committed;
- example YAML files reference the schema; and
- CI runs the generator and rejects a dirty diff.

REST type generation is a separate architectural decision. It requires a real
REST schema source, generated client types, and a migration of each consumer.
The former empty `ui/src/types/generated` placeholder is removed.

## Consequences

Positive:

- authoring fields have one source of truth;
- CI catches schema drift;
- editors validate examples against the shipping contract; and
- the repository does not claim nonexistent REST code generation.

Trade-offs:

- schema changes appear in review diffs;
- contributors must run `make schema` after authoring-struct changes; and
- REST types remain maintained independently until a wire-schema decision is
  accepted and implemented.

## Alternatives

Hand-maintained JSON Schema was rejected because it duplicates the Go
authoring contract. Generating REST types from the YAML schema was rejected
because it would encode a false API contract.

## Related work

- Issue #665 proposed schema generation.
- PR #682 introduced the NIAC generator and drift gate.
