package main

import (
	"errors"
	"testing"
	"time"
)

func TestOnceArgsAcceptsThePositionalForm(t *testing.T) {
	options := &daemonOptions{}

	iface, configPath, err := onceArgs(options, []string{"lo0", "clinic.yaml"})
	if err != nil {
		t.Fatalf("onceArgs: %v", err)
	}
	if iface != "lo0" || configPath != "clinic.yaml" {
		t.Fatalf("got (%q, %q), want (lo0, clinic.yaml)", iface, configPath)
	}
}

// The flags are the alternative to positionals, so a caller scripting this
// does not have to build an argv in a fixed order.
func TestOnceArgsFallsBackToFlags(t *testing.T) {
	options := &daemonOptions{onceInterface: "eth0", onceConfig: "from-flags.yaml"}

	iface, configPath, err := onceArgs(options, nil)
	if err != nil {
		t.Fatalf("onceArgs: %v", err)
	}
	if iface != "eth0" || configPath != "from-flags.yaml" {
		t.Fatalf("got (%q, %q), want (eth0, from-flags.yaml)", iface, configPath)
	}
}

func TestOnceArgsRequiresAnInterfaceAndAConfig(t *testing.T) {
	for name, options := range map[string]*daemonOptions{
		"neither":     {},
		"config only": {onceConfig: "only.yaml"},
		"iface only":  {onceInterface: "lo0"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := onceArgs(options, nil); err == nil {
				t.Fatal("onceArgs() = nil error, want a usage error")
			}
		})
	}
}

func TestOnceArgsRejectsExtraPositionals(t *testing.T) {
	if _, _, err := onceArgs(&daemonOptions{}, []string{"lo0", "a.yaml", "b.yaml"}); err == nil {
		t.Fatal("onceArgs() accepted three positional arguments")
	}
}

// A run that ends because its duration elapsed is a success; one ended by a
// signal is reported as such, so a caller can tell a completed soak from an
// interrupted one.
func TestWaitForOnceReportsTheDurationEnding(t *testing.T) {
	start := time.Now()

	if got := waitForOnce(20 * time.Millisecond); got != "duration" {
		t.Fatalf("waitForOnce() = %q, want %q", got, "duration")
	}
	if elapsed := time.Since(start); elapsed < 20*time.Millisecond {
		t.Fatalf("waitForOnce returned after %v, want at least the duration", elapsed)
	}
}

// The exit codes are the contract a script branches on: a config the daemon
// refused must not look like a run that crashed.
func TestOnceExitCodesAreDistinct(t *testing.T) {
	if onceExitConfig == onceExitRuntime {
		t.Fatal("config and runtime failures share an exit code; a caller cannot tell them apart")
	}
	if onceExitOK != 0 {
		t.Fatalf("success exit code = %d, want 0", onceExitOK)
	}

	err := withExitCode(onceExitConfig, errors.New("bad config"))
	var coded codedError
	if !errors.As(err, &coded) || coded.code != onceExitConfig {
		t.Fatalf("withExitCode did not carry the config exit code: %#v", err)
	}
}
