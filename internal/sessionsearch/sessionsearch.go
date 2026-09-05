// Package sessionsearch reads persisted office transcripts for the current
// project from ~/.theboringfloor/projects/<dirhash>/session.json, then the
// earlier same-product sessions/<dirhash>/session.json layout when the
// canonical file is absent. Writes remain exclusively in projects/; this
// package is READ-ONLY by design and never creates, modifies, or indexes
// session files. On-disk transcripts are tail-capped at 200 messages, so search
// results cover the recent tail rather than full history.
package sessionsearch

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/theboringhumane/theboringfloor/internal/brand"
	"github.com/theboringhumane/theboringfloor/internal/config"
	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	defaultTranscriptLimit = 50
	maxTranscriptLimit     = 500
	defaultSearchLimit     = 20
	maxSearchLimit         = 200
	maxSnippetRunes        = 240
)

// Session is the read-only subset of an on-disk office session snapshot.
type Session struct {
	Dir              string            `json:"dir"`
	PrimaryID        string            `json:"primaryID"`
	Backend          string            `json:"backend,omitempty"`
	PrimaryIDs       map[string]string `json:"primaryIDs,omitempty"`
	Agents           []state.Employee  `json:"agents"`
	Tasks            []state.BoardTask `json:"tasks"`
	Mails            []state.MailItem  `json:"mails"`
	Chat             []state.ChatMsg   `json:"chat"`
	PlanText         string            `json:"planText,omitempty"`
	ApprovedPlanText string            `json:"approvedPlanText,omitempty"`
	SavedAt          int64             `json:"savedAt"`
}

// Message is a persisted transcript message.
type Message = state.ChatMsg

// Hit is one search match and a rune-safe excerpt around its first match.
type Hit struct {
	Message Message
	Snippet string
}

// Meta is enough persisted information to answer an offline status request.
type Meta struct {
	Dir       string
	Backend   string
	PrimaryID string
	ChatCount int
	SavedAt   int64
}

// Load reads session.json for dir from projects first, then the earlier
// sessions layout if the canonical file is absent. Missing, unreadable, and
// malformed snapshots all return ok=false. A malformed canonical file does not
// fall through to older data, so stale content cannot be silently resurrected.
// Only session.json is opened; temporary write files are deliberately ignored.
func Load(dir string) (*Session, bool) {
	home := config.Env("HOME")
	if home == "" {
		home = os.Getenv("HOME")
	}
	paths := []string{
		filepath.Join(home, brand.DotDir, "projects", dirHash(dir), "session.json"),
		filepath.Join(earlierSessionsBase(home), dirHash(dir), "session.json"),
	}
	for i, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if i == 0 && os.IsNotExist(err) {
				continue
			}
			return nil, false
		}
		var session Session
		if err := json.Unmarshal(data, &session); err != nil || session.Dir == "" {
			return nil, false
		}
		return &session, true
	}
	return nil, false
}

// earlierSessionsBase is the prior same-product root. It is read-only
// compatibility; session writers continue to use projects.
func earlierSessionsBase(home string) string {
	return filepath.Join(home, brand.DotDir, "sessions")
}

// Transcript returns completed transcript messages in chronological order.
func Transcript(dir string, limit int) ([]Message, bool) {
	session, ok := Load(dir)
	if !ok {
		return nil, false
	}
	limit = boundedLimit(limit, defaultTranscriptLimit, maxTranscriptLimit)
	messages := make([]Message, 0, len(session.Chat))
	for _, message := range session.Chat {
		if !message.Pending {
			messages = append(messages, message)
		}
	}
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}
	return messages, true
}

// Search finds case-insensitive text or sender matches in the current
// project's completed transcript. Results rank by text occurrence count, then
// newest timestamp first.
func Search(dir string, query string, limit int) ([]Hit, bool) {
	session, ok := Load(dir)
	if !ok {
		return nil, false
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return []Hit{}, true
	}
	limit = boundedLimit(limit, defaultSearchLimit, maxSearchLimit)
	lowerQuery := strings.ToLower(query)
	type rankedHit struct {
		hit   Hit
		score int
		order int
	}
	results := make([]rankedHit, 0)
	for order, message := range session.Chat {
		if message.Pending {
			continue
		}
		lowerText := strings.ToLower(message.Text)
		lowerFrom := strings.ToLower(message.From)
		textMatch := strings.Index(lowerText, lowerQuery)
		if textMatch < 0 && !strings.Contains(lowerFrom, lowerQuery) {
			continue
		}
		snippetSource := message.Text
		snippetMatch := textMatch
		if snippetMatch < 0 {
			snippetSource = message.From
			snippetMatch = strings.Index(lowerFrom, lowerQuery)
		}
		results = append(results, rankedHit{
			hit:   Hit{Message: message, Snippet: snippet(snippetSource, snippetMatch, lowerQuery)},
			score: strings.Count(lowerText, lowerQuery),
			order: order,
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		if results[i].hit.Message.At != results[j].hit.Message.At {
			return results[i].hit.Message.At > results[j].hit.Message.At
		}
		return results[i].order < results[j].order
	})
	if len(results) > limit {
		results = results[:limit]
	}
	hits := make([]Hit, len(results))
	for i, result := range results {
		hits[i] = result.hit
	}
	return hits, true
}

// ApprovedPlan returns the persisted member-approved plan text.
func ApprovedPlan(dir string) (string, bool) {
	session, ok := Load(dir)
	if !ok {
		return "", false
	}
	return session.ApprovedPlanText, true
}

// Draft returns the persisted unapproved plan text.
func Draft(dir string) (string, bool) {
	session, ok := Load(dir)
	if !ok {
		return "", false
	}
	return session.PlanText, true
}

// Info returns persisted metadata for the current project.
func Info(dir string) (Meta, bool) {
	session, ok := Load(dir)
	if !ok {
		return Meta{}, false
	}
	return Meta{
		Dir:       session.Dir,
		Backend:   session.Backend,
		PrimaryID: session.PrimaryID,
		ChatCount: len(session.Chat),
		SavedAt:   session.SavedAt,
	}, true
}

func dirHash(dir string) string {
	canonical, err := filepath.Abs(dir)
	if err != nil {
		canonical = dir
	}
	if resolved, err := filepath.EvalSymlinks(canonical); err == nil {
		canonical = resolved
	}
	sum := sha1.Sum([]byte(canonical))
	return hex.EncodeToString(sum[:])
}

func boundedLimit(limit, defaultLimit, maximum int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maximum {
		return maximum
	}
	return limit
}

func snippet(text string, matchByte int, lowerQuery string) string {
	runes := []rune(text)
	if len(runes) <= maxSnippetRunes {
		return text
	}
	matchStart := utf8.RuneCountInString(strings.ToLower(text)[:matchByte])
	matchLength := utf8.RuneCountInString(lowerQuery)
	contentLimit := maxSnippetRunes - 2 // reserve room for both ellipses
	start := matchStart
	if matchLength < contentLimit {
		start -= (contentLimit - matchLength) / 2
	}
	if start < 0 {
		start = 0
	}
	end := start + contentLimit
	if end > len(runes) {
		end = len(runes)
		start = end - contentLimit
	}
	if start < 0 {
		start = 0
	}
	result := string(runes[start:end])
	if start > 0 {
		result = "…" + result
	}
	if end < len(runes) {
		result += "…"
	}
	return result
}
