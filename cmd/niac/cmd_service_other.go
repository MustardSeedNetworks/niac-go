//go:build !windows

package main

import "github.com/spf13/cobra"

// addServiceCommand is a no-op on non-Windows platforms.
// Windows service management is only available on Windows.
func addServiceCommand(_ *cobra.Command, _ *serviceOptions) {}
