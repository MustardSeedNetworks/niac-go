//go:build unix

package main

import "os"

func isPrivileged() bool { return os.Geteuid() == 0 }
