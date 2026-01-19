# Breaking Change Policy

## Semantic Versioning

NIAC-Go follows [Semantic Versioning 2.0.0](https://semver.org/):

- **MAJOR** (X.0.0): Breaking changes
- **MINOR** (x.X.0): New features, backward compatible
- **PATCH** (x.x.X): Bug fixes, backward compatible

## Breaking Changes

### What Constitutes a Breaking Change?

- Removing API endpoints or changing their behavior
- Changing configuration file format incomp atibly
- Removing CLI flags or commands
- Changing default behavior that affects existing workflows
- Modifying packet formats or protocol behavior

### Deprecation Process

1. **Announce**: Document in CHANGELOG with **[DEPRECATED]** tag
2. **Grace Period**: Minimum 2 minor versions (e.g., deprecated in 2.5, removed in 2.7)
3. **Warnings**: Log warnings when deprecated features are used
4. **Remove**: Only in MAJOR version bump

### API Versioning

API versions are embedded in URLs (`/api/v1/`). When breaking changes are needed:

1. Create new version (`/api/v2/`)
2. Support old version for 6+ months
3. Document migration path
4. Announce end-of-life date

## Backward Compatibility Guarantees

### Configuration Files

YAML configuration format is stable within MAJOR versions. Minor versions may add new fields but won't remove or change existing ones.

### API Stability

Within a MAJOR version:
- Endpoint URLs won't change
- Response formats remain compatible
- New optional fields may be added
- Required fields won't be added to requests

### CLI Stability

Command-line interface is stable within MAJOR versions:
- Flags won't be removed
- Flag behavior won't change
- New flags may be added
- Deprecated flags will warn before removal

## Migration Guides

Breaking changes will always include detailed migration guides in:
- CHANGELOG.md
- GitHub release notes
- Migration guide documents (docs/MIGRATION-vX.md)

## Exception: Security Fixes

Security vulnerabilities may require breaking changes in PATCH versions if:
- Critical security impact (CVSS >= 9.0)
- No backward-compatible fix possible
- Affects default secure operation

These will be clearly documented and announced.
