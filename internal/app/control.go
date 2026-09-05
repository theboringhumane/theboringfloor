package app

import (
	"encoding/json"
	"fmt"
	"sync/atomic"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	controlTranscriptDefault = 50
	controlTranscriptMax     = 500
)

// controlReplies is deliberately process-wide: the HTTP server and Bubble Tea
// update loop need one small hand-off, while the model must remain the only
// owner allowed to read live office state.
var controlReplies atomic.Pointer[control.Registry]

func init() { controlReplies.Store(control.NewRegistry()) }

// SetControlRegistry joins the loopback server to the UI's reply path. A nil
// registry is ignored so an incomplete optional control setup cannot turn a
// received query into a panic.
func SetControlRegistry(registry *control.Registry) {
	if registry != nil {
		controlReplies.Store(registry)
	}
}

// applyControl projects live UI state for a loopback request. It runs on the
// Bubble Tea goroutine, rather than in an HTTP handler, so callers never race
// the model's ordinary event reductions.
func (m *Model) applyControl(ev state.Event) tea.Cmd {
	if ev.Kind != state.EvControlQuery {
		return nil
	}

	var payload []byte
	switch ev.ControlQuery {
	case control.QueryPlan:
		draft := ""
		if m.plan != nil {
			draft = m.plan.Value()
		}
		approved := m.approvedPlanText()
		payload = marshalControlResponse(control.PlanResponse{
			Draft: draft, Approved: approved, HasApproved: approved != "",
		})
	case control.QueryTranscript:
		payload = marshalControlResponse(m.controlTranscript(ev.ControlLimit))
	case control.QueryStatus:
		draft := ""
		if m.plan != nil {
			draft = m.plan.Value()
		}
		approved := m.approvedPlanText()
		payload = marshalControlResponse(control.StatusResponse{
			Dir:             m.memoryDir(),
			Backend:         m.backendName(),
			PrimaryID:       m.PrimarySessionID(),
			PlanDraftLen:    utf8.RuneCountInString(draft),
			PlanApprovedLen: utf8.RuneCountInString(approved),
			ChatCount:       len(m.st.Chat),
		})
	default:
		payload = marshalControlResponse(control.ErrorResponse{
			Error: fmt.Sprintf("unknown control query %q", ev.ControlQuery),
		})
	}

	// The registry channel is buffered; fulfilling cannot hold up the UI loop.
	// The registry may already have timed out and cancelled the request, which
	// still counts as this event's single, intentional fulfilment attempt.
	controlReplies.Load().Fulfill(ev.ControlReqID, payload)
	return nil
}

// controlTranscript converts only completed chat rows, then keeps their tail:
// control clients receive chronological messages without observing a partial
// incoming bubble as if it were a completed transcript record.
func (m *Model) controlTranscript(limit int) control.TranscriptResponse {
	if limit <= 0 {
		limit = controlTranscriptDefault
	}
	if limit > controlTranscriptMax {
		limit = controlTranscriptMax
	}
	messages := make([]control.TranscriptMessage, 0, len(m.st.Chat))
	for _, message := range m.st.Chat {
		if message.Pending {
			continue
		}
		messages = append(messages, control.TranscriptMessage{
			ID: message.ID, From: message.From, Kind: message.Kind,
			Text: message.Text, At: message.At,
		})
	}
	truncated := len(messages) > limit
	if truncated {
		messages = messages[len(messages)-limit:]
	}
	return control.TranscriptResponse{Messages: messages, Truncated: truncated}
}

func marshalControlResponse(response any) []byte {
	payload, err := json.Marshal(response)
	if err == nil {
		return payload
	}
	// Every current response is marshal-safe, but a response error must not
	// strand the HTTP request if a future field makes that assumption false.
	payload, _ = json.Marshal(control.ErrorResponse{Error: "control response marshal failed"})
	return payload
}
