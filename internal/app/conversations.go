package app

import (
	"crypto/sha256"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/control"
	"github.com/theboringhumane/theboringfloor/internal/state"
	"github.com/theboringhumane/theboringfloor/internal/workspace"
)

// One writer per model serializes autosave and quit. Sequence numbers prevent
// a delayed autosave from overwriting a newer final save or conversation switch.
type sessionWriter struct {
	mu     sync.Mutex
	latest uint64
	next   atomic.Uint64
	digest [32]byte
	err    error
}

// Reserving an autosave never waits on the disk writer's mutex.
func (w *sessionWriter) reserve() uint64 { return w.next.Add(1) }
func (w *sessionWriter) save(seq uint64, dir string, sf SessionFile, full []state.ChatMsg, title, team string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if seq < w.latest {
		return
	}
	w.latest = seq
	payload, _ := json.Marshal(struct {
		Session     SessionFile
		Chat        []state.ChatMsg
		Title, Team string
	}{sf, full, title, team})
	digest := sha256.Sum256(payload)
	if digest == w.digest && w.err == nil {
		return
	}
	if sf.PrimaryID != "" {
		archive := sf
		archive.Chat = full
		archive.SavedAt = time.Now().UnixMilli()
		base := workspace.ConversationDir(dir, sf.Backend, sf.PrimaryID)
		// Preserve archived history when restoring from an older, capped snapshot.
		if old, ok := loadConversation(dir, sf.Backend, sf.PrimaryID); ok {
			archive.Chat = mergeArchiveChat(old.Chat, full)
		}
		if title == "" {
			for _, c := range archive.Chat {
				if c.From == "user" && strings.TrimSpace(c.Text) != "" {
					title = strings.Join(strings.Fields(c.Text), " ")
					break
				}
			}
		}
		if title == "" {
			title = "Untitled conversation"
		}
		r := []rune(title)
		if len(r) > 80 {
			title = string(r[:80]) + "…"
		}
		archive.Title, archive.Team = title, team
		meta := workspace.Conversation{ID: sf.PrimaryID, Backend: sf.Backend, Title: title, Team: team, Updated: archive.SavedAt, Messages: len(archive.Chat)}
		w.err = workspace.WriteJSON(filepath.Join(base, "session.json"), archive)
		if w.err == nil {
			w.err = workspace.WriteJSON(filepath.Join(base, "meta.json"), meta)
		}
		if w.err != nil {
			return
		}
	}
	w.err = SaveSession(dir, sf)
	if w.err == nil {
		w.digest = digest
	}
}
func mergeArchiveChat(old, next []state.ChatMsg) []state.ChatMsg {
	out := append([]state.ChatMsg(nil), old...)
	ids := map[string]int{}
	for i, c := range out {
		if c.ID != "" {
			ids[c.ID] = i
		}
	}
	for _, c := range next {
		if c.Meta == bootNoticeMeta || c.Meta == bootWarnNoticeMeta || c.Pending {
			continue
		}
		if i, ok := ids[c.ID]; ok && c.ID != "" {
			out[i] = c
		} else {
			if c.ID != "" {
				ids[c.ID] = len(out)
			}
			out = append(out, c)
		}
	}
	// Match the live transcript fuse; archives are per conversation, never
	// overwritten by a new conversation or another backend's session id.
	if len(out) > chatCap {
		out = out[len(out)-chatCap:]
	}
	return out
}
func loadConversation(dir, backend, id string) (*SessionFile, bool) {
	raw, err := os.ReadFile(filepath.Join(workspace.ConversationDir(dir, backend, id), "session.json"))
	if err != nil {
		return nil, false
	}
	var sf SessionFile
	if json.Unmarshal(raw, &sf) != nil || sf.PrimaryID != id || sf.Backend != backend || SessionDirHash(sf.Dir) != SessionDirHash(dir) {
		return nil, false
	}
	return &sf, true
}

// ArchiveTranscript projects saved rows with the same attachment metadata as
// live conversations. The phone decides which rows belong in its reading view.
func ArchiveTranscript(sf SessionFile) []control.TranscriptMessage {
	rows := make([]control.TranscriptMessage, 0, len(sf.Chat))
	for _, m := range sf.Chat {
		if m.Pending {
			continue
		}
		rows = append(rows, control.TranscriptMessage{ID: m.ID, From: m.From, Kind: m.Kind, Text: m.Text, At: m.At, Attachments: controlTranscriptAttachments(m.Meta)})
	}
	return rows
}
