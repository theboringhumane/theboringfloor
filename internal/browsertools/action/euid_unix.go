//go:build unix

package action

import "os"

// runningAsRoot — true under uid 0 (chrome needs --no-sandbox there).
func runningAsRoot() bool { return os.Geteuid() == 0 }
