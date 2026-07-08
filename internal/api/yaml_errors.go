package api

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// yamlLinePattern matches the "line N" fragment that gopkg.in/yaml.v3 embeds
// in its error strings (both plain syntax errors and *yaml.TypeError). yaml.v3
// does not expose a structured line/column field for syntax errors, so this is
// the only reliable way to recover the offending line for UI highlighting.
const yamlLinePattern = `line (\d+)`

// parseYAMLError extracts a 1-based line number and a human-readable message
// from a gopkg.in/yaml.v3 parse/unmarshal error, unwrapping any %w context
// (e.g. "failed to parse YAML config: ...") added by callers first. Line is 0
// when the underlying error carries no line information (yaml.v3 omits it for
// some syntax errors, e.g. "mapping values are not allowed in this context"
// at the top of a document). The returned message never includes the
// "yaml: " prefix so it reads naturally in API responses and the UI.
func parseYAMLError(err error) (int, string) {
	if err == nil {
		return 0, ""
	}

	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) && len(typeErr.Errors) > 0 {
		// Report the first sub-error; yaml.v3 already sorts these by
		// document order, so it is the earliest problem encountered.
		return extractYAMLLine(typeErr.Errors[0]), typeErr.Errors[0]
	}

	root := rootCause(err).Error()

	return extractYAMLLine(root), stripYAMLPrefix(root)
}

// rootCause walks the standard-library error-wrapping chain (errors.Unwrap)
// down to the innermost error, which for a YAML parse failure is the
// original gopkg.in/yaml.v3 error before any caller-added "failed to ...: "
// context.
func rootCause(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}

		err = unwrapped
	}
}

// extractYAMLLine pulls the first "line N" occurrence out of a yaml.v3 error
// string. Returns 0 when no line number is present.
func extractYAMLLine(msg string) int {
	re := regexp.MustCompile(yamlLinePattern)

	match := re.FindStringSubmatch(msg)
	if match == nil {
		return 0
	}

	n, convErr := strconv.Atoi(match[1])
	if convErr != nil {
		return 0
	}

	return n
}

// stripYAMLPrefix removes the leading "yaml: " prefix that gopkg.in/yaml.v3
// adds to top-level errors, so messages read naturally in API responses.
func stripYAMLPrefix(msg string) string {
	return strings.TrimPrefix(msg, "yaml: ")
}
