# CI/CD Pipeline

The CI pipeline runs on every pull request and merge-queue group. A push to main
runs `ci-smoke.yml` instead of this suite. **All checks must pass.**

Every job except `changes` is gated on the `changes` job's path filters, so a
docs-only PR does not pay for a Go build. `ci-complete` is the single required
status check — it depends on every other job, so adding a job to `ci.yml`
without adding it to `ci-complete`'s `needs:` list makes that job advisory.

`changes`, `backend`, `race`, `frontend`, `storybook`, and `e2e` call the
reusable workflows in `MustardSeedNetworks/.github`, pinned to commit
`e092081a97245d29978e229be7425179cc1dcde1`. Caller jobs grant the permissions
those workflows declare (`id-token: write` for path filters, `actions: read`
to download an artifact, `actions: write` for the Playwright reports). That
commit pins `apt-install` and `flake-budget` to the called repository, so this
repo does not vendor them. Jobs that already call `apt-install` keep the remote
`v1.13.0` ref.

`ci-complete` stays in this file. The shared aggregator would publish the
check as `ci-complete / CI Complete`, and branch protection plus `release.yml`
require the plain name `CI Complete`. The local job still runs on
`if: ${{ !cancelled() }}` and fails unless every needed job succeeded or was
skipped. macOS and Windows builds are inside the `backend` caller, so they are
not separate `needs` entries.

These jobs stay in this file because the shared workflows have no input for
them: `build-ui` (the shared frontend workflow does not produce the production
bundle), `lint-windows` (`only-new-issues`, pull requests only), the raw-socket
boundary, security scanning, quality, release notes, workflow lint, i18n, docs,
the release-shaped build, the Docker image check, and `ci-complete`.
`phone-width`, `semgrep`, `dependency-review`, `ci-conformance`, and
`codeql-alert-gate` were already callers of other shared workflows and are
unchanged.

## GitHub Actions Workflows

### ci.yml - Main CI Pipeline

| Job | Description | Checks |
| ------------------- | --------------------------- | ----------------------------------------------------------------------------- |
| `changes` | Path filtering | Shared `ci-changes.yml`. Decides which downstream jobs run |
| `backend` | Go checks | Shared `ci-backend.yml`: Linux gates, plus macOS and Windows off pull requests |
| `lint-windows` | Windows lint | Local. `only-new-issues` on pull requests; the shared job cannot express that |
| `race` | Go race detector | Shared `ci-race.yml`. `go test -race` |
| `capture-rawsocket` | Raw-socket boundary (Linux) | Local. `rawsocket`-tagged capture tests, which need CAP_NET_RAW |
| `build-ui` | Build UI (shared artifact) | Local. Builds the frontend once; `build` and `e2e` consume the artifact |
| `frontend` | React/TS checks | Shared `ci-frontend.yml`: tsc, Biome, Vitest coverage, token discipline, Storybook build |
| `storybook` | Storybook | Shared `ci-storybook.yml`: interactions, accessibility, injected-defect contract |
| `security` | Security scans | govulncheck (hard gate), gosec, npm audit, gitleaks, Trivy |
| `semgrep` | SAST | Semgrep rules |
| `ci-conformance` | Fleet CI conformance | Reusable workflow from `MustardSeedNetworks/.github` |
| `quality` | Code quality gates | banned vocabulary, file size ratchet, output escaping, sensitive files |
| `workflow-lint` | Workflow static analysis | actionlint; zizmor (blocks on High) |
| `i18n` | Internationalization | Catalog completeness, no translated standard terms |
| `docs` | Documentation | Markdown lint (blocking, scoped to changed files) |
| `build` | Build verification | Multi-arch binaries with full ldflags, UIBuildHash verified |
| `e2e` | Browser tests | Shared `ci-e2e.yml`: sharded chromium and webkit, plus `@smoke` and the auth, history, query, permissions, and unsaved suites, in the Playwright 1.64 image |
| `codeql-alert-gate` | CodeQL alert gate | Fails on open High/Critical CodeQL alerts; reusable from `.github` |
| `ci-complete` | Aggregate gate | Local job named `CI Complete`. The required status check |

### Other Workflows

| Workflow | Purpose |
| ---------------------- | ----------------------------------------------------- |
| `cache-npcap-sdk.yml` | Cache the Npcap SDK for Windows builds |
| `codeql.yml` | CodeQL security analysis (Go, JS/TS) |
| `dead-code.yml` | Weekly dead code detection |
| `docs-link-check.yml` | Weekly external link check (split out of `ci.yml`) |
| `label-sync.yml` | Sync label definitions |
| `license-check.yml` | Verify dependency licenses |
| `main-retry.yml` | Retry a failed main run once (see the file header) |
| `merge-queue-ejection.yml` | Stop a PR the merge queue keeps ejecting |
| `pr-hygiene.yml` | Title lint, PR body template, auto-labels |
| `release-please.yml` | Automated version management and release PRs |
| `release.yml` | goreleaser release builds, signing, SLSA provenance |
| `scorecard.yml` | OpenSSF Scorecard |
| `todo-tracker.yml` | Weekly TODO tracking |

