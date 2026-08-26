// soundexport — writes the office's seven chimes into the website's static
// assets, byte-identical to what the app synthesizes at runtime.
//
//	go run ./cmd/soundexport        regenerate website/public/sounds/*.wav
//
// The wavs come straight from internal/sound (fixed seeds, fixed phases), so
// they are bit-for-bit reproducible. Run manually when the synthesis changes;
// nothing is go:generate-hooked.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/theboringhumane/theboringoffice/internal/sound"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "soundexport:", err)
		os.Exit(1)
	}
}

func run() error {
	// Resolve the repo root from this file's location so the exporter works
	// from any working directory.
	_, self, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("locate exporter source path")
	}
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(self)))
	destDir := filepath.Join(repoRoot, "website", "public", "sounds")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "soundexport-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	for _, name := range sound.Names() {
		src, err := sound.EnsureWav(tmp, name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		// Write a sibling then rename over the target — the replace is
		// atomic, so the site never sees a half-written wav.
		dest := filepath.Join(destDir, name+".wav")
		staging := dest + ".tmp"
		if err := os.WriteFile(staging, data, 0o644); err != nil {
			return err
		}
		if err := os.Rename(staging, dest); err != nil {
			return err
		}
		fmt.Println(dest)
	}
	return nil
}
