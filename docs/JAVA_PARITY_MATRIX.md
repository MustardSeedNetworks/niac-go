# Java Parity Matrix

This document records the minimum user-facing parity target for NIAC Go.
The Java implementation is the legacy/reference implementation, so NIAC Go
CLI, TUI, and Web UI must cover at least the same operator capabilities before
a V1 release.

## Scope

Java has two relevant user-facing surfaces:

- Java application entry point: `fluke.niac.Niac`
- Generated demo wrapper scripts: `demo_configs/niac` and `demo_configs/niac.bat`

The Go implementation may expose a capability differently, but it must not
leave users worse off for the same baseline workflow.

## Minimum Capability Matrix

| Java capability | Java surface | NIAC Go CLI | NIAC Go TUI | NIAC Go Web UI | Status |
| --- | --- | --- | --- | --- | --- |
| Run simulation on a selected interface with a config file | `Niac <interface> <network.cfg>` | `niac run <iface> <config>` | `niac interactive <iface> <config>` | `/runtime` interface picker + config picker | Covered |
| Set debug level 0-3 | `-d<n>` | `--debug`, `--quiet`, `--verbose` | Debug level visible/cyclable in TUI | Protocol debug page and runtime/debug APIs | Covered |
| Run without interactive controls | default Java run | `niac run` | Not applicable | Start simulation from `/runtime` | Covered |
| Run with interactive error injection | `-i`, `--interactive` | `niac interactive` or `niac run --tui` | Interactive error injection panel/keys | Traffic/error injection page | Covered |
| Inject common SNMP/interface errors during a live run | Java interactive menu | `niac inject ...` | Interactive error injection | `/traffic` error injection panel | Covered |
| Clear one or all injected errors | Java interactive menu | `niac inject clear` | Clear active/all errors | Error injection panel clear actions | Covered |
| Modify interface speed/duplex during a live run | Java interactive menu | `niac config interface set ...` updates speed/duplex/status metadata | Device Config Interfaces tab cycles speed/duplex/admin status | Device editor Interfaces section persists `interface_details`; API calls `ApplyConfig` after save | Covered |
| Show active injected errors | Java interactive menu | `niac inject list`, `niac status` | Active errors panel | Error injection/status pages | Covered |
| List available adapters/interfaces | Java no-arg run prints adapters | `niac list interfaces` and legacy `niac --list-interfaces` | Runtime interface is shown after launch | `/runtime` and packet inspector use `/api/v1/interfaces` | Covered |
| Show usage/help/version | Java `--help`, `--version`, usage on missing args, demo `help` | Cobra help/man/completion/version | Help panel/keys | Page help drawer | Covered |
| Run named demo scenario | Demo wrapper `run <scenario>` | `niac run <iface> <scenario>` and `niac interactive <iface> <scenario>` resolve built-in templates or library networks | Template browser support | Template/config picker support | Covered |
| List demo scenarios | Demo wrapper `list scenarios` | `niac list scenarios` | Template browser support | Template picker/library pages | Covered |
| List device walks by vendor | Demo wrapper `list walks`, `walk <vendor>` | `niac list walks [vendor-or-prefix]` | SNMP walk browser exists | Library walks page | Covered |
| List packet captures | Demo wrapper `list captures` | `niac list captures` | PCAP replay support exists | Library PCAPs + packet inspector | Covered |
| Convert text walk to binary MIB zip | `fluke.niac.MibZip` | `niac mibzip compress`, `expand`, and `inspect` | Not applicable | Not applicable | Covered |

## Required Before V1

Completed CLI parity work:

- `niac list interfaces`, `niac list scenarios`, `niac list walks`, and
  `niac list captures` cover the Java/demo listing workflows.
- `niac run <iface> <scenario>` and `niac interactive <iface> <scenario>`
  resolve built-in templates and default-library network names in addition to
  direct config file paths.
- `niac mibzip compress`, `niac mibzip expand`, and `niac mibzip inspect`
  cover the legacy Java `fluke.niac.MibZip` workflow and add validation
  utility.
- Interface speed, duplex, and status metadata now round-trip through YAML,
  API device responses, API device updates, CLI config editing, TUI controls,
  and Web UI device editing.

No Java CLI parity blockers are currently open.

## Release Rule

NIAC Go can exceed Java, but it should not regress the baseline Java workflow.
For every Java demo or operator task, there must be one clear Go path in at
least the relevant surface:

- CLI for scripted/operator workflows
- TUI for terminal/live-demo workflows
- Web UI for guided/browser workflows
