package app

import (
	"regexp"
	"strings"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

// Submissions cross the event loop before transport I/O so mode, focus, and
// routing change together. No background command mutates the model.
type chatSubmitMsg struct {
	text string
	atts []state.Attachment
}

var requestAction = regexp.MustCompile(`\b(add|build|implement|create|migrate|redesign|refactor|replace|develop|integrate|introduce|overhaul)\b`)
var substantialScope = regexp.MustCompile(`\b((major|large|big) (feature|task|change|refactor|redesign)|(full|complete|entire) .{0,60}(system|app|dashboard|project|storage|database|website|product)|end-to-end|multi-tenant|multi-project|architecture|authentication|authorization|workspaces|migration)\b`)
var requestQuestion = regexp.MustCompile(`^(what|why|how|where|when|explain|describe|summarize|compare)\b`)

// This is deliberately a conservative fast path. The boss's semantic scope
// assessment covers substantial tasks this lexical check cannot identify.
func needsPlanning(text string) bool {
	s := strings.ToLower(strings.TrimSpace(text))
	if s == "" || strings.HasPrefix(s, "/") || strings.HasPrefix(text, approvePrefix) || requestQuestion.MatchString(s) {
		return false
	}
	actions := requestAction.FindAllString(s, -1)
	if len(actions) == 0 {
		return false
	}
	if substantialScope.MatchString(s) {
		return true
	}
	if strings.Contains(s, "frontend") && strings.Contains(s, "backend") {
		return true
	}
	return len(actions) >= 3 && len(strings.Fields(s)) >= 45
}

func (m *Model) prepareRequestMode(text string) {
	if strings.HasPrefix(strings.TrimSpace(text), "/") {
		return
	}
	if m.planAutoSkipOnce {
		m.planAutoSkipOnce = false
		return
	}
	if m.agentMode == agentModePlan || !needsPlanning(text) {
		return
	}
	if m.plan != nil {
		m.planRequestDraft = m.plan.Value()
	}
	m.revealPlan()
	m.notice("[office] substantial request — planning first · review the draft before build · ctrl+p returns to build")
}

// Plan tools must be visible even if the member was in zen, a worker thread,
// terminal capture, or expanded tools. Transient forms keep their draft and
// remain above the plan until dismissed.
func (m *Model) revealPlan() {
	m.approveArmAt = time.Time{}
	m.setAgentMode(agentModePlan)
	m.zen = false
	m.focusPanel = false
	m.floorNavFocused = false
	m.closeThreadFocus()
	m.setTermCaptured(false)
	m.tabs.SetActive(0)
	m.frameNonce++
}
