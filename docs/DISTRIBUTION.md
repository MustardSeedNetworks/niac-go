# Distribution

NIAC distribution is handled by GitHub Releases.

Release Please owns version selection, changelog PRs, release tags, and GitHub
release creation. When a `v*` tag is created, `.github/workflows/release.yml`
builds native artifacts on GitHub-hosted runners:

| Platform | Runner | Artifact |
| --- | --- | --- |
| Linux amd64 | `ubuntu-latest` | `.tar.gz` |
| Linux arm64 | `ubuntu-24.04-arm` | `.tar.gz` |
| macOS amd64 | `macos-13` | `.tar.gz` |
| macOS arm64 | `macos-14` | `.tar.gz` |
| Windows amd64 | `windows-latest` | `.zip` |
| Windows arm64 | `windows-11-arm` | `.zip` |

Each release also publishes `checksums.txt` and a GitHub build-provenance
attestation.

## Local Development

Local builds are for development and validation:

```bash
make build
make test
make verify
```

Local commands do not build release artifacts for other operating systems. The
full release matrix belongs in GitHub Actions so Linux, macOS, and Windows
artifacts are built on their native operating systems.

## Release Flow

1. Merge normal feature/fix commits into `main`.
2. Release Please opens or updates the release PR.
3. Review and merge the release PR.
4. Release Please creates the GitHub release and `v*` tag.
5. The release workflow attaches native artifacts to that release.

Do not create local release tags or local GitHub releases by hand except for an
explicit emergency recovery.
