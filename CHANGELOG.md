# Changelog

All notable changes to NIAC will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.88.0](https://github.com/krisarmstrong/niac-go/compare/v0.87.0...v0.88.0) (2026-05-28)


### Features

* **ui:** add slim HeaderBar (Phase 2) ([#744](https://github.com/krisarmstrong/niac-go/issues/744)) ([3e36a05](https://github.com/krisarmstrong/niac-go/commit/3e36a0570d1dec229cf6ccf1c839352655b164dc))
* **ui:** sync canonical shell from stem + decouple drawers (Phase 1) ([#738](https://github.com/krisarmstrong/niac-go/issues/738)) ([5cf30fc](https://github.com/krisarmstrong/niac-go/commit/5cf30fc3bde7364a3b21b716d755fd32b4e4f12a))


### Bug Fixes

* **ci:** unblock NIAC main — duplicate heading + Lighthouse cert ([#745](https://github.com/krisarmstrong/niac-go/issues/745)) ([3d4a5ad](https://github.com/krisarmstrong/niac-go/commit/3d4a5ad8aea7c72e71d6d52c25f53455124e8d30))
* **ui:** restore missing spacing utility classes + dual-aside e2e selector ([#747](https://github.com/krisarmstrong/niac-go/issues/747)) ([f56ad75](https://github.com/krisarmstrong/niac-go/commit/f56ad75dbe724bdaf4f226ef218acd9a17a29483))

## [0.87.0](https://github.com/krisarmstrong/niac-go/compare/v0.86.0...v0.87.0) (2026-05-27)


### Features

* **license:** model bgp/ospf/snmpv3 on device, gate bgp+ospf ([#735](https://github.com/krisarmstrong/niac-go/issues/735)) ([5738d90](https://github.com/krisarmstrong/niac-go/commit/5738d905ad21efe9207e7c32c4381bb6a96c1e4b)), closes [#129](https://github.com/krisarmstrong/niac-go/issues/129)

## [0.86.0](https://github.com/krisarmstrong/niac-go/compare/v0.85.0...v0.86.0) (2026-05-27)


### Features

* **api:** add strict JSON decode helpers + sweep handlers ([#718](https://github.com/krisarmstrong/niac-go/issues/718)) ([#721](https://github.com/krisarmstrong/niac-go/issues/721)) ([b22344f](https://github.com/krisarmstrong/niac-go/commit/b22344f7a2bb845fe4bff3fa4f55e52de881e4d0))
* **forms:** adopt react-hook-form + zod resolver ([#725](https://github.com/krisarmstrong/niac-go/issues/725)) ([#729](https://github.com/krisarmstrong/niac-go/issues/729)) ([82bcdf4](https://github.com/krisarmstrong/niac-go/commit/82bcdf46e0664feb9684fdd45071a0dba826cec2))
* **forms:** migrate UploadTemplateModal + ErrorInjectionPanel ([#730](https://github.com/krisarmstrong/niac-go/issues/730)) ([#731](https://github.com/krisarmstrong/niac-go/issues/731)) ([c50d3a7](https://github.com/krisarmstrong/niac-go/commit/c50d3a729874b64c5f2f47ca3bd1966dd510f801))
* **i18n:** add check-keys.py — t() call ↔ EN locale cross-reference ([#723](https://github.com/krisarmstrong/niac-go/issues/723)) ([e71ca2b](https://github.com/krisarmstrong/niac-go/commit/e71ca2bdde321e852c48a3abb88e994beee91a10))
* **i18n:** add per-repo dynamic-prefixes allowlist for check-keys.py ([#732](https://github.com/krisarmstrong/niac-go/issues/732)) ([4ecab2c](https://github.com/krisarmstrong/niac-go/commit/4ecab2c2336dc25373c509b40c63d70a40184fbd))
* **i18n:** migrate 8 long-tail components to t() (16 strings) ([#720](https://github.com/krisarmstrong/niac-go/issues/720)) ([2caed9b](https://github.com/krisarmstrong/niac-go/commit/2caed9b2b4ce7d9dea9c2be115c5eeac969c950d))
* **i18n:** Phase 3 NIAC — bootstrap runtime + migrate ~110 hardcoded JSX strings ([#717](https://github.com/krisarmstrong/niac-go/issues/717)) ([ffeb7b1](https://github.com/krisarmstrong/niac-go/commit/ffeb7b1ff06cd913153b2af7f616d480ff372305))
* **i18n:** pluralization + Intl APIs + locale-aware formatters ([#719](https://github.com/krisarmstrong/niac-go/issues/719)) ([b8bf03d](https://github.com/krisarmstrong/niac-go/commit/b8bf03d9c7f11f6ff52b3288365f1d02a777eb1b))


### Bug Fixes

* **ci:** honor PLAYWRIGHT_IGNORE_HTTPS_ERRORS + Lighthouse cert flag ([#722](https://github.com/krisarmstrong/niac-go/issues/722)) ([eae281f](https://github.com/krisarmstrong/niac-go/commit/eae281f896529763055cffd11c2c6d0bde67842a))

## [0.85.0](https://github.com/krisarmstrong/niac-go/compare/v0.84.0...v0.85.0) (2026-05-26)


### Features

* **api:** add GET /api/v1/license read endpoint ([#710](https://github.com/krisarmstrong/niac-go/issues/710)) ([50e16c7](https://github.com/krisarmstrong/niac-go/commit/50e16c7ad8022f3594cd9cd93f63bd60237b41fc))

## [0.84.0](https://github.com/krisarmstrong/niac-go/compare/v0.83.1...v0.84.0) (2026-05-26)


### Features

* **i18n:** add errors.license.* keys for tier-gating UI ([#709](https://github.com/krisarmstrong/niac-go/issues/709)) ([76b2a93](https://github.com/krisarmstrong/niac-go/commit/76b2a934265c8cc7f5da120cab5ee4700ff4da4f))
* **license:** add per-route feature gating framework ([#704](https://github.com/krisarmstrong/niac-go/issues/704)) ([710b566](https://github.com/krisarmstrong/niac-go/commit/710b5668dd4a65dfb57d8df3ecf269aded4ee406))
* **license:** enforce Free-tier 10-device soft cap on device create ([#706](https://github.com/krisarmstrong/niac-go/issues/706)) ([44e254f](https://github.com/krisarmstrong/niac-go/commit/44e254fbfb34817ea8628a3a51072fb5ee10e714))
* **license:** gate pcap, templates, traffic_shaping, multi_ip ([#708](https://github.com/krisarmstrong/niac-go/issues/708)) ([0c87ce2](https://github.com/krisarmstrong/niac-go/commit/0c87ce2b2b72ba3ad04529cdfce6e831272bcaa6))
* **license:** gate STP/FTP/NetBIOS protocols on device create+update ([#707](https://github.com/krisarmstrong/niac-go/issues/707)) ([fe16dab](https://github.com/krisarmstrong/niac-go/commit/fe16dab121ce82fcc91c6a4eb699ae00bdb7ebf2))

## [0.83.1](https://github.com/krisarmstrong/niac-go/compare/v0.83.0...v0.83.1) (2026-05-26)


### Bug Fixes

* **ci:** switch e2e + lighthouse to HTTPS for TLS-only daemon ([#701](https://github.com/krisarmstrong/niac-go/issues/701)) ([b8aa9a5](https://github.com/krisarmstrong/niac-go/commit/b8aa9a536d45c24f9eee89f82a0286c37a87fe6b))
* **e2e:** switch fullstack config to HTTPS for TLS-only daemon ([#698](https://github.com/krisarmstrong/niac-go/issues/698)) ([7338525](https://github.com/krisarmstrong/niac-go/commit/7338525955e53a4195e2f5ac4bb679788ef82eb1))
* **license:** add RWMutex to Manager for safe concurrent access ([#703](https://github.com/krisarmstrong/niac-go/issues/703)) ([ed0d427](https://github.com/krisarmstrong/niac-go/commit/ed0d427440f9b9dd1ea3e7725b73b781450198dc))
* **scripts:** clean up all shellcheck warnings + pin severity=warning ([#696](https://github.com/krisarmstrong/niac-go/issues/696)) ([71ac82d](https://github.com/krisarmstrong/niac-go/commit/71ac82d544bfcbf148fc34ed28e6cbb90e230adc))

## [0.83.0](https://github.com/krisarmstrong/niac-go/compare/v0.82.0...v0.83.0) (2026-05-25)


### Features

* **converter:** add struct-tag validation for Config ([#685](https://github.com/krisarmstrong/niac-go/issues/685)) ([269a924](https://github.com/krisarmstrong/niac-go/commit/269a924d6d353b5ab183c5c57928ad14709d492d)), closes [#669](https://github.com/krisarmstrong/niac-go/issues/669)


### Bug Fixes

* **ci:** inject UIBuildHash ldflag (Universal Build Contract) ([#682](https://github.com/krisarmstrong/niac-go/issues/682)) ([d37b4f9](https://github.com/krisarmstrong/niac-go/commit/d37b4f961106dba2d37053e604995ee5a7e5f98d))
* **docs:** correct PR template 'cd web' -&gt; 'cd ui' ([#683](https://github.com/krisarmstrong/niac-go/issues/683)) ([3e5dc18](https://github.com/krisarmstrong/niac-go/commit/3e5dc187f64edd9083d13374676123f34d56a26b))
* **scripts:** deploy-validate add HTTPS support + canonical port 8445 ([#692](https://github.com/krisarmstrong/niac-go/issues/692)) ([740a746](https://github.com/krisarmstrong/niac-go/commit/740a746021241fde2fbcc1bfbf6037cfe9f8ed51))

## [0.82.0](https://github.com/krisarmstrong/niac-go/compare/v0.81.1...v0.82.0) (2026-05-25)


### Features

* **license:** add offline license framework with trial and keygen contract ([#671](https://github.com/krisarmstrong/niac-go/issues/671)) ([18d7b29](https://github.com/krisarmstrong/niac-go/commit/18d7b29c5ee774889198cc0d7816ee7ddb5aa043))
* **security:** Require HTTPS unconditionally ([#1070](https://github.com/krisarmstrong/niac-go/issues/1070)) ([#663](https://github.com/krisarmstrong/niac-go/issues/663)) ([1e33b59](https://github.com/krisarmstrong/niac-go/commit/1e33b59c64981a5555211dc62c9da844027cd38f))


### Bug Fixes

* **e2e:** repair niac config-diff strict-mode + gui-daemon test.skip typo ([#661](https://github.com/krisarmstrong/niac-go/issues/661)) ([2434259](https://github.com/krisarmstrong/niac-go/commit/2434259a04a85eeb23b6b61f11d178c003d3dbe5))

## [0.81.1](https://github.com/krisarmstrong/niac-go/compare/v0.81.0...v0.81.1) (2026-05-22)


### Performance Improvements

* **e2e:** bump CI workers 1-&gt;2 and retries 2-&gt;1 ([#658](https://github.com/krisarmstrong/niac-go/issues/658)) ([65afe89](https://github.com/krisarmstrong/niac-go/commit/65afe899a529814f34dca11892374e8310230e79))

## [0.81.0](https://github.com/krisarmstrong/niac-go/compare/v0.80.0...v0.81.0) (2026-05-22)


### Features

* **theme:** adopt botanical-earth surface palette (Phase 4) ([8133adb](https://github.com/krisarmstrong/niac-go/commit/8133adbe2e7fd01d8a4125d761fc1945a9b1256f))
* **theme:** adopt canonical responsive type scale (Phase 3) ([4f411f2](https://github.com/krisarmstrong/niac-go/commit/4f411f212b6d9269dc5a2c375735d740eb0d5d28))
* **theme:** Apply 2026-05-22 brand audit — NIAC becomes indigo + 5 modules ([26bc0f0](https://github.com/krisarmstrong/niac-go/commit/26bc0f0daaeaf8682ecf1b3d9d7cadd463863116))
* **theme:** differentiate NIAC modules + flatten component primitives (Phase 6) ([06764c0](https://github.com/krisarmstrong/niac-go/commit/06764c0d2f6ef0763dc0bcaf29c1e6e0ce77eb40))
* **theme:** identity shift — NIAC becomes indigo (Phase 5) ([5a2f110](https://github.com/krisarmstrong/niac-go/commit/5a2f110c1c08d7fb47cc81322cbaede9131a8d11))
* **theme:** self-host fonts via [@fontsource-variable](https://github.com/fontsource-variable), drop Space Grotesk (Phase 2) ([0010faf](https://github.com/krisarmstrong/niac-go/commit/0010faf6c4478efc29ecf55bd49e643a2239678f))
* **theme:** swap status palette to canonical brand-tied anchors (Phase 1) ([e806bb7](https://github.com/krisarmstrong/niac-go/commit/e806bb70b37e685a8a11ce6b3b5ac30783448dad))


### Bug Fixes

* **vite:** stop inlining font assets as data: URLs (CSP fix) ([4efa048](https://github.com/krisarmstrong/niac-go/commit/4efa0488194bd380e1b09c917fc4aea41b0a2d5f))
* **vite:** Stop inlining font assets as data: URLs (CSP fix) ([35e87ca](https://github.com/krisarmstrong/niac-go/commit/35e87ca52c333f517706d6c41aec8f09c0642ddc))

## [0.80.0](https://github.com/krisarmstrong/niac-go/compare/v0.79.0...v0.80.0) (2026-05-22)


### Features

* **stories:** cover 5 context-heavy src/ui/ primitives (Wave 5 / niac-W5-2c) ([#645](https://github.com/krisarmstrong/niac-go/issues/645)) ([d427a05](https://github.com/krisarmstrong/niac-go/commit/d427a05f23b4fa6d007965a21386783a98b2b25b))
* **stories:** cover 8 more src/ui/ primitives (Wave 5 / niac-W5-2b) ([#641](https://github.com/krisarmstrong/niac-go/issues/641)) ([191b195](https://github.com/krisarmstrong/niac-go/commit/191b195f0dc60b97c7df018159760fee6f099036))
* **stories:** primitive storybook coverage for src/ui/ (Wave 5 / niac-W5-2) ([#639](https://github.com/krisarmstrong/niac-go/issues/639)) ([6fac3bb](https://github.com/krisarmstrong/niac-go/commit/6fac3bb238e9ed55b9660e0a3883242d9e5346f0))
* **ui:** scaffold storybook 10 (Wave 5 / niac-W5-1, closes [#636](https://github.com/krisarmstrong/niac-go/issues/636)) ([#638](https://github.com/krisarmstrong/niac-go/issues/638)) ([4919cf1](https://github.com/krisarmstrong/niac-go/commit/4919cf14dc5085f0f4947dd3c5d381f1784c04f2))


### Bug Fixes

* **tsconfig:** drop deprecated baseUrl from tsconfig.app.json (TS 6 compat) ([#648](https://github.com/krisarmstrong/niac-go/issues/648)) ([a55317c](https://github.com/krisarmstrong/niac-go/commit/a55317cc65a6fa53c5daa0520894c7b02ffbd2d5))

## [0.79.0](https://github.com/krisarmstrong/niac-go/compare/v0.78.1...v0.79.0) (2026-05-20)


### Features

* **auth:** sighup token rotation + scoped tokens ([#632](https://github.com/krisarmstrong/niac-go/issues/632)) ([48567a9](https://github.com/krisarmstrong/niac-go/commit/48567a94e0444d7ca06148e1b7a3d1a86ea20a1e))
* TLS by default + canonical port 8445 + HTTP redirector + default-secure non-loopback (Wave 1) ([#630](https://github.com/krisarmstrong/niac-go/issues/630)) ([ea81fff](https://github.com/krisarmstrong/niac-go/commit/ea81fff4b3a42d8837ee569f62507a9c9237e998))

## [0.78.1](https://github.com/krisarmstrong/niac-go/compare/v0.78.0...v0.78.1) (2026-05-19)


### Bug Fixes

* **ci:** add target_tag input to SLSA backfill ([#75](https://github.com/krisarmstrong/niac-go/issues/75) follow-up) ([#626](https://github.com/krisarmstrong/niac-go/issues/626)) ([520803a](https://github.com/krisarmstrong/niac-go/commit/520803ad9daf6411fa9d77fcac287df15630862d))

## [0.78.0](https://github.com/krisarmstrong/niac-go/compare/v0.77.0...v0.78.0) (2026-05-19)


### Features

* **ci:** Add provenance_only mode for SLSA backfill ([#75](https://github.com/krisarmstrong/niac-go/issues/75)) ([#623](https://github.com/krisarmstrong/niac-go/issues/623)) ([d771b28](https://github.com/krisarmstrong/niac-go/commit/d771b28e3082990896dc3546db2c935714fa7b61))

## [0.77.0](https://github.com/krisarmstrong/niac-go/compare/v0.76.2...v0.77.0) (2026-05-19)


### Features

* Graceful port fallback when canonical port is in use ([#69](https://github.com/krisarmstrong/niac-go/issues/69)) ([#621](https://github.com/krisarmstrong/niac-go/issues/621)) ([ed835bd](https://github.com/krisarmstrong/niac-go/commit/ed835bdeb6c5f37029d5dd993dd6bd4d2f9863f6))

## [0.76.2](https://github.com/krisarmstrong/niac-go/compare/v0.76.1...v0.76.2) (2026-05-19)


### Bug Fixes

* **ci:** point Lighthouse at the real served URLs ([#65](https://github.com/krisarmstrong/niac-go/issues/65)) ([#619](https://github.com/krisarmstrong/niac-go/issues/619)) ([d8c04b3](https://github.com/krisarmstrong/niac-go/commit/d8c04b3c74b8e8b7485f3c6f7c6ac9378654c824))

## [0.76.1](https://github.com/krisarmstrong/niac-go/compare/v0.76.0...v0.76.1) (2026-05-19)


### Bug Fixes

* **ci:** exclude advisory e2e + lighthouse from CI Complete needs ([#616](https://github.com/krisarmstrong/niac-go/issues/616)) ([c49b92b](https://github.com/krisarmstrong/niac-go/commit/c49b92b0fe98ce7347f40338571053ee99f8b1df))

## [0.76.0](https://github.com/krisarmstrong/niac-go/compare/v0.75.1...v0.76.0) (2026-05-19)


### Features

* **ui:** Topbar with theme toggle + color sync with stem ([#613](https://github.com/krisarmstrong/niac-go/issues/613)) ([542771f](https://github.com/krisarmstrong/niac-go/commit/542771f21c41ff49db63d03c3c7a76ebd17742eb))

## [0.75.1](https://github.com/krisarmstrong/niac-go/compare/v0.75.0...v0.75.1) (2026-05-18)


### Bug Fixes

* **release:** replace broken SLSA generator with attest-build-provenance ([#611](https://github.com/krisarmstrong/niac-go/issues/611)) ([4471a33](https://github.com/krisarmstrong/niac-go/commit/4471a331ee60a9bd7ae024e311b8b79f484f8291))

## [0.75.0](https://github.com/krisarmstrong/niac-go/compare/v0.74.0...v0.75.0) (2026-05-18)


### Features

* dev-run target + product favicon + SPDX header migration ([#607](https://github.com/krisarmstrong/niac-go/issues/607)) ([557f3a9](https://github.com/krisarmstrong/niac-go/commit/557f3a97647a8e6883eab8b370e73f8cd99a5c26))
* **i18n:** add Spanish (es) locale with full namespace parity ([#608](https://github.com/krisarmstrong/niac-go/issues/608)) ([d04c911](https://github.com/krisarmstrong/niac-go/commit/d04c911a308263d12c1933d90600b3c6cd037da9))
* **ui:** add sun/moon toggle to sidebar footer + cmdk install ([#609](https://github.com/krisarmstrong/niac-go/issues/609)) ([08b2517](https://github.com/krisarmstrong/niac-go/commit/08b25171e2a29b8f7fb434c94e2a5f6563a028f8))

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
