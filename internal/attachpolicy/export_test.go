// SPDX-License-Identifier: BUSL-1.1

//go:build unix

package attachpolicy

import "io/fs"

// JudgeTrust and MaxFileBytes expose the trust decision and the size bound to
// the external tests.
func JudgeTrust(mode fs.FileMode, uid, owner uint32) error { return judgeTrust(mode, uid, owner) }

const MaxFileBytes = maxFileBytes
