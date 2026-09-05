package mcpinstall

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func setup(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("HOME", root)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "xdg"))
	t.Setenv("PATH", "")
	bin := filepath.Join(root, "thefloor_mcp")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, bin
}

func configPath(t *testing.T) string {
	t.Helper()
	p, err := openCodeConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestEnsureFreshInstallPreservesUnrelatedValuesAndIsIdempotent(t *testing.T) {
	root, bin := setup(t)
	path := configPath(t)
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		t.Fatalf("path escaped temp root: %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	before := "{\n  \"theme\": \"dark\",\n  \"nested\": {\"keep\": [1, true, \"x\"]}\n}\n"
	if err := os.WriteFile(path, []byte(before), 0o640); err != nil {
		t.Fatal(err)
	}
	result, err := Ensure(bin)
	if err != nil {
		t.Fatal(err)
	}
	if result.OpenCode.Status != "installed" || result.Claude.Status != "skipped" {
		t.Fatalf("unexpected result: %+v", result)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc["theme"] != "dark" || !reflect.DeepEqual(doc["nested"], map[string]any{"keep": []any{float64(1), true, "x"}}) {
		t.Fatalf("unrelated values changed: %#v", doc)
	}
	entry := doc["mcp"].(map[string]any)[serverName].(map[string]any)
	if entry["command"].([]any)[0] != bin {
		t.Fatalf("relative or wrong command: %#v", entry)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(raw)
	time.Sleep(20 * time.Millisecond)
	second, err := Ensure(bin)
	if err != nil {
		t.Fatal(err)
	}
	if second.OpenCode.Status != "already-present" {
		t.Fatalf("second result: %+v", second)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	infoAfter, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if hash != sha256.Sum256(after) || !info.ModTime().Equal(infoAfter.ModTime()) {
		t.Fatalf("second Ensure wrote config")
	}
}

func TestEnsureFreshMissingConfig(t *testing.T) {
	root, bin := setup(t)
	result, _ := Ensure(bin)
	path := configPath(t)
	if !strings.HasPrefix(path, root+string(os.PathSeparator)) {
		t.Fatalf("path escaped temp root: %s", path)
	}
	if result.OpenCode.Status != "installed" {
		t.Fatalf("%+v", result.OpenCode)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestEnsureMalformedAndWrongTypeSkip(t *testing.T) {
	_, bin := setup(t)
	path := configPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"{ nope", `{"mcp": []}`} {
		if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
		got, _ := Ensure(bin)
		if got.OpenCode.Status != "skipped" {
			t.Fatalf("%q: %+v", raw, got.OpenCode)
		}
		out, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(out) != raw {
			t.Fatalf("unsafe rewrite: got %q want %q", out, raw)
		}
	}
}

func TestEnsurePreservesNumberLiterals(t *testing.T) {
	_, bin := setup(t)
	path := configPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	before := `{
  "large": 9007199254740993,
  "precise_float": 0.12345678901234567890123456789,
  "negative": -987654321.123456789,
  "exponent": 1.234567890123456789e+123
}
`
	if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _ := Ensure(bin)
	if got.OpenCode.Status != "installed" {
		t.Fatalf("%+v", got.OpenCode)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, literal := range []string{
		`"large": 9007199254740993`,
		`"precise_float": 0.12345678901234567890123456789`,
		`"negative": -987654321.123456789`,
		`"exponent": 1.234567890123456789e+123`,
	} {
		if !strings.Contains(string(after), literal) {
			t.Fatalf("number literal changed or missing %q in:\n%s", literal, after)
		}
	}
}

func TestEnsurePreservesSymlinkedConfig(t *testing.T) {
	root, bin := setup(t)
	path := configPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "dotfiles-opencode.json")
	if err := os.WriteFile(target, []byte(`{"theme":"dark"}`), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	got, _ := Ensure(bin)
	if got.OpenCode.Status != "installed" {
		t.Fatalf("%+v", got.OpenCode)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config symlink was replaced: mode=%v", info.Mode())
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"thefloor_mcp"`) {
		t.Fatalf("resolved config was not updated:\n%s", raw)
	}
}

func TestEnsureDanglingSymlinkSkips(t *testing.T) {
	root, bin := setup(t)
	path := configPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "missing-dotfiles-opencode.json")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	got, _ := Ensure(bin)
	if got.OpenCode.Status != "skipped" || !strings.Contains(got.OpenCode.Reason, "dangling symlink") {
		t.Fatalf("%+v", got.OpenCode)
	}
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("dangling config symlink was replaced: mode=%v", info.Mode())
	}
	if _, err := os.Lstat(target); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dangling target was unexpectedly created: %v", err)
	}
}

func TestEnsureOptOutAndMissingBinary(t *testing.T) {
	root, bin := setup(t)
	t.Setenv("THEBORINGOFFICE_NO_MCP_INSTALL", "1")
	got, _ := Ensure(bin)
	if got.OpenCode.Status != "skipped" || got.Claude.Status != "skipped" {
		t.Fatalf("%+v", got)
	}
	if _, err := os.Stat(configPath(t)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("opt-out wrote config: %v", err)
	}
	t.Setenv("THEBORINGOFFICE_NO_MCP_INSTALL", "")
	got, _ = Ensure(filepath.Join(root, "missing"))
	if !strings.Contains(got.OpenCode.Reason, "binary not found") || got.Claude.Status != "skipped" {
		t.Fatalf("%+v", got)
	}
}

func TestEnsureClaudeCLIArgv(t *testing.T) {
	root, bin := setup(t)
	claude := filepath.Join(root, "claude")
	if err := os.WriteFile(claude, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root)
	old := commandRunner
	t.Cleanup(func() { commandRunner = old })
	var calls [][]string
	commandRunner = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		if reflect.DeepEqual(args, []string{"mcp", "get", serverName}) {
			return errors.New("not registered")
		}
		return nil
	}
	got, _ := Ensure(bin)
	if got.Claude.Status != "installed" {
		t.Fatalf("%+v", got.Claude)
	}
	want := [][]string{{claude, "mcp", "get", serverName}, {claude, "mcp", "add", "--scope", "user", serverName, "--", bin}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("argv got %#v want %#v", calls, want)
	}
}
