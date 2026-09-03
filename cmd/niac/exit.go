package main

import "os"

// exitProcess terminates the process. Every command's failure path goes through
// this rather than calling os.Exit directly, so a test can substitute it and
// assert what a command does when it fails.
//
// Production behaviour is unchanged: os.Exit never returns, so the statements
// after a call are unreachable. A substitute must preserve that — returning
// normally would let a failing command run on past the point where it would
// really have died, and the test would assert against a state the binary can
// never be in. See withExitCapture in the tests.
var exitProcess = os.Exit
