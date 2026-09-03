//go:build windows

package term

import (
	"fmt"
	"os"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// startupInfoEx is STARTUPINFOEXW; x/sys/windows doesn't export one.
type startupInfoEx struct {
	windows.StartupInfo
	lpAttributeList *windows.ProcThreadAttributeList
}

// conptyMaster wraps the two unidirectional ConPTY pipe ends into a
// single io.ReadWriteCloser. Ready for use once ptyResult.master is
// widened from *os.File to io.ReadWriteCloser.
type conptyMaster struct {
	r *os.File // ptyOutRead  — shell output
	w *os.File // ptyInWrite  — shell input
}

func (m *conptyMaster) Read(p []byte) (int, error)  { return m.r.Read(p) }
func (m *conptyMaster) Write(p []byte) (int, error) { return m.w.Write(p) }
func (m *conptyMaster) Close() error {
	m.w.Close()
	return m.r.Close()
}

func platformStart(cfg TermConfig, env []string) (*ptyResult, error) {
	// --- two pipe pairs for ConPTY I/O ---
	var ptyInR, ptyInW, ptyOutR, ptyOutW windows.Handle
	sa := &windows.SecurityAttributes{
		Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
	}
	if err := windows.CreatePipe(&ptyInR, &ptyInW, sa, 0); err != nil {
		return nil, fmt.Errorf("conpty: input pipe: %w", err)
	}
	if err := windows.CreatePipe(&ptyOutR, &ptyOutW, sa, 0); err != nil {
		windows.CloseHandle(ptyInR)
		windows.CloseHandle(ptyInW)
		return nil, fmt.Errorf("conpty: output pipe: %w", err)
	}
	ok := false
	defer func() {
		if !ok {
			windows.CloseHandle(ptyInR)
			windows.CloseHandle(ptyInW)
			windows.CloseHandle(ptyOutR)
			windows.CloseHandle(ptyOutW)
		}
	}()

	// --- pseudo console ---
	coord := windows.Coord{X: int16(cfg.Cols), Y: int16(cfg.Rows)}
	var hPC windows.Handle
	if err := windows.CreatePseudoConsole(coord, ptyInR, ptyOutW, 0, &hPC); err != nil {
		return nil, fmt.Errorf("conpty: create console: %w", err)
	}
	defer func() {
		if !ok {
			windows.ClosePseudoConsole(hPC)
		}
	}()

	// --- thread attribute list ---
	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, fmt.Errorf("conpty: attribute list: %w", err)
	}
	defer func() {
		if !ok {
			attrList.Delete()
		}
	}()
	if err := attrList.Update(
		windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		unsafe.Pointer(&hPC),
		unsafe.Sizeof(hPC),
	); err != nil {
		return nil, fmt.Errorf("conpty: set console attribute: %w", err)
	}

	// --- command line + cwd ---
	cmdLine, err := windows.UTF16PtrFromString(cfg.Shell)
	if err != nil {
		return nil, fmt.Errorf("conpty: shell path: %w", err)
	}
	var cwdPtr *uint16
	if cfg.CWD != "" {
		cwdPtr, err = windows.UTF16PtrFromString(cfg.CWD)
		if err != nil {
			return nil, fmt.Errorf("conpty: cwd: %w", err)
		}
	}

	// --- CreateProcess ---
	si := startupInfoEx{lpAttributeList: attrList.List()}
	si.Cb = uint32(unsafe.Sizeof(si))

	var pi windows.ProcessInformation
	err = windows.CreateProcess(
		nil, cmdLine,
		nil, nil, false,
		windows.EXTENDED_STARTUPINFO_PRESENT|windows.CREATE_UNICODE_ENVIRONMENT,
		buildEnvBlock(env), cwdPtr,
		&si.StartupInfo, &pi,
	)
	if err != nil {
		return nil, fmt.Errorf("conpty: create process: %w", err)
	}

	// Process is live — close ConPTY-side pipe ends and the thread handle.
	ok = true
	windows.CloseHandle(ptyInR)
	windows.CloseHandle(ptyOutW)
	windows.CloseHandle(pi.Thread)

	// Wrap our pipe ends as Go files.
	readFile := os.NewFile(uintptr(ptyOutR), "conpty-out")
	writeFile := os.NewFile(uintptr(ptyInW), "conpty-in")

	return &ptyResult{
		master: &conptyMaster{r: readFile, w: writeFile},
		pid:    int(pi.ProcessId),
		waitFn: func() (int, error) {
			defer windows.CloseHandle(pi.Process)
			if _, err := windows.WaitForSingleObject(pi.Process, windows.INFINITE); err != nil {
				return -1, fmt.Errorf("conpty: wait: %w", err)
			}
			var code uint32
			if err := windows.GetExitCodeProcess(pi.Process, &code); err != nil {
				return -1, fmt.Errorf("conpty: exit code: %w", err)
			}
			return int(code), nil
		},
		resizeFn: func(cols, rows int) error {
			return windows.ResizePseudoConsole(hPC, windows.Coord{
				X: int16(cols), Y: int16(rows),
			})
		},
		closeFn: func() {
			windows.ClosePseudoConsole(hPC)
			attrList.Delete()
		},
	}, nil
}

// buildEnvBlock creates a Windows environment block: null-separated
// KEY=VALUE pairs, double-null terminated, UTF-16 encoded.
func buildEnvBlock(env []string) *uint16 {
	if len(env) == 0 {
		return nil
	}
	var runes []rune
	for _, e := range env {
		runes = append(runes, []rune(e)...)
		runes = append(runes, 0)
	}
	runes = append(runes, 0) // double-null terminator
	block := utf16.Encode(runes)
	return &block[0]
}
