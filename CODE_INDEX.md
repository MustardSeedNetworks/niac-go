# Code capability index

This index records shared capabilities that must be checked before adding a
parallel implementation. It is intentionally organized by purpose.

## Configuration loading and validation

| Capability | Canonical location | Notes |
| --- | --- | --- |
| YAML authoring DTO | `internal/converter/types.go` | Source for generated YAML schema |
| YAML field validation | `internal/converter/validate.go` | Shape validation using YAML field names |
| Runtime config loading | `internal/config/yaml_load.go` | One file/bytes conversion pipeline |
| Routed YAML adapters | `internal/config/yaml_fabric.go` | Converts authoring DTOs into runtime config |
| Complete-config validation | `internal/config/validator.go` | Existing device validation; not routed semantics |

## Routed fabric and topology

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Routed semantic compiler | `internal/fabric/compiler.go` | Pure config plus binding to immutable topology |
| Routed device compilation | `internal/fabric/compiler_devices.go` | Interfaces, routes, and DHCP ownership |
| Fabric domain and diagnostics | `internal/fabric/types.go` | Physical binding is distinct from virtual networks |
| Device/link UI graph | `internal/topology/topology.go` | Projection only; not a forwarding compiler |
| Physical VLAN engines | `internal/protocols/stack_init.go` | ADR 0008 segments; not routed virtual networks |
| IEEE vendor registry | `internal/oui/registry.go` | Lookup and deterministic isolated-lab MAC allocation |

## Simulation lifecycle API

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Simulation request contract | `internal/api/server.go` | Shared start and preflight request fields |
| Simulation request validation | `internal/api/validation.go` | Interface/config source boundary validation |
| Simulation lifecycle handlers | `internal/api/handlers_simulation.go` | Strict decoder and standard error envelope |
| Route security policy | `internal/api/routes.go` | Methods, rate limits, and CSRF are registered here |
| Config preparation and start | `internal/daemon/daemon.go` | Preflight must not persist or open capture |

## Modern UI reuse points

| Capability | Canonical location | Notes |
| --- | --- | --- |
| Guided simulation flow | `ui/src/pages/NewSimulationWizardPage.tsx` | Extend instead of creating another lab builder |
| Physical binding preflight | `ui/src/components/wizard/PreflightStep.tsx` | Review direct/access attachment and gate start on server diagnostics |
| Config source picker | `ui/src/components/simulation/ConfigPicker.tsx` | Templates, saved networks, and uploads |
| API request boundary | `ui/src/api/client.ts` | Add typed sibling functions here |
| API response types | `ui/src/api/api-response-types.ts` | Handwritten; generated types do not yet exist |
| Runtime topology view | `ui/src/pages/TopologyPage.tsx` | Authored/observed device and link projection |
