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