## Build contract

`build` verifies the Universal Build Contract, not just that compilation
succeeds: every binary embeds `Version`, `Commit`, `BuildTime` and
`UIBuildHash`. The "Verify UIBuildHash embedded" step (added in #1251)
recomputes the expected 32-character md5 hash from `internal/api/ui/`,
confirms `internal/api/ui/` is not empty (the frontend-dist artifact from the
`build-ui` job actually landed), and greps the built binary's strings for
that hash — catching a build that ships without the `-X ...UIBuildHash=...`
ldflag actually taking effect. A raw `go build` in CI would otherwise produce
a binary whose `/__version` reports `"unknown"`, the silent failure this
check exists to catch.

## Workflow security

`workflow-lint` runs two scanners over `.github/workflows/` itself:

- **actionlint** — syntax, expression and shell errors inside `run:` blocks.
  It catches things a plain YAML parse does not, including duplicate `with:`
  keys, which `yaml.safe_load` accepts silently by keeping the last one.
  `SC2129` is ignored as a pure style preference; every correctness rule stays on.
- **zizmor** (pinned 1.29.0) — Actions security scanner. **Blocks on High
  findings.** The repo sits at zero High. One finding survived review and
  carries a `# zizmor: ignore[...]` comment with the reasoning inline (in
  `release-please.yml`); anything else that reaches High fails the build.
  Medium, Low and Informational are reported but not yet enforced (currently
  1 medium, 1 low, 6 informational).

Permissions follow least privilege: workflows declare `permissions: {}` (or
`contents: read`) at the top level and grant scopes per job. A new job that
needs a write scope declares it on the job, never workflow-wide. `release.yml`
deliberately runs without npm caching, because its output is published and
attested and a restored cache entry could land inside a signed artifact; it
opts out by passing `cache: ""` to the `setup-node` composite action.

## The Node.js pin lives in one file

`.nvmrc` is the single source of truth for the Node version. Every workflow that
needs Node uses `./.github/actions/setup-node`, and that composite reads
`.nvmrc` via `node-version-file` — it has **no `node-version` input**, so no
caller can override it and no second copy of the version can exist.

That input used to default to a literal, and it drifted: Renovate bumps the
manifests it can see and cannot see a default buried inside a composite, so CI
ran 26.7.0 against manifests demanding 26.8.1 and logged EBADENGINE on every
job for weeks. "Must stay in step" was the previous rule here, and a rule that
depends on someone remembering is not a mechanism.

The remaining pair that can disagree is `.nvmrc` and the `engines` field in
`package.json`. Making that a hard failure (`engine-strict=true`) is the obvious
next step and is deliberately **not** taken yet: Homebrew's newest `node` is
26.7.0, so 26.8.1 is not installable through the fleet's normal channel, and
turning the mismatch fatal would block local development in all four repos. See
the linked issue.

The npm version is still declared in the composite; `packageManager` in
`package.json` is what `engines` checks it against.

## CI Must Pass Before Merge

`main` is protected. Push a feature branch, open a PR, and let CI gate it:

```bash
gh pr create --fill
gh pr merge --auto
```

`main` uses a **merge queue**, which rejects `--squash` and `--delete-branch`
on `gh pr merge`: the queue owns the merge method. A queued PR reports
`BLOCKED` with an entry under `mergeQueue`, not `CLEAN` — check
`mergeQueueEntry.state` rather than `autoMergeRequest`, which stays null.

Fix issues locally first:

```bash
make all       # Full local verification
make verify    # lint, test, security, build, schema
make test-e2e  # Build and run frontend E2E tests against the HTTPS daemon
```

## Running CI Checks Locally

### Backend

```bash
make lint-backend      # golangci-lint v2.13.2 (the version ci.yml pins)
make test-backend      # Go tests
make test-coverage     # Coverage report
make security-backend  # govulncheck
```

### Frontend

```bash
make lint-frontend     # Biome
make test-frontend     # Vitest
make build-frontend    # Vite build into internal/api/ui/
```

### Security

```bash
make security          # All security scans
make security-secrets  # gitleaks
make security-trivy    # Trivy
```

### Workflows

```bash
actionlint -ignore 'SC2129'
zizmor --min-severity high .github/workflows/
```
