package changes

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	full := append([]string{"-C", root}, args...)
	cmd := exec.Command("git", full...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestMtimeFallback(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.md", "a\n")
	write(t, root, "sub/b.md", "b\n")

	set, mode, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if mode != ModeMtime {
		t.Fatalf("mode = %s, want mtime", mode)
	}
	if !set["a.md"] || !set["sub/b.md"] {
		t.Fatalf("first run without state must report everything: %v", set)
	}
	// Record b.md with an old mtime so the later modification is guaranteed
	// detectable even on coarse-granularity filesystems.
	past := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(filepath.Join(root, "sub/b.md"), past, past); err != nil {
		t.Fatal(err)
	}
	if err := SaveMtime(root); err != nil {
		t.Fatal(err)
	}

	set, _, err = Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(set) != 0 {
		t.Fatalf("unchanged vault after save must be clean: %v", set)
	}

	write(t, root, "sub/b.md", "b changed\n")
	set, _, err = Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !set["sub/b.md"] || len(set) != 1 {
		t.Fatalf("changed = %v, want only sub/b.md", set)
	}
}

func TestMtimeSkipsDotFiles(t *testing.T) {
	root := t.TempDir()
	write(t, root, "x.md", "x\n")
	write(t, root, ".hidden.md", "h\n")
	set, _, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if set[".hidden.md"] {
		t.Fatalf("dot files must be skipped: %v", set)
	}
}

func TestGitChangedNoCommitsFallsBack(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "t@t")
	runGit(t, root, "config", "user.name", "t")
	write(t, root, "a.md", "a\n")

	set, mode, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if mode != ModeMtime {
		t.Fatalf("repo without commits must fall back to mtime, mode = %s", mode)
	}
	if !set["a.md"] {
		t.Fatalf("fallback must report the untracked file: %v", set)
	}
}

func TestSaveMtimeError(t *testing.T) {
	root := t.TempDir()
	// _system as a regular file blocks the state directory creation.
	write(t, root, "_system", "not a dir")
	write(t, root, "x.md", "x\n")
	if err := SaveMtime(root); err == nil {
		t.Fatal("expected SaveMtime to fail when the state dir cannot be created")
	}
}

func TestDetectWalkError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	root := t.TempDir()
	write(t, root, "x.md", "x\n")
	if err := os.Chmod(root, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(root, 0o755)
	if _, _, err := Detect(root); err == nil {
		t.Fatal("expected Detect to fail on an unreadable vault")
	}
}

func TestGitChanged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "t@t")
	runGit(t, root, "config", "user.name", "t")
	write(t, root, "a.md", "a\n")
	write(t, root, "sub/b.md", "b\n")
	runGit(t, root, "add", "-A")
	runGit(t, root, "commit", "-qm", "init")

	set, mode, err := Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if mode != ModeGit {
		t.Fatalf("mode = %s, want git", mode)
	}
	if len(set) != 0 {
		t.Fatalf("clean repo must be empty: %v", set)
	}

	write(t, root, "sub/b.md", "b changed\n")
	write(t, root, "new.md", "new\n")
	set, _, err = Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if !set["sub/b.md"] || !set["new.md"] || len(set) != 2 {
		t.Fatalf("changed = %v, want sub/b.md and new.md", set)
	}
}
