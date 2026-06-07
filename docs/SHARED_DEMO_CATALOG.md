# Shared NIAC Demo Catalog

The large Go `examples/` corpus is generated from the shared NIAC demo catalog:

```text
git@github.com:MustardSeedNetworks/niac-demo-catalog.git
```

This keeps the Go and Java repositories aligned without committing the same
walks, captures, and demo scenarios in every implementation repo.

## Generate Examples

Linux/macOS:

```sh
./scripts/sync-demo-catalog.sh --sync
```

Windows PowerShell:

```powershell
.\scripts\sync-demo-catalog.ps1 -Mode Sync
```

Check an existing generated `examples/` tree:

```sh
./scripts/sync-demo-catalog.sh --check
```

```powershell
.\scripts\sync-demo-catalog.ps1 -Mode Check
```

## Generated Layout

| Catalog path | Go generated path |
| --- | --- |
| `scenarios/go-yaml/` | `examples/` |
| `walks/raw/` | `examples/device_walks/` |
| `walks/sanitized/` | `examples/device_walks_sanitized/` |
| `captures/shared/` | `examples/captures/` |
| `captures/go-extra/` | `examples/pcaps/` |
| `tools/walk-scripts/go/` | `examples/walk_scripts/` |
| `tools/walk-scripts/java/run_demo.sh` | `examples/walk_scripts/run_demo.sh` |
| `docs/imported/go-examples/` | `examples/` |

## Environment

- `NIAC_DEMO_CATALOG_URL`: catalog git URL.
- `NIAC_DEMO_CATALOG_REF`: catalog branch, tag, or commit. Defaults to `main`.
- `NIAC_DEMO_CATALOG_DIR`: existing catalog checkout. Defaults to `.catalog/niac-demo-catalog`.
- `NIAC_GO_EXAMPLES_DIR`: generated output path. Defaults to `examples`.
- `NIAC_DEMO_CATALOG_OFFLINE=1`: require an existing local catalog and skip `git fetch`.

Do not add shared demo assets directly to this repo. Add or update them in
`niac-demo-catalog`, then regenerate `examples/`.
