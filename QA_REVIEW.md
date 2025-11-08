# QA Assessment

## Test Runs
- `go test ./... -coverprofile=coverage.txt`
  - All packages compile and their unit tests pass, but several packages report `0.0%` coverage (the Go tool prints this per package during the run). The generated `coverage.txt` is available for further inspection.

## Gaps & Risks

1. **Config management commands lack regression tests (Medium)**  
   The logic behind `niac config export/diff/merge` and the interactive generator (`cmd/niac/config.go:1-200`, `cmd/niac/generate.go:15-160`) is completely untested. There are no `_test.go` companions covering success/error flows, so changes to file I/O, YAML normalization, or diff logic can break without detection. Adding golden-file tests around these commands would protect the DevOps-centric workflows highlighted in the README.

2. **Converter module ships without automated coverage (Medium)**  
   `internal/converter/converter.go` (~600 lines) performs the DSL→YAML translation and validation, yet `go test` reports `github.com/krisarmstrong/niac-go/internal/converter	coverage: 0.0% of statements`. Without unit tests around edge cases (invalid tokens, malformed leases, protocol blocks), format regressions will only be caught manually.

3. **No automated signal/shutdown coverage for capture + protocol stack (High)**  
   Critical runtime code in `pkg/capture/capture.go:21-107` and `pkg/protocols/stack.go:167-241` runs only in privileged environments, so there are no tests that validate graceful shutdown, stop-channel semantics, or interaction with `pcap`. This blind spot is what allowed the current Ctrl+C hang to ship. Consider abstracting the capture engine behind an interface so that stop/start behavior and queue draining can be exercised in CI without a live interface.
