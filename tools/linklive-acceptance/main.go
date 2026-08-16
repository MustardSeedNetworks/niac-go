// Command linklive-acceptance drives a running niac simulation against a
// Link-Live inventory export and reports topology/finding mismatches,
// exercising the comparator in internal/acceptance/linklive as a CI gate.
package main

import (
	"context"
	"fmt"
	"os"
)

func main() {
	runner := newRunner(os.Getenv, os.Stdout)
	if err := runner.run(context.Background(), os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
