package backend

import (
	"strings"
	"testing"
	"time"

	"github.com/theboringhumane/theboringoffice/internal/state"
)

func TestDemoIdleCheckinRecapsWithoutHiring(t *testing.T) {
	b := newDemoBackend(nil)
	log := &eventLog{}
	if err := b.Start(log.emit); err != nil {
		t.Fatal(err)
	}
	defer b.Stop()
	prompt := "[theboringfloor] check-in: idle 2m, no workers running, last chat wasn't from the boss or office. Recap the shift for the member in a few lines (what finished, what's blocked, or that nothing landed). Do not start new work."
	if err := b.SendWith(prompt, nil); err != nil {
		t.Fatal(err)
	}
	log.waitFor(t, 2*time.Second, func() bool {
		return len(eventsMatching(log, func(e state.Event) bool {
			return e.Kind == state.EvChatBoss && !e.Msg.Pending &&
				strings.Contains(e.Msg.Text, "Shift recap:")
		})) == 1
	}, "demo boss pins an idle recap")
	time.Sleep(400 * time.Millisecond)
	hired := eventsMatching(log, func(e state.Event) bool {
		return e.Kind == state.EvDispatch && strings.HasPrefix(e.Task.ID, "adhoc-")
	})
	if len(hired) != 0 {
		t.Fatalf("idle check-in must not hire, got %+v", hired)
	}
}
