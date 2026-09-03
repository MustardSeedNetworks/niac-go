package api

import (
	"errors"
	"strings"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/converter"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// validateScenarioYAML runs the whole authoring check -- semantic validation
// followed by the configuration half of the fabric compile -- over raw YAML,
// and returns one detail per finding.
//
// This is the single entry point every authoring surface uses, so a file that
// `niac validate` accepts is a file the daemon will start, and a file either
// one refuses is refused everywhere with the same codes (P1b-4). Surfaces used
// to disagree: the library accepted whatever parsed, and `niac validate` never
// ran the compiler at all.
func validateScenarioYAML(content []byte) []ErrorDetail {
	cfg, err := config.LoadYAMLBytes(content)
	if err != nil {
		if details := structTagDetails(err); details != nil {
			return details
		}
		line, message := parseYAMLError(err)
		return []ErrorDetail{{Issue: message, Line: line}}
	}
	return validateScenarioConfig(cfg)
}

// validateScenarioConfig is validateScenarioYAML for an already-parsed config.
func validateScenarioConfig(cfg *config.Config) []ErrorDetail {
	if result := config.NewValidator("").Validate(cfg); result.HasErrors() {
		return validationErrorDetails(result)
	}
	// A flat scenario carries no networks for the compiler to resolve against,
	// and reading its interfaces would invent findings no other surface
	// reports. fabric.IsRouted is the one predicate that decides this.
	if !fabric.IsRouted(cfg) {
		return nil
	}
	report := fabric.CompileConfig(cfg)
	if report.Safe {
		return nil
	}
	return diagnosticDetails(report.Diagnostics)
}

// diagnosticDetails renders compiler findings as error details, keeping the
// stable code so a client can classify a finding without matching its text.
func diagnosticDetails(diagnostics []fabric.Diagnostic) []ErrorDetail {
	details := make([]ErrorDetail, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		details = append(details, ErrorDetail{
			Code:  string(diagnostic.Code),
			Field: diagnostic.Field,
			Issue: diagnostic.Message,
		})
	}
	return details
}

// structTagListSeparator is how converter.validateConfigStruct joins its
// findings into one error string.
const structTagListSeparator = "\n  - "

// structTagDetails splits a struct-tag validation failure back into one detail
// per finding. The converter formats its findings into a single error and the
// wrapping chain bottoms out at the bare sentinel, so unwrapping loses the
// list entirely -- a caller that reported the root cause showed the operator
// "config validation failed" and nothing else. These findings carry no stable
// code: unlike the fabric compiler, the struct-tag layer has no code
// vocabulary, so Code is left empty rather than invented.
//
// Returns nil when err is not a struct-tag failure.
func structTagDetails(err error) []ErrorDetail {
	if !errors.Is(err, converter.ErrConfigInvalid) {
		return nil
	}
	_, list, found := strings.Cut(err.Error(), converter.ErrConfigInvalid.Error()+":")
	if !found {
		return nil
	}
	var details []ErrorDetail
	for finding := range strings.SplitSeq(list, structTagListSeparator) {
		finding = strings.TrimSpace(finding)
		if finding == "" {
			continue
		}
		field, issue, hasField := strings.Cut(finding, ": ")
		if !hasField {
			details = append(details, ErrorDetail{Issue: finding})
			continue
		}
		details = append(details, ErrorDetail{
			Field: strings.TrimPrefix(field, "Config."), Issue: issue,
		})
	}
	return details
}
