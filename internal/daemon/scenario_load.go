package daemon

import (
	"fmt"

	"github.com/MustardSeedNetworks/niac-go/internal/api/builtins"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
)

// loadNamedScenario loads a scenario by bare name from the scenario
// directories, then the managed roots (which hold the content library).
func loadNamedScenario(name string) (*config.Config, string, error) {
	// Loading built-in scenarios by name preserves the scenario's own
	// directory as the include_path base — needed for vendor
	// scenarios with `include_path: ".."` plus relative walk_file
	// refs that resolve to sibling directories (e.g. examples/
	// device_walks_sanitized/...). Fetching the YAML text and
	// POSTing it as ConfigData would lose that context.
	roots := simulationConfigRoots()
	scenarioPath := builtins.Find(name)
	if scenarioPath == "" {
		// The scenario directories builtins.Find walks are a
		// development-tree and system-install idea; the deb and rpm
		// create none of them. What an installed host has is the
		// library, which first run seeds with exactly the scenarios
		// `niac scenario list` advertises -- and which is also where
		// their `include_path: ../walks` resolves from, so loading one
		// from there keeps the directory context this case exists for.
		// Composed here rather than inside builtins.Dirs(), which is a
		// leaf over stdlib and may not know about the library.
		//
		// Only existence is asked here; the name itself is handed on, so
		// the load resolves and confines it exactly as a named ConfigPath
		// is. Passing the resolved real path instead would fail the
		// pre-symlink containment check wherever a root sits under a
		// symlinked directory, which is every macOS /var.
		//
		// The library lists a network under its bare name whichever
		// extension the file has, so both are tried.
		for _, ext := range []string{".yaml", ".yml"} {
			candidate := name + ext
			if _, libraryErr := config.ResolveManagedConfigPath(candidate, roots); libraryErr == nil {
				scenarioPath = candidate
				break
			}
		}
		if scenarioPath == "" {
			return nil, "", fmt.Errorf("%w: %s", ErrScenarioNotFound, name)
		}
	}
	cfg, managedPath, err := config.LoadYAMLManaged(scenarioPath, roots)
	if err != nil {
		return nil, "", fmt.Errorf("load scenario %q: %w", name, err)
	}
	return cfg, managedPath, nil
}
