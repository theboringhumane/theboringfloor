// boot_ledger_test.go — the boot splash's memory row contract: it sits as
// the third agentmemory-fed cascade line (behind the board/mail pair), it
// renders "memory: ledger armed" while the lane probed live (and by
// default), and SetMemoryLane("file-only") flips it to the refused-port
// note — the office NEVER shows a blank memory row.
package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

// The splash's memory row, fully revealed: default (armed).
func TestBootMemoryLineArmedDefault(t *testing.T) {
	b := bootFeed(NewBoot(100, 30), 40) // full cold cascade lands at tick 39
	plain := ansi.Strip(b.View())
	if !strings.Contains(plain, "memory: ledger armed") {
		t.Fatalf("the revealed cascade must carry the armed memory row:\n%s", plain)
	}
	// Position contract: board → mail → memory → mcp.
	iBoard := strings.Index(plain, "board:")
	iMail := strings.Index(plain, "mail:")
	iMemory := strings.Index(plain, "memory:")
	iMCP := strings.Index(plain, "mcp:")
	if !(iBoard >= 0 && iBoard < iMail && iMail < iMemory && iMemory < iMCP) {
		t.Fatalf("memory row must sit behind the board/mail pair (third), got %d,%d,%d,%d", iBoard, iMail, iMemory, iMCP)
	}
}

// The verdict flips the row: SetMemoryLane("file-only") renders the
// refused-port note instead of the armed text (and the line-typed shape —
// stagger slot, OK flip — rides along untouched).
func TestBootMemoryLineFileOnly(t *testing.T) {
	b := NewBoot(100, 30)
	b.SetMemoryLane("file-only")
	// The refused-port row is the cascade's longest (45 cells): cold cadence
	// completes it at tick 47, UNDER the 4s cap (bootMaxTicks=50).
	b = bootFeed(b, 48)
	plain := ansi.Strip(b.View())
	if !strings.Contains(plain, bootMemoryFileOnlyLine) {
		t.Fatalf("file-only verdict must reveal the refused-port row:\n%s", plain)
	}
	if strings.Contains(plain, "memory: ledger armed") {
		t.Fatalf("the armed row must not linger after the file-only verdict:\n%s", plain)
	}
	// Six lines, six OKs — the dynamic-width memory row completes its OK.
	if n := strings.Count(plain, "OK"); n != 6 {
		t.Fatalf("the file-only cascade still completes all 6 rows, got %d", n)
	}
	if !b.cascadeDone() {
		t.Fatalf("cascadeDone must hold for the (longer) file-only row")
	}
}

// Mid-boot verdict: a file-only landing before the row's stagger slot
// never reveals an armed residue.
func TestBootMemoryLineVerdictBeforeReveal(t *testing.T) {
	b := NewBoot(100, 30)
	b = bootFeed(b, 24) // memory's cold stagger slot opens at tick 24
	b.SetMemoryLane("file-only")
	b = bootFeed(b, 12)
	plain := ansi.Strip(b.View())
	if strings.Contains(plain, "ledger armed") {
		t.Fatalf("a pre-reveal verdict must never type the armed text:\n%s", plain)
	}
	if !strings.Contains(plain, "memory: file-only") {
		t.Fatalf("the verdict's row must be typing (or done), got:\n%s", plain)
	}
}
