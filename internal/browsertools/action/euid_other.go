//go:build !unix

package action

// runningAsRoot — no euid concept off unix; never add --no-sandbox.
func runningAsRoot() bool { return false }
