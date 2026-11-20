// SPDX-License-Identifier: BUSL-1.1

//go:build unix

package attachpolicy_test

import "syscall"

func setUmask(mask int) int { return syscall.Umask(mask) }
