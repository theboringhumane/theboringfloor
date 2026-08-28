//go:build !unix

package headless

// runningAsRoot — no euid concept off unix; never add --no-sandbox.
func runningAsRoot() bool { return false }
