# Distribution

GitHub Actions is the only release build environment, and GitHub Releases is
the canonical distribution channel. Release Please owns version selection,
changelog PRs, tags, and release creation. A `v*` tag starts
`.github/workflows/release.yml`.

## Release Artifacts

GoReleaser Cross builds the Linux and macOS archives from one pinned container.
Windows binaries are built on native GitHub Windows runners with CGO and the
Npcap SDK, then added to the same release. The workflow publishes:

- Linux amd64 and arm64 archives and packages
- macOS arm64 archive (Apple Silicon only)
- Windows amd64 and arm64 archives
- checksums, SBOMs, signatures, and build provenance

The workflow does not produce an Intel macOS build or a macOS installer
package. `deploy/macos/build-pkg.sh` is a development-validation helper only;
its output is never a release artifact.

## Release Flow

1. Merge reviewed fixes into `main` through required CI.
2. Review and merge the Release Please PR.
3. Release Please creates the GitHub release and `v*` tag.
4. The release workflow builds and attaches all artifacts.
5. Install the published Linux packages on the Ubuntu and Fedora targets.
6. Query each installed service over its default loopback listener with
   `ssh <server> 'curl -sk https://localhost:8445/__version' | jq` and verify
   the response reports the tag, commit, build time, and a non-empty UI build
   hash.

Do not create release tags or GitHub releases by hand except for an explicitly
approved release-recovery operation.

## Local Validation

Local builds prove source quality. They never produce or replace release
artifacts; all release builds run in GitHub Actions.

```bash
make lint
make fmt-check
make test
make test-e2e
make security
make build
```
