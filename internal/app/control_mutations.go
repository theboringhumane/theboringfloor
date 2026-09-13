package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

// applyControlMutations performs remote-control writes on the Bubble Tea
// goroutine, keeping the model's ordinary input, stop, and session seams as
// the sole owners of their respective state transitions.
func (m *Model) applyControlMutations(ev state.Event) tea.Cmd {
	switch ev.Kind {
	case state.EvControlWorkspace:
		return m.applyWorkspaceAction(ev)
	case state.EvControlSend:
		text := strings.TrimSpace(ev.ControlText)
		if text == "" {
			return nil
		}
		m.prepareRequestMode(text)
		return currentBackendSend(m.currentBackend, m.plan, text, ev.ControlAttachments)
	case state.EvControlStop:
		cmd := m.stopWork()
		m.notice("remote: stopped current work")
		return cmd
	case state.EvControlNew:
		m.newOffice()
		m.notice("remote: started a new session")
		return nil
	case state.EvControlPermissionAnswer:
		return m.applyControlPermissionAnswer(ev)
	case state.EvControlQuestionAnswer:
		return m.applyControlQuestionAnswer(ev)
	default:
		return nil
	}
}

// applyControlPermissionAnswer answers a permission prompt from a remote
// client through the SAME queue bookkeeping and backend seam the keyboard's
// permAnswerMsg case uses (model.go): the queue only advances when
// ev.ControlPermissionID matches the DISPLAYED front. A mismatch —
// including no prompt pending at all — fulfills the waiting HTTP request
// with an ErrorResponse instead of silently landing on the wrong prompt (or
// on nothing), so controlsrv maps it to 409 rather than a false 200.
func (m *Model) applyControlPermissionAnswer(ev state.Event) tea.Cmd {
	p := m.permQ.front()
	if p == nil || p.ID != ev.ControlPermissionID {
		fulfillControlAnswer(ev.ControlReqID, "no permission prompt pending with that id")
		return nil
	}
	pid, response := p.ID, ev.ControlPermissionResponse
	m.permQ.pending = m.permQ.pending[1:]
	m.permQ.escd = dropPrompt(m.permQ.escd, pid) // defensive: an ask lives in exactly one slice
	delete(m.permNotifyIDs, pid)                 // the cohort shrinks; empty re-arms the next ping
	m.chat.SetPermission(m.permQ.view())
	fulfillControlAnswer(ev.ControlReqID, "")
	// HOOKUP (browser_open.go): a parked browser-action hold resolves
	// LOCALLY — the office-minted pid never rides the backend's
	// AnswerPermission wire, exactly like the keyboard's permAnswerMsg case.
	if handled, cmd := m.consumeBrowserActionPerm(pid, response); handled {
		return cmd
	}
	current := m.currentBackend
	return func() tea.Msg {
		err := current.lease(func(b state.Backend) error {
			return b.AnswerPermission(pid, response)
		})
		if err != nil {
			return sendErrMsg{err: err}
		}
		return nil
	}
}

// applyControlQuestionAnswer answers the office's currently open boss
// question from a remote client through the SAME hold and backend seam the
// keyboard's questionAnswerMsg case uses. Unlike the keyboard's page-at-a-
// time wizard, the remote client submits the FULL accumulated answer set
// (or Reject) in one call, so the id check is against the hold's batched
// wire ids rather than its current wizard page.
func (m *Model) applyControlQuestionAnswer(ev state.Event) tea.Cmd {
	h := m.question
	if h == nil || !hasQuestionID(h.IDs, ev.ControlQuestionID) {
		fulfillControlAnswer(ev.ControlReqID, "no boss question pending with that id")
		return nil
	}
	if h.IDs[0] == bypassConfirmID {
		// the /bypass enable confirm is OFFICE-LOCAL (model.go's
		// questionAnswerMsg case): the answer never rides AnswerQuestion/
		// RejectQuestion. Anything but exactly one answer set of ["enable"]
		// is a no-op, same as esc/custom-text/cancel locally.
		m.question = nil
		m.chat.SetQuestion(nil)
		m.chat.SetPermission(m.permQ.view())
		fulfillControlAnswer(ev.ControlReqID, "")
		if !ev.ControlQuestionReject && len(ev.ControlQuestionAnswers) == 1 &&
			len(ev.ControlQuestionAnswers[0]) == 1 && ev.ControlQuestionAnswers[0][0] == "enable" {
			m.bypassDesired = true
			return m.respawnForBypass()
		}
		return nil
	}
	ids := append([]string(nil), h.IDs...)
	m.question = nil
	m.chat.SetQuestion(nil)
	// the question popover hides the permission popover — re-push the
	// queue front now the region is free again (mirrors the keyboard path).
	m.chat.SetPermission(m.permQ.view())
	fulfillControlAnswer(ev.ControlReqID, "")
	current := m.currentBackend
	if ev.ControlQuestionReject {
		return func() tea.Msg {
			err := current.lease(func(b state.Backend) error {
				for _, qid := range ids {
					if err := b.RejectQuestion(qid); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				return sendErrMsg{err: err}
			}
			return nil
		}
	}
	answers := ev.ControlQuestionAnswers
	return func() tea.Msg {
		err := current.lease(func(b state.Backend) error {
			for _, qid := range ids {
				if err := b.AnswerQuestion(qid, answers); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return sendErrMsg{err: err}
		}
		return nil
	}
}

// fulfillControlAnswer completes a permission/question answer's pending HTTP
// request through the SAME process-wide registry EvControlQuery reads use
// (see control.go): an empty message fulfills with OKResponse (the answer
// landed on the model), any other message fulfills with ErrorResponse so
// controlsrv.Server maps it to 409 instead of a false 200.
func fulfillControlAnswer(reqID, errMessage string) {
	var payload []byte
	if errMessage == "" {
		payload = marshalControlResponse(control.OKResponse{OK: true})
	} else {
		payload = marshalControlResponse(control.ErrorResponse{Error: errMessage})
	}
	controlReplies.Load().Fulfill(reqID, payload)
}
