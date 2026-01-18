# Changelog

All notable changes to NIAC will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed
- Migrated `pkg/` packages to `internal/` for better encapsulation
- Renamed `pkg/httpapi` to `internal/api`
- Moved `pkg/snmp` to `internal/protocols/snmp`
- Renamed `test/` to `tests/` for consistency
- Renamed `ui/src/context/` to `ui/src/contexts/` for consistency

## [0.1.0] - Initial Release

- Initial NIAC implementation
