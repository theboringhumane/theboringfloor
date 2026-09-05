// Package mcpinstall safely registers thefloor_mcp in global OpenCode and
// Claude Code MCP settings. It is JSON-only (never rewrites JSONC),
// additive-only, idempotent, and uses atomic writes for OpenCode. Ensure is
// never fatal so it cannot block office boot; THEFLOOR_NO_MCP_INSTALL=1
// opts out. Claude configuration deliberately goes through Claude's supported
// CLI rather than merging ~/.claude.json, which is shared user state.
package mcpinstall

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/theboringhumane/theboringfloor/internal/config"
)

const (
	serverName = "thefloor_mcp"
	timeout    = 5 * time.Second
)

// HostResult records the non-fatal result for one host. Status is one of
// "installed", "already-present", "skipped", or "failed".
type HostResult struct {
	Status string
	Reason string
}

// Result records the independent outcomes for the two global host configs.
type Result struct {
	OpenCode HostResult
	Claude   HostResult
}

// commandRunner is replaceable by tests. It intentionally returns only an
// error because Ensure only needs to distinguish a successful `mcp get` from
// an absent/unavailable server.
var commandRunner = func(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

// Ensure registers binPath with both hosts. It reports failures in Result and
// always returns a nil error so startup remains best-effort and non-blocking.
func Ensure(binPath string) (Result, error) {
	if config.EnvBool("NO_MCP_INSTALL") {
		return Result{
			OpenCode: HostResult{Status: "skipped", Reason: "disabled by THEFLOOR_NO_MCP_INSTALL"},
			Claude:   HostResult{Status: "skipped", Reason: "disabled by THEFLOOR_NO_MCP_INSTALL"},
		}, nil
	}

	path, ok := usableBinary(binPath)
	if !ok {
		return Result{
			OpenCode: HostResult{Status: "skipped", Reason: "thefloor_mcp binary not found or not executable"},
			Claude:   HostResult{Status: "skipped", Reason: "thefloor_mcp binary not found or not executable"},
		}, nil
	}

	return Result{OpenCode: ensureOpenCode(path), Claude: ensureClaude(path)}, nil
}

// ResolveBinary returns the absolute path to thefloor_mcp, preferring a
// sibling of the current executable before consulting PATH.
func ResolveBinary() (string, bool) {
	if current, err := os.Executable(); err == nil {
		if path, ok := usableBinary(filepath.Join(filepath.Dir(current), serverName)); ok {
			return path, true
		}
	}
	if path, err := exec.LookPath(serverName); err == nil {
		return usableBinary(path)
	}
	return "", false
}

func usableBinary(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return "", false
	}
	return abs, true
}

// ensureOpenCode resolves a symlinked config before atomically replacing its
// target, preserving the member's symlink rather than renaming over it.
func ensureOpenCode(binPath string) HostResult {
	path, err := openCodeConfigPath()
	if err != nil {
		return HostResult{Status: "failed", Reason: err.Error()}
	}

	linkInfo, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return createOpenCodeConfig(path, binPath)
	}
	if err != nil {
		return HostResult{Status: "failed", Reason: "lstat " + path + ": " + err.Error()}
	}

	writePath := path
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		// Resolve symlinks before writing so atomic rename updates the linked file,
		// rather than replacing a member's symlinked configuration.
		writePath, err = filepath.EvalSymlinks(path)
		if err != nil {
			return HostResult{Status: "skipped", Reason: "refusing to rewrite dangling symlink " + path + ": " + err.Error()}
		}
	}

	raw, err := os.ReadFile(writePath)
	if err != nil {
		return HostResult{Status: "failed", Reason: "read " + path + ": " + err.Error()}
	}

	merged, changed, err := mergeOpenCode(raw, binPath)
	if err != nil {
		return HostResult{Status: "skipped", Reason: "refusing to rewrite " + path + ": " + err.Error()}
	}
	if !changed {
		return HostResult{Status: "already-present", Reason: "thefloor_mcp is already registered"}
	}
	info, err := os.Stat(writePath)
	if err != nil {
		return HostResult{Status: "failed", Reason: "stat " + path + ": " + err.Error()}
	}
	if err := atomicWrite(writePath, merged, info.Mode().Perm()); err != nil {
		return HostResult{Status: "failed", Reason: "write " + path + ": " + err.Error()}
	}
	return HostResult{Status: "installed", Reason: "registered in " + path}
}

func createOpenCodeConfig(path, binPath string) HostResult {
	doc := map[string]any{"mcp": map[string]any{serverName: openCodeEntry(binPath)}}
	raw, err := marshalConfig(doc)
	if err != nil {
		return HostResult{Status: "failed", Reason: "marshal OpenCode config: " + err.Error()}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return HostResult{Status: "failed", Reason: "mkdir " + filepath.Dir(path) + ": " + err.Error()}
	}
	if err := atomicWrite(path, raw, 0o644); err != nil {
		return HostResult{Status: "failed", Reason: "write " + path + ": " + err.Error()}
	}
	reason := "registered in newly created " + path
	if _, err := os.Stat(filepath.Join(filepath.Dir(path), "opencode.jsonc")); err == nil {
		reason += " (left existing opencode.jsonc untouched)"
	}
	return HostResult{Status: "installed", Reason: reason}
}

func mergeOpenCode(raw []byte, binPath string) ([]byte, bool, error) {
	var doc map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&doc); err != nil {
		return nil, false, fmt.Errorf("invalid JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return nil, false, errors.New("invalid JSON: multiple top-level values")
		}
		return nil, false, fmt.Errorf("invalid JSON: %w", err)
	}
	if doc == nil {
		return nil, false, errors.New("top-level config is null")
	}
	mcp, exists := doc["mcp"]
	if !exists {
		doc["mcp"] = map[string]any{serverName: openCodeEntry(binPath)}
	} else {
		servers, ok := mcp.(map[string]any)
		if !ok || servers == nil {
			return nil, false, errors.New("mcp is not an object")
		}
		if _, present := servers[serverName]; present {
			return raw, false, nil
		}
		servers[serverName] = openCodeEntry(binPath)
	}
	out, err := marshalConfig(doc)
	return out, true, err
}

func openCodeEntry(binPath string) map[string]any {
	return map[string]any{"type": "local", "command": []string{binPath}, "enabled": true}
}

func marshalConfig(doc map[string]any) ([]byte, error) {
	raw, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(raw, '\n'), nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".thefloor-mcp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func ensureClaude(binPath string) HostResult {
	claude, err := exec.LookPath("claude")
	if err != nil {
		return HostResult{Status: "skipped", Reason: "claude executable not found on PATH; ~/.claude.json was not modified"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	err = commandRunner(ctx, claude, "mcp", "get", serverName)
	cancel()
	if err == nil {
		return HostResult{Status: "already-present", Reason: "thefloor_mcp is already registered"}
	}

	ctx, cancel = context.WithTimeout(context.Background(), timeout)
	err = commandRunner(ctx, claude, "mcp", "add", "--scope", "user", serverName, "--", binPath)
	cancel()
	if err != nil {
		return HostResult{Status: "failed", Reason: "claude mcp add failed: " + err.Error()}
	}
	return HostResult{Status: "installed", Reason: "registered through Claude CLI"}
}

func openCodeConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "opencode", "opencode.json"), nil
	}
	if home := os.Getenv("HOME"); home != "" {
		return filepath.Join(home, ".config", "opencode", "opencode.json"), nil
	}
	return "", errors.New("cannot determine OpenCode config path: HOME and XDG_CONFIG_HOME are unset")
}
