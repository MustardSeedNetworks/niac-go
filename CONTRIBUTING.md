# Contributing to NIAC

Thanks for your interest in contributing.

## Toolchain

| Tool       | Version  | Why                                                |
|------------|----------|----------------------------------------------------|
| Go         | 1.25.5+  | All backend code (CGO + libpcap)                   |
| Node.js    | 25.2.1+  | UI build (vite, biome)                             |
| libpcap-dev| latest   | `gopacket/pcap` requires it. `apt install libpcap-dev` (Linux), `brew install libpcap` (macOS) |
| nfpm       | v2.46+   | Builds `.deb` + `.rpm` cross-arch (`make deb` / `make rpm`) |
| golangci-lint | v2.12.1 | Pinned in CI; `make tools` installs it locally   |

## First-time setup

```sh
git clone git@github.com:krisarmstrong/niac-go.git
cd niac-go
make tools           # installs husky hooks, golangci-lint, nfpm, etc.
make build           # frontend + backend
```

`make tools` wires up the husky git hooks (`pre-commit` runs gitleaks +
size limits; `commit-msg` runs commitlint). If a hook fails, fix the
underlying issue rather than `--no-verify`-ing past it.

## Branching and commits

- Branch from `main`. Naming: `feat/...`, `fix/...`, `chore/...`,
  `ci/...` — matches the conventional commit type.
- Commit messages must follow
  [Conventional Commits](https://www.conventionalcommits.org/). The
  husky `commit-msg` hook enforces this via `commitlint.config.js`.
  Allowed types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`,
  `test`, `chore`, `ci`, `build`, `revert`.

  Examples:
  ```
  feat(snmp): add bulk walk support
  fix(capture): handle packet overflow on arm64
  chore(deps): bump gopacket to v1.5.0
  ci(release): switch nfpm pin to v2.46.3
  ```

- PR titles follow the same Conventional Commits format. The
  `Title Lint` workflow comments on PRs that don't match.

## Pre-commit checks

The husky `pre-commit` hook runs:

1. **gitleaks** — secret scan on staged files.
2. **scripts/check-file-size.sh** — Go files capped at 800 lines hard
   (warn at 500), TS files at 500 hard (warn at 300). Functions: 50
   lines / 40 statements (`funlen` linter).

If a hook fails, fix the file. Don't commit with `--no-verify` unless
you've discussed it.

## Local quality gates

Before pushing:

```sh
make lint            # golangci-lint + biome (zero warnings)
make test            # go test ./... + ui vitest + playwright e2e (smoke)
```

CI runs the same gates; making them green locally first saves runner
minutes.

## Local packaging

Both formats use the same `.nfpm.yaml` that CI uses, so output matches
byte-for-byte:

```sh
make deb             # → dist/niac_<ver>_amd64.deb
make rpm             # → dist/niac-<ver>-1.x86_64.rpm
make packages        # both
```

Linux deployments to test servers:

```sh
make deploy-ubuntu HOST=niac-srv-ubuntu
make deploy-fedora HOST=niac-srv-fedora
```

## Pull requests

1. Create a feature branch.
2. Write tests. New code should not lower the coverage threshold in
   `ci.yml` (currently 40%, trending toward 90%).
3. `make lint test` locally.
4. Push and open a PR. The PR template walks you through the rest.
5. CI must be green and at least one CODEOWNERS approval before merge.

## Releasing

Releases are tagged manually from `main`. The release pipeline supports
a dry-run mode so you can validate end-to-end without burning a tag:

```sh
# Dry-run on the current branch (default dry_run=true).
gh workflow run release.yml --ref my-branch -f dry_run=true

# Watch progress
gh run watch

# Inspect the bundle artifact uploaded by the dry-run
gh run download <run-id> -n release-bundle-dryrun-<run-id>
```

When a real release is ready:

```sh
git tag -a v0.66.NN -m "vX.Y.Z — short subject

Multi-line release notes describing the changes; this becomes the
annotated-tag message and shows up in 'git show'."

git push origin v0.66.NN
```

The push triggers `release.yml` on the tag, which builds all six
platforms (linux-amd64/arm64, darwin-amd64/arm64, windows-amd64/arm64),
signs every artifact with cosign keyless OIDC, generates CycloneDX
SBOMs, and publishes a GitHub release.

## Reporting issues and vulnerabilities

- Public bugs: open an issue using one of the templates under
  `.github/ISSUE_TEMPLATE/`.
- Security vulnerabilities: see [SECURITY.md](./SECURITY.md). **Do not**
  open a public issue for a vulnerability.

## Code of conduct

Participation is governed by the [Code of Conduct](./CODE_OF_CONDUCT.md)
(Contributor Covenant 2.1).
