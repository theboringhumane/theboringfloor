// clipboard.go — SYSTEM-CLIPBOARD writes from inside panels (the
// chat_attach.go shell-out precedent): the terminal selection's copy needs
// a real OS pasteboard, and panels cannot reach internal/app's helpers
// (import cycle), so the platform tool table lives here. Resolution is
// lazy and cached: the FIRST available tool wins, and a miss resolves to
// a nil runner so callers can degrade to a dim note instead of failing
// silently.
//
//	darwin   pbcopy                                   (stdin carries the text)
//	windows  clip                                     (stdin)
//	other    wl-copy · xclip -selection clipboard · xsel --clipboard --input
//
// Tests NEVER exec a real shell-out: they stub the exported-within-package
// seam var clipboardCopyText (the copy gate in terminal.go reads it), and
// must restore it (t.Cleanup) so parallel suites don't leak stubs.
package panels

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
)

// clipboardCopyText is THE write seam: the terminal selection calls it on a
// dragged release and gates its confirmation note on the verdict. The
// default resolves the platform tool lazily; tests swap in a recorder.
var clipboardCopyText = systemClipboardCopy

var (
	clipOnce sync.Once
	clipRun  func(string) error // nil when no tool exists on this host
)

// systemClipboardCopy writes text to the OS pasteboard via the lazily
// resolved tool; the error is wrapped with the tool name so a failed copy
// note can say WHO failed.
func systemClipboardCopy(text string) error {
	clipOnce.Do(func() {
		type tool struct {
			name string
			args []string
		}
		var tools []tool
		switch runtime.GOOS {
		case "darwin":
			tools = []tool{{"pbcopy", nil}}
		case "windows":
			tools = []tool{{"clip", nil}}
		default: // linux + the BSDs: wayland first, then X11
			tools = []tool{
				{"wl-copy", nil},
				{"xclip", []string{"-selection", "clipboard"}},
				{"xsel", []string{"--clipboard", "--input"}},
			}
		}
		for _, t := range tools {
			if _, err := exec.LookPath(t.name); err == nil {
				t := t
				clipRun = func(text string) error {
					cmd := exec.Command(t.name, t.args...)
					cmd.Stdin = strings.NewReader(text)
					if err := cmd.Run(); err != nil {
						return fmt.Errorf("%s: %w", t.name, err)
					}
					return nil
				}
				return
			}
		}
	})
	if clipRun == nil {
		return errNoClipboardTool
	}
	return clipRun(text)
}

// errNoClipboardTool is the one degraded-platform verdict the terminal
// panel renders as a dim in-panel note (requirement: degrade, never fail
// silently).
var errNoClipboardTool = fmt.Errorf("no clipboard tool (pbcopy / wl-copy / xclip / xsel)")
