package daemon

import (
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/api"
	"github.com/MustardSeedNetworks/niac-go/internal/api/builtins"
	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
)

// resolveExampleRequest turns a request that names a shipped example into one
// that carries its document, so staging, preflight and recovery treat it like
// any other inline configuration.
//
// A name is looked up the way the daemon already resolves a bare library name:
// a scenario directory or a file in a managed root wins, which is what lets an
// operator's own "hospital.yaml" shadow the shipped pack. Only a name nothing on
// disk answers falls through to the examples the binary carries. Anything else,
// including a path, a name no example has and an unreadable example, is returned
// untouched so the ordinary load reports it in the ordinary words.
func resolveExampleRequest(req api.SimulationRequest) api.SimulationRequest {
	name, ok := exampleReference(req)
	if !ok || !library.IsExample(name) || resolvesOnDisk(req, name) {
		return req
	}
	walks := filepath.Join(library.DefaultRoot(), string(library.KindWalks))
	content, err := library.ExampleStartContent(name, walks)
	if err != nil {
		return req
	}
	req.ScenarioName = ""
	req.ConfigPath = ""
	req.ConfigData = string(content)
	return req
}

// exampleReference extracts the bare name a request refers to, if it refers to
// a scenario by name rather than by inline data or by path.
func exampleReference(req api.SimulationRequest) (string, bool) {
	switch {
	case req.ScenarioName != "":
		return config.PlainConfigName(req.ScenarioName)
	case req.ConfigData == "" && req.ConfigPath != "":
		return config.PlainConfigName(req.ConfigPath)
	default:
		return "", false
	}
}

// resolvesOnDisk reports whether the existing resolution finds name: the
// scenario directories, then the managed roots, with and without ".yaml".
func resolvesOnDisk(req api.SimulationRequest, name string) bool {
	if req.ScenarioName != "" && builtins.Find(req.ScenarioName) != "" {
		return true
	}
	roots := simulationConfigRoots()
	for _, candidate := range []string{name + ".yaml", name + ".yml"} {
		if _, err := config.ResolveManagedConfigPath(candidate, roots); err == nil {
			return true
		}
	}
	return false
}
