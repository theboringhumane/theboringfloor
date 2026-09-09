package workspace

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

type FileEntry struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	Directory bool   `json:"directory"`
}
type FilePage struct {
	Path      string      `json:"path"`
	Entries   []FileEntry `json:"entries,omitempty"`
	Content   string      `json:"content,omitempty"`
	Directory bool        `json:"directory"`
	Truncated bool        `json:"truncated"`
	Binary    bool        `json:"binary"`
}

// OpenRoot confines every lookup, including symlinks, to the project even if
// a repository entry is replaced while the request is being served.
func ReadFiles(dir, name string) (FilePage, error) {
	page := FilePage{Path: name}
	if name == "" {
		name = "."
	}
	if !filepath.IsLocal(name) {
		return page, errors.New("path must stay inside this project")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return page, err
	}
	defer root.Close()
	info, err := root.Stat(name)
	if err != nil {
		return page, err
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return page, errors.New("only regular files can be previewed")
	}
	f, err := root.Open(name)
	if err != nil {
		return page, err
	}
	defer f.Close()
	page.Directory = info.IsDir()
	if info.IsDir() {
		entries, err := f.ReadDir(1001)
		if err != nil && err != io.EOF {
			return page, err
		}
		page.Truncated = len(entries) > 1000
		if page.Truncated {
			entries = entries[:1000]
		}
		page.Entries = []FileEntry{}
		for _, entry := range entries {
			if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules" || entry.Name() == ".dart_tool" || entry.Name() == "build") {
				continue
			}
			path := filepath.Join(name, entry.Name())
			stat, err := root.Stat(path)
			if err != nil {
				continue
			}
			if !stat.IsDir() && !stat.Mode().IsRegular() {
				continue
			}
			page.Entries = append(page.Entries, FileEntry{Name: entry.Name(), Path: filepath.ToSlash(path), Directory: stat.IsDir()})
		}
		sort.Slice(page.Entries, func(i, j int) bool {
			a, b := page.Entries[i], page.Entries[j]
			if a.Directory != b.Directory {
				return a.Directory
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		})
		return page, nil
	}
	raw, err := io.ReadAll(io.LimitReader(f, 64*1024+1))
	if err != nil {
		return page, err
	}
	page.Truncated = len(raw) > 64*1024
	if page.Truncated {
		raw = raw[:64*1024]
		for len(raw) > 0 && !utf8.Valid(raw) && len(raw) > 64*1024-4 {
			raw = raw[:len(raw)-1]
		}
	}
	page.Binary = bytes.IndexByte(raw, 0) >= 0 || !utf8.Valid(raw)
	if !page.Binary {
		page.Content = string(raw)
	}
	return page, nil
}
