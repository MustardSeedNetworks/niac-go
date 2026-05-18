# Changelog

All notable changes to NIAC will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.74.0](https://github.com/krisarmstrong/niac-go/compare/v0.73.0...v0.74.0) (2026-05-18)


### Features

* **ui:** harmonize color theme with seed/stem (MSN Green, dark default + light toggle) ([#604](https://github.com/krisarmstrong/niac-go/issues/604)) ([e475b0c](https://github.com/krisarmstrong/niac-go/commit/e475b0cdf49a482edf2ddb5d10718682309f5904))

## [0.73.0](https://github.com/krisarmstrong/niac-go/compare/v0.72.0...v0.73.0) (2026-05-18)


### Features

* **ui:** comprehensive tooltip parity — add ~16 tooltips for icon-only buttons + complex actions ([#602](https://github.com/krisarmstrong/niac-go/issues/602)) ([c80484d](https://github.com/krisarmstrong/niac-go/commit/c80484dd5f7d5d5cf5fa858ffc89933c99e9b518))

## [0.72.0](https://github.com/krisarmstrong/niac-go/compare/v0.71.0...v0.72.0) (2026-05-18)


### Features

* **ui:** comprehensive in-app help system with 7 tabbed sections ([#598](https://github.com/krisarmstrong/niac-go/issues/598)) ([7cb663e](https://github.com/krisarmstrong/niac-go/commit/7cb663ea22facd8197767b192e558090977899b8))

## [0.71.0](https://github.com/krisarmstrong/niac-go/compare/v0.70.0...v0.71.0) (2026-05-18)


### Bug Fixes

* **ci:** grant security-events: write to Security Scanning job ([#587](https://github.com/krisarmstrong/niac-go/issues/587)) ([9aa3af0](https://github.com/krisarmstrong/niac-go/commit/9aa3af0fc0be198aad6f648aaccd5258aaf468c0))


### Miscellaneous Chores

* cut v0.71.0 release with refactor + CI work ([#594](https://github.com/krisarmstrong/niac-go/issues/594)) ([ae4516e](https://github.com/krisarmstrong/niac-go/commit/ae4516e8c4ecfd5518e4005958c4ddd798e6c317))

## [Unreleased]

### Added
- Windows arm64 release binary (`niac-*-windows-arm64.zip`).
- Admin-side `--webhook-allowed-host` daemon flag (repeatable). When set,
  the alert webhook only dispatches to listed hostnames — exact-match
  allowlist, which is the canonical CodeQL barrier for `go/request-forgery`.
- `workflow_dispatch` `dry_run` input on `release.yml`. Dispatching with
  `dry_run=true` (default) builds and signs every artifact and uploads
  them as a workflow artifact for inspection, without publishing a
  GitHub release.
- `.gitattributes` (LF defaults, binary classification, linguist hints).
- `CODE_OF_CONDUCT.md` (Contributor Covenant 2.1).

### Changed
- **Packaging:** `release.yml` and the local `make deb` / `make rpm`
  targets both now use `nfpm` instead of `dpkg-deb` + `rpmbuild`. nfpm is
  pure Go, so cross-arch RPMs build cleanly without rpmrc/platform
  plumbing — the v0.66.34 arm64 .rpm gap is closed.
- **Dependency:** `github.com/google/gopacket v1.1.19` →
  `github.com/gopacket/gopacket v1.5.0` (maintained fork). Unblocks
  windows/arm64 (the upstream lacked `defs_windows_arm64.go`) and
  catches up to two years of security and protocol fixes.
- **CI workflows:** every multi-trigger workflow now has a concurrency
  group so rapid pushes cancel stale runs (`release.yml` keeps
  `cancel-in-progress: false` so back-to-back tags don't cancel each
  other — they're independent versions).
- **Action versions:** standardised to current latest majors —
  `actions/checkout@v6`, `actions/upload-artifact@v7`,
  `actions/download-artifact@v8`, `actions/setup-python@v6`,
  `actions/github-script@v9`, `aquasecurity/trivy-action@0.36.0`,
  `securego/gosec@v2.26.1`, `sigstore/cosign-installer@v4`,
  `softprops/action-gh-release@v3`, `codecov/codecov-action@v6`.
- **RPM filename:** uses canonical `x86_64` / `aarch64` instead of
  `amd64` / `arm64` (in-package `Architecture:` header was already
  correct via nfpm's translation).
- **Dependabot:** added the npm ecosystem (`/ui`), grouped per-ecosystem,
  conventional-commit prefixes (`chore(deps)`, `chore(ci)`,
  `chore(ui-deps)`).
- Migrated `pkg/` packages to `internal/` for better encapsulation.
- Renamed `pkg/httpapi` → `internal/api`.
- Moved `pkg/snmp` → `internal/protocols/snmp`.
- Renamed `test/` → `tests/` for consistency.
- Renamed `ui/src/context/` → `ui/src/contexts/` for consistency.

### Removed
- `.github/workflows/release-please.yml` — pointed `extra-files: VERSION`
  at a non-existent file, has been a no-op since it landed.
- `deploy/deb/{control,postinst,prerm,postrm}` and
  `deploy/rpm/niac.spec` — fully replaced by `.nfpm.yaml` and shared
  scripts under `deploy/nfpm/`.

### Security
- 15 of 16 open CodeQL alerts cleared in code (path-injection inline
  barriers at every filesystem sink, integer/allocation bound checks,
  SSRF host check inlined at the dispatch site). The remaining
  `go/request-forgery` alert was dismissed pending deployments adopting
  the new `--webhook-allowed-host` allowlist; tracked in #484.

## [0.66.35] - 2026-05-06

### Changed
- First release on the new nfpm-based pipeline. `release.yml` no longer
  invokes `dpkg-deb` or `rpmbuild`; both .deb and .rpm are produced from
  a single `.nfpm.yaml` per Linux arch.
- Replaces six earlier release attempts (v0.66.27..34) that all got
  stuck on Ubuntu's `rpm` cross-arch limitations.

### Added
- arm64 `.rpm` artifact restored (was dropped in v0.66.34 as a
  workaround).
- 32 signed artifacts: tar.gz + deb + rpm (linux), tar.gz + pkg (macOS),
  zip (windows-amd64), each with cosign bundle and CycloneDX SBOM, plus
  a signed `checksums.txt`.

## [0.66.27] – [0.66.34] - 2026-05-05/06 (release pipeline iteration)

These tags exist on the repository but did not produce GitHub releases.
Each one was a fix to the release pipeline, surfaced by the next stage
of the matrix:

- **0.66.27** — initial post-PR-#483 tag; broke on a dead `cp -r ui/dist/*`
  step in `release.yml` (vite emits straight to `internal/api/ui` so
  `ui/dist/` never exists).
- **0.66.28** — fixed the `ui/dist` copy; arm64 still failed because
  the apt-source-restriction `sed` mangled `microsoft-prod.list`.
- **0.66.29** — sed now skips pre-bracketed `deb [...]` lines; arm64
  still failed because Ubuntu noble's deb822 `ubuntu.sources` wasn't
  arch-restricted.
- **0.66.30** — replaced `ubuntu.sources` with explicit `[arch=amd64]`
  `.list` entries; arm64 still failed at `rpmbuild --target aarch64`.
- **0.66.31** — `--target aarch64-linux`; same rpmbuild error (Ubuntu's
  `rpm` package doesn't ship aarch64 platform configs).
- **0.66.32** — bypassed `--target` with explicit `_arch`/`_target_*`
  defines; same error (rpmrc itself gates valid build archs).
- **0.66.33** — synthesized the missing
  `/usr/lib/rpm/platform/aarch64-linux/macros` file inline; same error
  (rpmrc, not platform macros, is the gate).
- **0.66.34** — pragmatic workaround: skip arm64 .rpm entirely. Shipped
  4/5 platforms × 4 formats. The retro of this chain motivated the
  v0.66.35 nfpm migration.

The v0.66.27..33 tags remain as historical pointers to specific commits;
they are not formal releases (no GitHub release object, no published
artifacts).

## [0.1.0] - Initial Release

- Initial NIAC implementation.
