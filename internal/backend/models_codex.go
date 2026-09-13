package backend

// Model discovery uses the metadata-only app-server protocol. Actual turns
// continue to use exec JSONL. No thread or turn is created by this client.
import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/theboringhumane/theboringfloor/internal/state"
)

const (
	codexMetadataTimeout = 15 * time.Second
	codexMetadataBytes   = 8 << 20
	codexMetadataPages   = 100
)

func validateCodexModel(ref string) error {
	if ref == "" || len(ref) > 512 || !utf8.ValidString(ref) || strings.HasPrefix(ref, "-") || strings.ContainsFunc(ref, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}) {
		return errors.New("Codex model must be a nonempty native model token, without whitespace or control characters, and must not start with '-'")
	}
	return nil
}

func (b *codexBackend) SetModel(ctx context.Context, target state.ModelTarget, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if target.Agent != "" {
		return codexAgentModelUnsupported()
	}
	if ref == "" {
		return errors.New("Codex cannot reset the model of a resumed thread reliably; select an explicit native model instead")
	}
	if err := validateCodexModel(ref); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped {
		return errors.New("Codex is stopped")
	}
	b.model = ref
	return nil
}

func (b *codexBackend) ListModels(ctx context.Context) ([]state.ModelInfo, error) {
	var models []state.ModelInfo
	err := b.codexMetadata(ctx, func(rpc *codexMetadataRPC) error {
		var cursor *string
		seenCursor, seenModel := map[string]bool{}, map[string]bool{}
		for page := 0; page < codexMetadataPages; page++ {
			var result struct {
				Data []struct {
					ID, Model, Description string
					DisplayName            string `json:"displayName"`
					Hidden, Disabled       bool
					IsDefault              bool `json:"isDefault"`
				} `json:"data"`
				NextCursor *string `json:"nextCursor"`
			}
			if err := rpc.call("model/list", map[string]any{"cursor": cursor, "limit": 100, "includeHidden": false}, &result); err != nil {
				return fmt.Errorf("model catalog page %d: %w", page+1, err)
			}
			if result.Data == nil {
				return errors.New("Codex model/list returned no data array; update Codex CLI")
			}
			for _, row := range result.Data {
				if row.Hidden {
					continue
				}
				if err := validateCodexModel(row.Model); err != nil {
					return fmt.Errorf("Codex model/list returned an invalid model token: %w", err)
				}
				if seenModel[row.Model] {
					continue
				}
				seenModel[row.Model] = true
				name := row.DisplayName
				if name == "" {
					name = row.Model
				}
				models = append(models, state.ModelInfo{ID: row.ID, Ref: row.Model, Name: name,
					Description: row.Description, Disabled: row.Disabled, IsDefault: row.IsDefault})
			}
			if result.NextCursor == nil {
				return nil
			}
			if seenCursor[*result.NextCursor] {
				return errors.New("Codex model/list repeated a pagination cursor")
			}
			seenCursor[*result.NextCursor] = true
			cursor = result.NextCursor
		}
		return errors.New("Codex model/list exceeded the catalog page limit")
	})
	if err != nil {
		return nil, err // Never publish a partial catalog after a failed page.
	}
	return models, nil
}

type codexMetadataRPC struct {
	in     io.Writer
	out    *bufio.Scanner
	ctx    context.Context
	nextID int
	bytes  int
}

func (rpc *codexMetadataRPC) call(method string, params any, result any) error {
	rpc.nextID++
	id := rpc.nextID
	if err := json.NewEncoder(rpc.in).Encode(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return fmt.Errorf("Codex %s request: %w", method, err)
	}
	for rpc.out.Scan() {
		raw := rpc.out.Bytes()
		rpc.bytes += len(raw) + 1
		if rpc.bytes > codexMetadataBytes {
			return errors.New("Codex metadata exceeded the response byte limit")
		}
		if err := rpc.ctx.Err(); err != nil {
			return err
		}
		var message struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(raw, &message); err != nil {
			return fmt.Errorf("Codex %s returned invalid JSON: %w", method, err)
		}
		// Notifications and unrelated response IDs may be interleaved.
		if string(message.ID) != strconv.Itoa(id) {
			continue
		}
		if message.Error != nil {
			return fmt.Errorf("Codex %s: %s (RPC %d); update Codex CLI if this method is unsupported", method, message.Error.Message, message.Error.Code)
		}
		if len(message.Result) == 0 || string(message.Result) == "null" {
			return fmt.Errorf("Codex %s response omitted its result", method)
		}
		if err := json.Unmarshal(message.Result, result); err != nil {
			return fmt.Errorf("Codex %s result: %w", method, err)
		}
		return nil
	}
	if err := rpc.ctx.Err(); err != nil {
		return err
	}
	if err := rpc.out.Err(); err != nil {
		return fmt.Errorf("Codex %s response: %w", method, err)
	}
	return fmt.Errorf("Codex app-server exited before responding to %s", method)
}

func (b *codexBackend) codexMetadata(ctx context.Context, query func(*codexMetadataRPC) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, codexMetadataTimeout)
	defer cancel()
	b.mu.Lock()
	bin, dir, stopped := b.bin, b.dir, b.stopped
	b.mu.Unlock()
	if stopped {
		return errors.New("Codex is stopped")
	}
	cmd := exec.CommandContext(ctx, bin, "app-server", "--stdio")
	cmd.Dir = dir
	isolateProcessGroup(cmd)
	cmd.Cancel = func() error { return signalProcessGroup(cmd.Process, syscall.SIGKILL) }
	cmd.WaitDelay = stopKillGrace
	in, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer out.Close()
	tail := &codexTail{}
	cmd.Stderr = tail
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Codex metadata server (install or update Codex CLI): %w", err)
	}
	defer func() {
		// Always kill the entire group, even after successful metadata reads;
		// app-server is a service and might hold tool subprocesses or pipe FDs.
		_ = signalProcessGroup(cmd.Process, syscall.SIGKILL)
		_ = cmd.Wait()
		if err != nil && len(tail.data) != 0 {
			err = fmt.Errorf("%w; Codex stderr: %s", err, strings.TrimSpace(string(tail.data)))
		}
	}()
	scanner := bufio.NewScanner(io.LimitReader(out, codexMetadataBytes+1))
	scanner.Buffer(make([]byte, 4096), 1<<20)
	rpc := &codexMetadataRPC{in: in, out: scanner, ctx: ctx}
	var initialized map[string]any
	if err := rpc.call("initialize", map[string]any{"clientInfo": map[string]string{"name": "theboringfloor", "version": "1"}, "capabilities": map[string]any{}}, &initialized); err != nil {
		return fmt.Errorf("Codex metadata handshake: %w", err)
	}
	if err := json.NewEncoder(in).Encode(map[string]string{"method": "initialized"}); err != nil {
		return fmt.Errorf("Codex initialized notification: %w", err)
	}
	return query(rpc)
}
