# CI/CD Pipeline

The CI pipeline runs on every push and PR. **All checks must pass.**

Every job except `changes` is gated on the `changes` job's path filters, so a
docs-only PR does not pay for a Go build. `ci-complete` is the single required
status check — it depends on every other job, so adding a job to `ci.yml`
without adding it to `ci-complete`'s `needs:` list makes that job advisory.

## GitHub Actions Workflows

### ci.yml - Main CI Pipeline

| Job             | Description              | Checks                                                                 |
| --------------- | ------------------------ | ---------------------------------------------------------------------- |
| `changes`       | Path filtering           | Decides which downstream jobs run                                      |
| `backend`       | Go checks                | lint, vet, staticcheck, fmt, tests, coverage floor                     |
| `race`          | Go race detector         | `go test -race`, split from `backend` so it fails distinctly           |
| `frontend`      | React/TS checks          | tsgo typecheck, Biome, Vite build, Vitest, Storybook build             |
| `c-lint`        | C lint (C23)             | clang-format, clang-tidy                                               |
| `security`      | Security scans           | govulncheck (hard gate), gosec, npm audit, gitleaks, Trivy             |
| `semgrep`       | SAST                     | Semgrep rules                                                          |
| `quality`       | Code quality gates       | banned vocabulary, file size ratchet, output escaping, sensitive files |
| `workflow-lint` | Workflow static analysis | actionlint; zizmor (blocks on High)                                    |
| `i18n`          | Internationalization     | Catalog completeness, no translated standard terms                     |
| `docs`          | Documentation            | Markdown lint (blocking, scoped to changed files)                      |
| `build`         | Build verification       | Multi-arch binaries with full ldflags, UIBuildHash verified            |
| `lighthouse`    | Frontend performance     | Lighthouse budgets                                                     |
| `e2e`           | Browser tests            | Playwright, chromium + webkit + firefox                                |
| `ci-complete`   | Aggregate gate           | The required status check                                              |

### Other Workflows

| Workflow               | Purpose                                               |
| ---------------------- | ----------------------------------------------------- |
| `browser-channels.yml` | Playwright browser channel coverage                   |
| `cache-npcap-sdk.yml`  | Cache the Npcap SDK for Windows builds                |
| `codeql.yml`           | CodeQL security analysis (Go, JS/TS)                  |
| `dead-code.yml`        | Weekly dead code detection                            |
| `docs-link-check.yml`  | Weekly external link check (split out of `ci.yml`)    |
| `label-sync.yml`       | Sync label definitions                                |
| `labeler.yml`          | Auto-label PRs and issues                             |
| `license-check.yml`    | Verify dependency licenses                            |
| `pr-body-lint.yml`     | Enforce the PR body template                          |
| `release-please.yml`   | Automated version management and release PRs          |
| `release.yml`          | goreleaser release builds, signing, SLSA provenance   |
| `scorecard.yml`        | OpenSSF Scorecard                                     |
| `title-lint.yml`       | Lint PR and issue titles                              |
| `todo-tracker.yml`     | Weekly TODO tracking                                  |

## Build contract

`build` verifies the Universal Build Contract, not just that compilation
succeeds: every binary embeds `Version`, `Commit`, `BuildTime` and
`UIBuildHash`. The "Verify UIBuildHash embedded" step (added in #1251)
recomputes the expected 32-character md5 hash from `internal/api/ui/`,
confirms `internal/api/ui/` is not empty (the frontend-dist artifact from the
`frontend` job actually landed), and greps the built binary's strings for
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
  Low/Informational are reported but not yet enforced.

Permissions follow least privilege: workflows declare `permissions: {}` (or
`contents: read`) at the top level and grant scopes per job. A new job that
needs a write scope declares it on the job, never workflow-wide. `release.yml`
deliberately runs without npm caching, because its output is published and
attested and a restored cache entry could land inside a signed artifact; it
opts out by passing `cache: ""` to the `setup-node` composite action.

## The Node.js pin lives in one file

Every workflow that needs Node uses `./.github/actions/setup-node`; none pin a
`node-version:` literal. The composite is the single place the Node and npm
versions are declared, and it must stay in step with `.nvmrc` and the `engines` /
`packageManager` fields in `package.json`. Release and CI therefore build on the
same Node version by construction rather than by convention.

## CI Must Pass Before Merge

`main` is protected. Push a feature branch, open a PR, and let CI gate it:

```bash
gh pr create --fill
gh pr merge --auto --squash --delete-branch
```

Fix issues locally first:

```bash
make all       # Full local verification
make verify    # lint, test, security, build, schema
make test-e2e  # Build and run frontend E2E tests against the HTTPS daemon
```

## Running CI Checks Locally

### Backend

```bash
make lint-backend      # golangci-lint v2.12.2
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
