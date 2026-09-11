// Package changes computes the set of vault-relative files changed since the
// last lint run for the sblint --changed mode. It prefers git (the vault is
// inside a work tree); otherwise it falls back to an mtime state file under
// <root>/_system/status/lint/.sblint-changed.json.
package changes

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Mode identifies the change source used by Detect.
type Mode string

const (
	// ModeGit means changed files came from git.
	ModeGit Mode = "git"
	// ModeMtime means changed files came from the mtime state file fallback.
	ModeMtime Mode = "mtime"
)

// Set is a set of vault-relative changed paths (forward slashes).
type Set map[string]bool

// Detect returns the changed vault-relative paths for root. Git is used when
// available and root is inside a work tree; any git failure falls back to the
// mtime state file. The returned Mode tells the caller whether to call
// SaveMtime after the run.
func Detect(root string) (Set, Mode, error) {
	if set, ok := gitChanged(root); ok {
		return set, ModeGit, nil
	}
	set, err := mtimeChanged(root)
	return set, ModeMtime, err
}

func gitChanged(root string) (Set, bool) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, false
	}
	if _, err := git(root, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, false
	}
	set := Set{}
	for _, args := range [][]string{
		{"diff", "--relative", "--name-only", "HEAD"},
		{"ls-files", "--others", "--exclude-standard"},
	} {
		out, err := git(root, args...)
		if err != nil {
			return nil, false
		}
		for _, line := range strings.Split(out, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				set[line] = true
			}
		}
	}
	return set, true
}

func git(root string, args ...string) (string, error) {
	full := append([]string{"-C", root, "-c", "core.quotepath=false"}, args...)
	cmd := exec.Command("git", full...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return out.String(), nil
}

// statePath is the mtime fallback state file. The leading dot keeps it out of
// the vault scan.
func statePath(root string) string {
	return filepath.Join(root, "_system", "status", "lint", ".sblint-changed.json")
}

type mtimeState struct {
	Version int              `json:"version"`
	Seen    map[string]int64 `json:"seen"`
}

func mtimeChanged(root string) (Set, error) {
	prev := mtimeState{}
	if data, err := os.ReadFile(statePath(root)); err == nil {
		_ = json.Unmarshal(data, &prev)
	}
	set := Set{}
	_, err := walkVault(root, func(rel string, mt int64) {
		old, ok := prev.Seen[rel]
		if !ok || mt > old {
			set[rel] = true
		}
	})
	return set, err
}

// SaveMtime records the current vault file mtimes so the next mtime-fallback
// --changed run can diff against this run.
func SaveMtime(root string) error {
	st := mtimeState{Version: 1, Seen: map[string]int64{}}
	if _, err := walkVault(root, func(rel string, mt int64) { st.Seen[rel] = mt }); err != nil {
		return err
	}
	p := statePath(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

func walkVault(root string, visit func(rel string, mtime int64)) (int, error) {
	n := 0
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && strings.HasPrefix(d.Name(), ".") {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		rel, rerr := filepath.Rel(root, p)
		if rerr != nil {
			return rerr
		}
		info, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		visit(filepath.ToSlash(rel), info.ModTime().UnixNano())
		n++
		return nil
	})
	return n, err
}
