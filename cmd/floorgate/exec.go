package main

// The exec route grants full shell access to the host. It is disabled by
// default and must only ever be exposed on a tailnet or loopback, never a
// public interface.

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	maxExecBodyBytes   = 1 << 20
	maxExecOutput      = 256 << 10
	defaultExecTimeout = 30 * time.Second
	maxExecTimeout     = 120 * time.Second
)

type execRequest struct {
	Command   string `json:"command"`
	CWD       string `json:"cwd"`
	TimeoutMS int    `json:"timeoutMs"`
}

type execResponse struct {
	Stdout     string `json:"stdout"`
	Stderr     string `json:"stderr"`
	ExitCode   int    `json:"exitCode"`
	DurationMS int64  `json:"durationMs"`
	Truncated  bool   `json:"truncated"`
}

type cappedBuffer struct {
	bytes     []byte
	truncated bool
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	remaining := maxExecOutput - len(b.bytes)
	if remaining > 0 {
		if len(p) > remaining {
			b.bytes = append(b.bytes, p[:remaining]...)
			b.truncated = true
		} else {
			b.bytes = append(b.bytes, p...)
		}
	} else if len(p) > 0 {
		b.truncated = true
	}
	return len(p), nil
}

func (g *gateway) execCommand(w http.ResponseWriter, r *http.Request) {
	if !g.exec {
		writeError(w, http.StatusForbidden, "remote execution is disabled; enable with -exec or FLOORGATE_EXEC=1")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxExecBodyBytes)
	defer r.Body.Close()
	var request execRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeError(w, http.StatusRequestEntityTooLarge, "exec request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid exec request")
		return
	}
	if err := rejectTrailingJSON(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "invalid exec request")
		return
	}
	if strings.TrimSpace(request.Command) == "" {
		writeError(w, http.StatusBadRequest, "empty command")
		return
	}
	if request.CWD != "" {
		info, err := os.Stat(request.CWD)
		if err != nil || !info.IsDir() {
			writeError(w, http.StatusBadRequest, "cwd must be an existing directory")
			return
		}
	}

	timeout := defaultExecTimeout
	if request.TimeoutMS > 0 {
		timeout = time.Duration(request.TimeoutMS) * time.Millisecond
		if timeout > maxExecTimeout {
			timeout = maxExecTimeout
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	command := exec.CommandContext(ctx, "/bin/sh", "-c", request.Command)
	command.Dir = request.CWD
	prepareExecProcessGroup(command)
	command.Cancel = func() error { return killExecProcessGroup(command) }

	var stdout, stderr cappedBuffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	started := time.Now()
	err := command.Run()
	duration := time.Since(started).Milliseconds()

	response := execResponse{
		Stdout:     string(stdout.bytes),
		Stderr:     string(stderr.bytes),
		DurationMS: duration,
		Truncated:  stdout.truncated || stderr.truncated,
	}
	if ctx.Err() == context.DeadlineExceeded {
		response.ExitCode = -1
		response.Stderr += "command timed out\n"
	} else if err == nil {
		response.ExitCode = 0
	} else {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			response.ExitCode = exitError.ExitCode()
		} else {
			writeError(w, http.StatusInternalServerError, "could not execute command")
			return
		}
	}
	writeJSON(w, http.StatusOK, response)
}
