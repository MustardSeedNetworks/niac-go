// Package userdir resolves the home directory that per-user state belongs in.
package userdir

import "os/user"

// ResolveHome returns the home directory that per-user state belongs in.
//
// Under sudo the process runs as root while the operator's own configuration
// lives in their home directory, so prefer the invoking user's home whenever
// sudo reports one. Writing to /root instead would strand state the operator
// cannot find and cannot easily read back without sudo.
func ResolveHome(current *user.User, sudoUser string) string {
	if current.Uid == "0" && sudoUser != "" && sudoUser != "root" {
		if invoking, err := user.Lookup(sudoUser); err == nil && invoking.HomeDir != "" {
			return invoking.HomeDir
		}
	}
	return current.HomeDir
}
