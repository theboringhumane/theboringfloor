package gitx

import (
	"slices"
	"testing"
)

// envOf adapts a map to the getenv signature the helpers take, so tests
// never touch the process env.
func envOf(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// TestMajdoorAuthorEnvActive pins the flag grammar: only exactly "true"
// (any case) activates the injection; unset/empty/everything else is off.
func TestMajdoorAuthorEnvActive(t *testing.T) {
	if AutoCommitFlag != "THEFLOOR_AUTO_COMMIT" {
		t.Fatalf("AutoCommitFlag = %q", AutoCommitFlag)
	}
	cases := []struct {
		name string
		val  *string // nil = unset
		want bool
	}{
		{name: "unset", val: nil, want: false},
		{name: "empty", val: ptr(""), want: false},
		{name: "false", val: ptr("false"), want: false},
		{name: "1", val: ptr("1"), want: false},
		{name: "true", val: ptr("true"), want: true},
		{name: "TRUE", val: ptr("TRUE"), want: true},
		{name: "True", val: ptr("True"), want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			env := map[string]string{}
			if c.val != nil {
				env[AutoCommitFlag] = *c.val
			}
			if got := MajdoorAuthorEnvActive(envOf(env)); got != c.want {
				t.Fatalf("MajdoorAuthorEnvActive(%v) = %v, want %v", env, got, c.want)
			}
		})
	}
}

func ptr(s string) *string { return &s }

// TestMajdoorAuthorEnv pins the four injected vars BYTE-EXACT (literals,
// not const-rebuilds — a wrong const value fails here).
func TestMajdoorAuthorEnv(t *testing.T) {
	want := []string{
		"GIT_AUTHOR_NAME=TheBoringMajdoor",
		"GIT_AUTHOR_EMAIL=themajdoor@theboring.name",
		"GIT_COMMITTER_NAME=TheBoringMajdoor",
		"GIT_COMMITTER_EMAIL=themajdoor@theboring.name",
	}
	got := MajdoorAuthorEnv()
	if !slices.Equal(got, want) {
		t.Fatalf("MajdoorAuthorEnv() = %q, want %q", got, want)
	}
}

// TestMajdoorEnvMerge pins the child-env contract: off → the slice passes
// through untouched (a parent GIT_AUTHOR_NAME survives); on → the four
// majdoor vars WIN over pre-existing parent values (deduped to exactly one
// entry per key) while unrelated vars keep their order.
func TestMajdoorEnvMerge(t *testing.T) {
	parent := []string{
		"PATH=/usr/bin",
		"GIT_AUTHOR_NAME=Boss Person",
		"GIT_COMMITTER_EMAIL=boss@example.com",
		"HOME=/Users/boss",
	}

	t.Run("flag off → env untouched", func(t *testing.T) {
		got := MajdoorEnvMerge(parent, envOf(map[string]string{}))
		if !slices.Equal(got, parent) {
			t.Fatalf("merge with flag off = %q, want the parent env unchanged %q", got, parent)
		}
	})

	t.Run("flag on → majdoor wins, deduped, order kept", func(t *testing.T) {
		got := MajdoorEnvMerge(parent, envOf(map[string]string{AutoCommitFlag: "true"}))
		want := []string{
			"PATH=/usr/bin",
			"HOME=/Users/boss",
			"GIT_AUTHOR_NAME=TheBoringMajdoor",
			"GIT_AUTHOR_EMAIL=themajdoor@theboring.name",
			"GIT_COMMITTER_NAME=TheBoringMajdoor",
			"GIT_COMMITTER_EMAIL=themajdoor@theboring.name",
		}
		if !slices.Equal(got, want) {
			t.Fatalf("merge with flag on = %q, want %q", got, want)
		}
		for _, key := range []string{gitAuthorNameKey, gitAuthorEmailKey, gitCommitterNameKey, gitCommitterEmailKey} {
			n := 0
			for _, kv := range got {
				if len(kv) > len(key) && kv[:len(key)+1] == key+"=" {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("child env carries %d %s entries, want exactly 1: %q", n, key, got)
			}
		}
	})
}
