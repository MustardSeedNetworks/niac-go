# Documentation Review

## 1. Versioning story is inconsistent across artifacts (High)
- **Location**: `README.md:6-10`, `cmd/niac/root.go:8-22`, `CHANGELOG.md:17-140`
- **Issue**: The README (badge and "Current Version" callout) advertises 1.20.0, the code embeds `version = "v1.19.0"`, and the changelog lists 1.21.0 before 1.18.0 and then 1.20.0, which breaks reverse chronological order. Users cannot tell what they are running or which release notes apply.
- **Recommendation**: Align the version constant, README badge/callout, and changelog ordering before tagging the next release. Keeping the changelog strictly descending prevents future confusion.

## 2. CLI reference omits several supported commands (Medium)
- **Location**: `docs/CLI_REFERENCE.md:7-120`, `cmd/niac/config.go:1-130`, `cmd/niac/generate.go:15-70`, `cmd/niac/completion.go:9-63`, `cmd/niac/man.go:11-49`
- **Issue**: The reference only documents `validate`, `template`, and `interactive`, yet the CLI exposes additional subcommands (`config` with export/diff/merge/generate, `completion`, `man`, `niac --list-devices`, etc.). Operators reading the CLI guide have no idea these features exist.
- **Recommendation**: Expand the CLI reference with entries for the `config` sub-tree (including `generate`), shell completion, and man-page generation, plus flag-based utilities such as `--list-devices` and profiling.

## 3. Architecture doc is stale compared to the code (Medium)
- **Location**: `docs/ARCHITECTURE.md:161-179`, `pkg/protocols/stack.go:18-118`, `pkg/config/config.go:1454-1486`
- **Issue**: The document still shows `Stack` as `handlers []Handler`, but the actual struct uses explicit handler fields and a send/receive queue pipeline. It also references `validateWalkFilePath` at `config.go:1377`, whereas the implementation lives around line 1454. This erodes trust in the architecture doc when onboarding engineers.
- **Recommendation**: Refresh the architecture section with the real `Stack` structure (queues, stop channel, handler members) and fix outdated file references.

## 4. Changelog sections are out of chronological order (Low)
- **Location**: `CHANGELOG.md:17-140`
- **Issue**: Versions are listed as 1.21.0 → 1.18.0 → 1.20.0, which violates Keep-a-Changelog expectations and makes diffs between releases hard to follow.
- **Recommendation**: Reorder the sections monotonically (1.21, 1.20, 1.19, 1.18, …) whenever new notes are added.
