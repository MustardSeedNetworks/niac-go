# Compatibility Policy

NIAC is pre-1.0. Until the `v1.0.0` tag, releases may replace configuration,
API, CLI, storage, and protocol behavior without a compatibility shim or
deprecation period. Active callers and documentation are migrated in the same
change.

Every breaking change must still be explicit in the changelog and release
notes. Silent behavior changes are not acceptable.

## Starting with v1.0.0

NIAC will follow Semantic Versioning 2.0.0:

- Major releases may contain breaking changes.
- Minor releases add backward-compatible capability.
- Patch releases contain backward-compatible fixes.

Any post-1.0 exception required to correct a security vulnerability will be
documented with its impact and migration steps.
