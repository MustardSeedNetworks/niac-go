package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/MustardSeedNetworks/niac-go/internal/library"
	"github.com/MustardSeedNetworks/niac-go/internal/templates"
)

// configSource is a config document plus where it came from.
type configSource struct {
	data []byte
	// label names the source for the run summary: a file path, or
	// "template:<name>" / "library:<name>" for a resolved scenario.
	label string
	// path is the file the document was read from, empty for a resolved
	// scenario. The daemon resolves relative capture-playback paths against
	// it, so a name that is not a file must not supply one.
	path string
}

// resolveConfigSource reads a literal config file first. If the path does not
// exist, it falls back to a built-in template or an installed library network
// of the same name -- the Java demo wrapper's "run a named scenario"
// convenience, kept when `run` was deleted in favour of `daemon --once`.
//
// It returns the document unparsed: the daemon owns validation, preflight and
// admission, and parsing here as well would be a second opinion that could
// disagree with the one that matters.
func resolveConfigSource(ref string) (configSource, error) {
	if _, err := os.Stat(ref); err == nil {
		data, readErr := os.ReadFile(filepath.Clean(ref))
		if readErr != nil {
			return configSource{}, fmt.Errorf("reading %s: %w", ref, readErr)
		}

		return configSource{data: data, label: ref, path: ref}, nil
	} else if !os.IsNotExist(err) {
		return configSource{}, fmt.Errorf("check config path %s: %w", ref, err)
	}

	if tmpl, err := templates.Get(ref); err == nil {
		return configSource{data: []byte(tmpl.Content), label: "template:" + tmpl.Name}, nil
	}

	lib, err := library.Open(library.DefaultRoot())
	if err != nil {
		return configSource{}, fmt.Errorf("config file not found and content library unavailable: %w", err)
	}
	network, err := lib.ReadNetwork(ref)
	if err != nil {
		return configSource{}, fmt.Errorf("config file or scenario not found: %s", ref)
	}

	return configSource{data: []byte(network.Content), label: "library:" + network.Name}, nil
}
