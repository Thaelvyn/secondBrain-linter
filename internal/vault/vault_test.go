package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestScanBasic(t *testing.T) {
	root := t.TempDir()
	write(t, root, "README.md", "hi\n")
	write(t, root, "personal/personal.md", "---\ntype: collection\n---\nbody\n")
	write(t, root, "personal/people/people.md", "---\ntype: collection\n---\n")
	write(t, root, "attachments/x.png", "PNG")
	write(t, root, ".obsidian/config.json", "{}")
	write(t, root, "personal/.hidden.md", "x")

	v, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Files) != 4 { // README, personal.md, people.md, x.png
		t.Fatalf("files = %d, want 4", len(v.Files))
	}
	for _, f := range v.Files {
		if f.Name == ".hidden.md" || f.Name == "config.json" {
			t.Errorf("dot files must be skipped, got %s", f.Path)
		}
		if f.Name == ".obsidian" {
			t.Errorf("dot dir must be skipped")
		}
	}
	pm, ok := v.ByPath["personal/personal.md"]
	if !ok {
		t.Fatal("personal/personal.md missing")
	}
	if !pm.HasFM || pm.Frontmatter["type"] != "collection" {
		t.Errorf("frontmatter not parsed: %+v", pm.Frontmatter)
	}
	png := v.ByPath["attachments/x.png"]
	if png == nil || png.Markdown {
		t.Error("non-markdown file misclassified")
	}
}

func TestParseFrontmatter(t *testing.T) {
	cases := []struct {
		in     string
		hasFM  bool
		key, v string
	}{
		{"---\ntype: entity\n---\n", true, "type", "entity"},
		{"no frontmatter\n", false, "", ""},
		{"---\nunterminated\n", false, "", ""},
		{"---\n", false, "", ""},
		{"---\na: [unclosed\n---\n", false, "", ""},
		{"---\ntype: day-note\ndate: 2026-01-01\n---\n\nbody", true, "type", "day-note"},
	}
	for _, c := range cases {
		has, fm := parseFrontmatter([]byte(c.in))
		if has != c.hasFM {
			t.Errorf("parseFrontmatter(%q) has = %v, want %v", c.in, has, c.hasFM)
		}
		if has && fm[c.key] != c.v {
			t.Errorf("parseFrontmatter(%q) fm[%s] = %v, want %v", c.in, c.key, fm[c.key], c.v)
		}
	}
}

func TestScanNotADir(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "x")
	_, err := Scan(filepath.Join(root, "a.txt"))
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
	_, err = Scan(filepath.Join(root, "nope"))
	if err == nil {
		t.Fatal("expected error for missing path")
	}
	_, err = Scan(filepath.Join(root, "a.txt"))
	if err == nil || !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("expected not-a-directory error, got %v", err)
	}
}

func TestScanSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "real.md", "x")
	if err := os.Symlink(filepath.Join(root, "real.md"), filepath.Join(root, "link.md")); err != nil {
		t.Skip("symlinks unsupported")
	}
	v, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Files) != 1 {
		t.Fatalf("symlink must be skipped, files=%d", len(v.Files))
	}
}

func TestScanEmptyDir(t *testing.T) {
	v, err := Scan(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Files) != 0 || len(v.Dirs) != 0 {
		t.Fatalf("empty dir: files=%d dirs=%d", len(v.Files), len(v.Dirs))
	}
}

func TestScanReadsMarkdownContent(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.md", "---\nx: 1\n---\nbody [[link]]\n")
	v, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	f := v.ByPath["a.md"]
	if f == nil || !f.Markdown {
		t.Fatal("a.md not loaded as markdown")
	}
	if !f.HasFM || f.Frontmatter["x"] != 1 {
		t.Errorf("frontmatter: %+v", f.Frontmatter)
	}
	if !containsStr(f.Content, "[[link]]") {
		t.Errorf("content not fully read: %q", f.Content)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
