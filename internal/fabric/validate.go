package fabric

import "github.com/MustardSeedNetworks/niac-go/internal/config"

// Validate runs the semantic validator and, for a routed scenario, folds the
// configuration-derived compiler findings into the same result, so every
// surface that answers "is this file valid" refuses exactly what the daemon
// refuses to start.
//
// Semantic validation alone passed files that preflight rejected on six
// counts, and validate then printed "Configuration is valid" for a scenario
// the daemon would not run (P1b-4). The compiler's stable codes travel with
// each finding so an operator can match a validate line to a preflight line.
func Validate(cfg *config.Config, file string) *config.ListError {
	result := config.NewValidator(file).Validate(cfg)
	// A flat scenario has no fabric to compile; its interfaces name no
	// network, which the compiler would read as references to a network that
	// does not exist.
	if !IsRouted(cfg) {
		return result
	}
	for _, diagnostic := range CompileConfig(cfg).Diagnostics {
		finding := config.NewConfigError(file, diagnostic.Field, diagnostic.Message)
		finding.Code = string(diagnostic.Code)
		result.Add(finding)
	}
	return result
}
