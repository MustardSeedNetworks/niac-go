# Dependency Notes

NiAC targets the current project toolchain:

- Go: 1.26.4 or newer within the 1.26 line.
- Node.js: 26.5.0 or newer.
- npm: 11.7.0 or newer.
- Tailwind CSS: 4.3.3.

The UI dependency graph is pinned in `ui/package-lock.json` and should be updated with npm so the lockfile remains authoritative. Use the repo's Node version requirement before running npm commands; older local Node 26 builds can install and test successfully, but npm will warn if the runtime is below `26.5.0`.

`npm audit` is expected to be clean for production and development dependencies. Some development-only packages may publish conservative `engines.node` metadata before they officially list Node 26 support. Treat those as upstream metadata warnings when install, audit, typecheck, build, and browser tests all pass.

Go modules should follow Go's minimal version selection. Update direct and meaningful indirect modules through `go get` and `go mod tidy`, but avoid pinning unused transitive modules only to make `go list -m -u all` silent.
