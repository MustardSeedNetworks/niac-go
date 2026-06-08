# Generated Examples

This directory is generated from the shared NIAC demo catalog:

```text
git@github.com:MustardSeedNetworks/niac-demo-catalog.git
```

Do not commit shared walks, captures, PCAPs, or generated scenario files directly to this repository. Add or update those assets in `niac-demo-catalog`, then regenerate this directory locally:

```bash
./scripts/sync-demo-catalog.sh --sync
```

On Windows:

```powershell
.\scripts\sync-demo-catalog.ps1 -Mode Sync
```

See [`docs/SHARED_DEMO_CATALOG.md`](../docs/SHARED_DEMO_CATALOG.md) for the catalog layout and sync workflow.
