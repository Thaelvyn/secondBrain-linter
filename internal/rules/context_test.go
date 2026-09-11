package rules

import (
	"strings"
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func TestWikiTarget(t *testing.T) {
	cases := []struct{ in, want string }{
		{"[[a/b/c]]", "a/b/c"},
		{"[[a/b/c|Alias]]", "a/b/c"},
		{"[[a/b/c#anchor]]", "a/b/c"},
		{"[[a/b/c#anchor|Alias]]", "a/b/c"},
		{"  [[a/b]]  ", "a/b"},
	}
	for _, c := range cases {
		if got := wikiTarget(c.in); got != c.want {
			t.Errorf("wikiTarget(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestValidSlug(t *testing.T) {
	ok := []string{"2026-01-01-x.md", "jane-doe", "a", "x.txt", "0"}
	bad := []string{"Hello World.md", "Bad_Dir", "éclair.md", "UPPER.md", "has space"}
	for _, s := range ok {
		if !validSlug(s) {
			t.Errorf("validSlug(%q) = false, want true", s)
		}
	}
	for _, s := range bad {
		if validSlug(s) {
			t.Errorf("validSlug(%q) = true, want false", s)
		}
	}
}

func TestResolveWiki(t *testing.T) {
	v, err := scanTestVault(t, map[string]string{
		"personal/people/jane-doe/jane-doe.md": "---\ntype: entity\n---\n",
		"a.md":                                 "x\n",
		"b.md":                                 "x\n",
		"dir/dir.md":                           "x\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		target, want string
	}{
		{"a", "a.md"},
		{"a.md", "a.md"},
		{"b", "b.md"},
		{"dir", "dir/dir.md"},
		{"dir/dir", "dir/dir.md"},
		{"dir/dir.md", "dir/dir.md"},
		{"personal/people/jane-doe", "personal/people/jane-doe/jane-doe.md"},
		{"missing", ""},
		{"dir/missing", ""}, {"dir/", "dir/dir.md"},
		{"dir/#anchor", "dir/dir.md"},
		{"", ""},
	}
	for _, c := range cases {
		if got := resolveWiki(c.target, v); got != c.want {
			t.Errorf("resolveWiki(%q) = %q, want %q", c.target, got, c.want)
		}
	}
}

func TestRule10SkipsEmbeds(t *testing.T) {
	v, err := scanTestVault(t, map[string]string{
		"personal/note.md": "see ![[missing.png]] and [[also-missing]]\n",
		"todos.md":         "x\n",
		"inbox/review.md":  "x\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, emptyTax())
	Run(c)
	if len(c.Findings) != 2 {
		t.Fatalf("expected 2 findings (R10 broken link + R12 embed), got %+v", c.Findings)
	}
	if c.Findings[0].Rule != 10 || c.Findings[1].Rule != 12 {
		t.Fatalf("expected R10 then R12, got %+v", c.Findings)
	}
	if !strings.Contains(c.Findings[0].Message, "also-missing") {
		t.Fatalf("expected broken-link message for also-missing, got %q", c.Findings[0].Message)
	}
}

func scanTestVault(t *testing.T, files map[string]string) (*vault.Vault, error) {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := dir + "/" + rel
		if err := mkdirAll(p); err != nil {
			return nil, err
		}
		if err := writeFile(p, content); err != nil {
			return nil, err
		}
	}
	return vault.Scan(dir)
}
