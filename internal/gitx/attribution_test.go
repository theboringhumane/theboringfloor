package gitx

import (
	"strings"
	"testing"
)

// TestEnsureMajdoorTrailer pins the trailer contract: exact line, idempotent
// placement, git-trailer blank-line rules, body preservation.
func TestEnsureMajdoorTrailer(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no trailer → added after a blank line",
			in:   "Add the git panel",
			want: "Add the git panel\n\n" + MajdoorTrailer,
		},
		{
			name: "already present → unchanged",
			in:   "Add the git panel\n\n" + MajdoorTrailer,
			want: "Add the git panel\n\n" + MajdoorTrailer,
		},
		{
			name: "already present, weird case on the email → unchanged",
			in:   "Add the git panel\n\nCo-authored-by: TheBoringMajdoor <THEMAJDOOR@THEBORING.NAME>",
			want: "Add the git panel\n\nCo-authored-by: TheBoringMajdoor <THEMAJDOOR@THEBORING.NAME>",
		},
		{
			name: "existing other trailer → appended inside the block, no blank line",
			in:   "Add the git panel\n\nSigned-off-by: Boss <boss@example.com>",
			want: "Add the git panel\n\nSigned-off-by: Boss <boss@example.com>\n" + MajdoorTrailer,
		},
		{
			name: "multiline body preserved verbatim",
			in:   "Add the git panel\n\nBody line one.\nBody line two with `code` and: colons.",
			want: "Add the git panel\n\nBody line one.\nBody line two with `code` and: colons.\n\n" + MajdoorTrailer,
		},
		{
			name: "empty message → trailer alone",
			in:   "",
			want: MajdoorTrailer,
		},
		{
			name: "whitespace-only message → trailer alone",
			in:   "   \n\n  \n",
			want: MajdoorTrailer,
		},
		{
			name: "trailing blank lines collapsed before placement",
			in:   "Add the git panel\n\n\n",
			want: "Add the git panel\n\n" + MajdoorTrailer,
		},
		{
			name: "trailer block directly under subject (no blank) stays a block",
			in:   "Add the git panel\nSigned-off-by: Boss <boss@example.com>",
			want: "Add the git panel\nSigned-off-by: Boss <boss@example.com>\n" + MajdoorTrailer,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EnsureMajdoorTrailer(tc.in)
			if got != tc.want {
				t.Fatalf("EnsureMajdoorTrailer(%q):\n got %q\nwant %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestEnsureMajdoorTrailerIdempotent pins the apply-twice rule: the second
// application must be a byte-exact no-op on every case shape.
func TestEnsureMajdoorTrailerIdempotent(t *testing.T) {
	inputs := []string{
		"Add the git panel",
		"Add the git panel\n\nbody\nmore body",
		"Add the git panel\n\nSigned-off-by: Boss <boss@example.com>",
		"",
		"  \n\n",
	}
	for _, in := range inputs {
		once := EnsureMajdoorTrailer(in)
		twice := EnsureMajdoorTrailer(once)
		if twice != once {
			t.Fatalf("not idempotent for %q:\nonce  %q\ntwice %q", in, once, twice)
		}
		if n := strings.Count(twice, MajdoorTrailer); n != 1 {
			t.Fatalf("trailer count for %q = %d, want exactly 1:\n%q", in, n, twice)
		}
	}
}

// TestMajdoorTrailerExact pins the byte-exact trailer string — GitHub's
// co-author parsing depends on this exact shape.
func TestMajdoorTrailerExact(t *testing.T) {
	const want = "Co-authored-by: TheBoringMajdoor <themajdoor@theboring.name>"
	if MajdoorTrailer != want {
		t.Fatalf("MajdoorTrailer = %q, want %q", MajdoorTrailer, want)
	}
}
