# CI/CD Setup for NIAC-Go

This document describes the continuous integration and continuous deployment setup for NIAC-Go.

## Overview

NIAC-Go uses GitHub Actions for automated testing and release building. Every commit is tested automatically, and releases are built when version tags are pushed.

## GitHub Actions Workflows

### 1. Test Workflow (`.github/workflows/test.yml`)

**Triggers:** Push to `master/main` branch, Pull Requests

**Jobs:**
- **test**: Runs all tests with race detection and coverage
  - Go version: 1.21
  - Runs: `go test -v -race -coverprofile=coverage.out -covermode=atomic ./...`
  - Generates coverage report
  - Enforces 40% minimum coverage threshold

- **lint**: Code quality checks
  - `go fmt` - Code formatting
  - `go vet` - Static analysis
  - `staticcheck` - Additional linting

- **build**: Verifies binary builds successfully
  - Compiles `./cmd/niac`
  - Tests binary execution

### 2. Release Workflow (`.github/workflows/release.yml`)

**Triggers:** Push of version tags (`v*.*.*`)

**Platforms:**
- Linux (amd64, arm64)
- macOS/Darwin (amd64, arm64)
- Windows (amd64)
- FreeBSD (amd64)

**Process:**
1. Builds binaries for all platforms
2. Generates SHA256 checksums
3. Creates GitHub release with all artifacts
4. Tests binaries on Ubuntu, macOS, and Windows

**Release artifacts:**
- `niac-vX.Y.Z-{platform}-{arch}[.exe]` - Binary
- `niac-vX.Y.Z-{platform}-{arch}[.exe].sha256` - Checksum

## Pre-commit Hooks

Local git hook that runs before each commit to catch issues early.

### Installation

```bash
./scripts/install-hooks.sh
```

### What it checks

1. **Code Formatting** - Ensures all Go files are `gofmt` formatted
2. **Static Analysis** - Runs `go vet` to catch common mistakes
3. **Tests** - Runs all tests with race detection
4. **Coverage** - Reports current test coverage (warns if below 40%)
5. **Build** - Verifies the project builds successfully

### Skipping the hook

If you need to commit without running checks:
```bash
git commit --no-verify
```

## Coverage Requirements

- **Minimum:** 40% (enforced in CI)
- **Current:** 44.8% (v1.8.0)
- **Target:** Maintain or improve coverage with new code

## Best Practices

1. **Run tests locally** before pushing
   ```bash
   go test -v -race ./...
   ```

2. **Check formatting** before committing
   ```bash
   gofmt -w .
   ```

3. **Run the full pre-commit check** manually
   ```bash
   .git/hooks/pre-commit
   ```

4. **Tag releases** using semantic versioning
   ```bash
   git tag -a v1.8.0 -m "Release v1.8.0"
   git push origin v1.8.0
   ```

## Automated Release Process

When you're ready to release:

1. Update `CHANGELOG.md` with release notes
2. Commit all changes
3. Tag the release:
   ```bash
   git tag -a v1.8.0 -m "Release v1.8.0: Testing & Bug Fixes"
   ```
4. Push the tag:
   ```bash
   git push origin v1.8.0
   ```
5. GitHub Actions automatically:
   - Builds binaries for all platforms
   - Runs tests on each platform
   - Creates GitHub release
   - Attaches all binaries and checksums

## Troubleshooting

### Tests fail in CI but pass locally
- Ensure you're using Go 1.21
- Check for race conditions: `go test -race ./...`
- Verify all dependencies are in `go.mod`

### Pre-commit hook fails
- Run the suggested command (usually `gofmt -w .`)
- Fix any `go vet` issues
- Ensure all tests pass

### Release build fails
- Verify the version tag format: `v1.2.3`
- Check `CHANGELOG.md` has an entry for the version
- Ensure all tests pass on `master` branch

## Files

- `.github/workflows/test.yml` - Test automation
- `.github/workflows/release.yml` - Release automation
- `.githooks/pre-commit` - Local pre-commit checks
- `scripts/install-hooks.sh` - Hook installation script
