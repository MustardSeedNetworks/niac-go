package library

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/MustardSeedNetworks/niac-go/internal/config"
	"github.com/MustardSeedNetworks/niac-go/internal/fabric"
)

// networkProblem loads path exactly as `niac validate` does and returns the
// first reason it would refuse the file, on one line, or "" when it is valid.
//
// The list used to decode only enough YAML to count devices, so a network the
// strict loader rejected was listed as ok and failed on start (#2202).
func networkProblem(path string) string {
	cfg, err := config.Load(path)
	if err != nil {
		return firstLoadError(err)
	}
	result := fabric.Validate(cfg, path)
	if !result.HasErrors() {
		return ""
	}
	first := result.Errors[0]
	if first.Field == "" {
		return first.Message
	}
	return first.Field + ": " + first.Message
}

// firstLoadError reduces a load failure to its first line. A strict decode
// reports every unknown field of the file in one multi-line error; the first
// is enough to tell an operator what to fix.
func firstLoadError(err error) string {
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) && len(typeErr.Errors) > 0 {
		return typeErr.Errors[0]
	}
	line, _, _ := strings.Cut(err.Error(), "\n")
	return line
}
