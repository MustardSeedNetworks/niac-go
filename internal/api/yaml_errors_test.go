package api

import (
	"errors"
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseYAMLError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		wantLine int
		wantMsg  string
	}{
		{
			name:     "nil error",
			err:      nil,
			wantLine: 0,
			wantMsg:  "",
		},
		{
			name:     "syntax error with line number",
			err:      mustYAMLSyntaxError(t, "foo: bar\nbaz: [1, 2\nqux: 3\n"),
			wantLine: 1,
			wantMsg:  "line 1: did not find expected ',' or ']'",
		},
		{
			name:     "syntax error with line number, later in document",
			err:      mustYAMLSyntaxError(t, "- a\n- b\n foo: bar\n"),
			wantLine: 3,
			wantMsg:  "line 3: mapping values are not allowed in this context",
		},
		{
			name:     "syntax error with no line number",
			err:      mustYAMLSyntaxError(t, "foo: : bar\n"),
			wantLine: 0,
			wantMsg:  "mapping values are not allowed in this context",
		},
		{
			name:     "type error uses first sub-error",
			err:      mustYAMLTypeError(t),
			wantLine: 2,
			wantMsg:  "line 2: cannot unmarshal !!str `not-a-n...` into int",
		},
		{
			name: "wrapped error still finds line and strips context",
			err: fmt.Errorf("failed to parse YAML config: %w",
				mustYAMLSyntaxError(t, "foo: bar\nbaz: [1, 2\nqux: 3\n")),
			wantLine: 1,
			wantMsg:  "line 1: did not find expected ',' or ']'",
		},
		{
			name:     "non-yaml error with no line info",
			err:      errNoLineInfo,
			wantLine: 0,
			wantMsg:  "something went wrong",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			line, msg := parseYAMLError(tc.err)
			if line != tc.wantLine {
				t.Errorf("line = %d, want %d", line, tc.wantLine)
			}

			if msg != tc.wantMsg {
				t.Errorf("message = %q, want %q", msg, tc.wantMsg)
			}
		})
	}
}

var errNoLineInfo = errors.New("something went wrong")

// mustYAMLSyntaxError produces a real gopkg.in/yaml.v3 syntax error by
// unmarshaling invalid YAML, so the test exercises the library's actual
// error strings rather than a hand-authored approximation.
func mustYAMLSyntaxError(t *testing.T, invalidYAML string) error {
	t.Helper()

	var data map[string]any
	if err := yaml.Unmarshal([]byte(invalidYAML), &data); err != nil {
		return err
	}

	t.Fatalf("expected yaml.Unmarshal to fail for input %q", invalidYAML)

	return nil
}

// mustYAMLTypeError produces a real *yaml.TypeError by unmarshaling a
// type-mismatched field.
func mustYAMLTypeError(t *testing.T) error {
	t.Helper()

	type target struct {
		Name string `yaml:"name"`
		Age  int    `yaml:"age"`
	}

	var s target
	if err := yaml.Unmarshal([]byte("name: foo\nage: not-a-number\n"), &s); err != nil {
		return err
	}

	t.Fatal("expected yaml.Unmarshal to fail with a type error")

	return nil
}
