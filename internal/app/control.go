package app

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	controlTranscriptDefault = 50
	// controlTranscriptMax bounds a single UI-goroutine transcript projection.
	// Larger requests are clamped rather than rejected so clients can recover.
	controlTranscriptMax = 500
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
	case control.QueryTranscript, control.QueryTranscript + "?page=1":
		response, _ := m.controlTranscript(ev.ControlLimit, "", ev.ControlQuery != control.QueryTranscript)
		payload = marshalControlResponse(response)
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
	case control.QueryBusy:
		pendingBoss := hasPendingBoss(m.st)
		payload = marshalControlResponse(control.BusyResponse{
			Busy:           m.controlWorking(),
			PendingBoss:    pendingBoss,
			Thinking:       m.st.BossThinking,
			Delegating:     m.st.BossDelegating,
			QuestionParked: m.questionParked,
		})
	default:
		if strings.HasPrefix(ev.ControlQuery, control.QueryTranscript+"?page=1&before=") {
			before, err := url.QueryUnescape(strings.TrimPrefix(ev.ControlQuery, control.QueryTranscript+"?page=1&before="))
			if err != nil || before == "" {
				payload = marshalControlResponse(control.ErrorResponse{Error: "invalid before cursor"})
			} else {
				response, found := m.controlTranscript(ev.ControlLimit, before, true)
				if !found {
					payload = marshalControlResponse(control.ErrorResponse{Error: "unknown before cursor"})
				} else {
					payload = marshalControlResponse(response)
				}
			}
			break
		}
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
// incoming bubble as if it were a completed transcript record. Its limit is a
// count of user turns; assistant activity following those turns rides along.
type controlTranscriptResponse struct {
	control.TranscriptResponse
	HasMore *bool `json:"hasMore,omitempty"`
}

func (m *Model) controlTranscript(limit int, before string, paged bool) (controlTranscriptResponse, bool) {
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
			Text: message.Text, At: message.At, Attachments: controlTranscriptAttachments(message.Meta),
		})
	}
	end := len(messages)
	if before != "" {
		found := false
		for i, message := range messages {
			if message.ID == before {
				end = i
				found = true
				break
			}
		}
		if !found {
			return controlTranscriptResponse{}, false
		}
	}
	// Count user turns while walking back. Starting at the user that fills the
	// page means activity after that turn remains with it, while older activity
	// remains for the preceding page instead of becoming an orphaned prefix.
	start := 0
	users := 0
	for i := end - 1; i >= 0; i-- {
		if messages[i].From != "user" {
			continue
		}
		users++
		if users == limit {
			start = i
			break
		}
	}
	page := messages[start:end]
	truncated := start > 0
	response := controlTranscriptResponse{TranscriptResponse: control.TranscriptResponse{
		Messages:  page,
		Truncated: truncated,
		Working:   m.controlWorking(),
	}}
	if paged {
		response.HasMore = &truncated
	}
	return response, true
}

// controlWorking is the same busy predicate exposed by the control busy
// projection and rendered by the TUI: a primary turn is pending, thinking,
// delegating, or parked awaiting completion.
func (m *Model) controlWorking() bool {
	return hasPendingBoss(m.st) || m.st.BossThinking || m.st.BossDelegating || m.questionParked
}

// controlTranscriptAttachments projects the compact attachment metadata that
// chat rows persist. AttachMeta carries outbound filenames only, while
// MediaMeta carries inbound image filenames and MIME types; neither carrier
// contains filesystem paths or image payload bytes.
func controlTranscriptAttachments(meta string) []control.TranscriptAttachment {
	var attachments []control.TranscriptAttachment
	if names, ok := state.ParseAttachMeta(meta); ok {
		for _, name := range names {
			attachments = append(attachments, control.TranscriptAttachment{Name: name})
		}
	}
	if items, ok := state.ParseMediaMeta(meta); ok {
		for _, item := range items {
			attachments = append(attachments, control.TranscriptAttachment{
				Name: item.Filename,
				Mime: item.Mime,
			})
		}
	}
	return attachments
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
