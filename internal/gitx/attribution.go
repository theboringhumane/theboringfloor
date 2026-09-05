package gitx

import (
	"regexp"
	"strings"
)

// Attribution constants for the office's bot profile. Every commit the
// theboringfloor code authors (today: none in-app — the gitx surface is
// read-only; tomorrow: whatever commit path wires this helper) must carry
// MajdoorTrailer so GitHub renders TheBoringMajdoor as a co-author.
const (
	// MajdoorName is the bot profile's display name.
	MajdoorName = "TheBoringMajdoor"
	// MajdoorEmail is the bot profile's attribution email. Presence
	// detection keys on this string (case-insensitively), not the name.
	MajdoorEmail = "themajdoor@theboring.name"
	// MajdoorTrailer is the exact git trailer line appended to commits.
	MajdoorTrailer = "Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>"
)

// trailerLine matches a git trailer line: a token of alphanumerics and
// hyphens, a colon, then a space (or end of line) before the value.
var trailerLine = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*:(\s|$)`)

// EnsureMajdoorTrailer returns msg guaranteed to carry MajdoorTrailer exactly
// once. It is idempotent: applying it to its own output is a no-op, and an
// existing majdoor trailer is detected case-insensitively on the email.
//
// Placement follows git's trailer rules:
//   - the trailer is separated from the body by a blank line;
//   - when the message already ends in a trailer block (a run of
//     "Key: value" lines), the trailer is appended inside that block with no
//     extra blank line;
//   - trailing blank lines in msg are collapsed before placement, and an
//     empty/whitespace-only message yields the trailer line alone (the
//     subject — empty — is preserved, nothing is invented).
func EnsureMajdoorTrailer(msg string) string {
	if hasMajdoorTrailer(msg) {
		return msg
	}
	lines := strings.Split(strings.TrimRight(msg, "\n"), "\n")
	// Drop trailing blank lines so placement math sees the real tail.
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return MajdoorTrailer
	}
	// Walk back over the trailing trailer block, if any.
	i := len(lines)
	for i > 0 && trailerLine.MatchString(lines[i-1]) {
		i--
	}
	body := strings.Join(lines, "\n")
	if i < len(lines) {
		return body + "\n" + MajdoorTrailer // inside the existing block
	}
	return body + "\n\n" + MajdoorTrailer
}

// hasMajdoorTrailer reports whether any trailer-shaped line already carries
// the majdoor email, matched case-insensitively.
func hasMajdoorTrailer(msg string) bool {
	email := strings.ToLower(MajdoorEmail)
	for _, ln := range strings.Split(msg, "\n") {
		if trailerLine.MatchString(ln) && strings.Contains(strings.ToLower(ln), email) {
			return true
		}
	}
	return false
}
