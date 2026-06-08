package main

import (
	"fmt"
	"os"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// loadConfigOrScenario loads a literal config file first. If the path
// does not exist, it falls back to a built-in template or installed
// library network with the same name. This gives Go the Java demo
// wrapper's "run a named scenario" convenience without changing the
// normal file-path behavior.
func loadConfigOrScenario(ref string) (*config.Config, string, error) {
	if _, err := os.Stat(ref); err == nil {
		cfg, loadErr := config.Load(ref)
		return cfg, ref, loadErr
	} else if !os.IsNotExist(err) {
		return nil, ref, fmt.Errorf("check config path %s: %w", ref, err)
	}

	if tmpl, err := templates.Get(ref); err == nil {
		cfg, loadErr := config.LoadYAMLBytes([]byte(tmpl.Content))
		return cfg, "template:" + tmpl.Name, loadErr
	}

	lib, err := library.Open(library.DefaultRoot())
	if err != nil {
		return nil, ref, fmt.Errorf("config file not found and content library unavailable: %w", err)
	}
	network, err := lib.ReadNetwork(ref)
	if err != nil {
		return nil, ref, fmt.Errorf("config file or scenario not found: %s", ref)
	}
	cfg, loadErr := config.LoadYAMLBytes([]byte(network.Content))
	return cfg, "library:" + network.Name, loadErr
}
