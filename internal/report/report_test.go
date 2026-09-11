package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

func TestDisplayRooted(t *testing.T) {
	cases := []struct{ root, rel, want string }{
		{"julien", "This is a test.md", "julien/This is a test.md"},
		{".", "a.md", "a.md"},
		{"julien/", "a.md", "julien/a.md"},
		{"/vault", "a/b.md", "/vault/a/b.md"},
	}
	for _, c := range cases {
		if got := DisplayRooted(c.root, c.rel); got != c.want {
			t.Errorf("DisplayRooted(%q, %q) = %q, want %q", c.root, c.rel, got, c.want)
		}
	}
}

func TestConsole(t *testing.T) {
	fs := []vault.Finding{
		{Rule: 1, Severity: vault.SeverityError, Path: "x.md", Message: "bad slug"},
		{Rule: 14, Severity: vault.SeverityWarning, Path: "d", Message: "orphan"},
	}
	got := Console("julien", fs)
	want := "[error] R1 julien/x.md: bad slug\n[warning] R14 julien/d: orphan\n1 error, 1 warning\n"
	if got != want {
		t.Errorf("console output:\n%q\nwant:\n%q", got, want)
	}
}

func TestJSON(t *testing.T) {
	fs := []vault.Finding{
		{Rule: 5, Severity: vault.SeverityError, Path: "e.md", Message: "missing fields"},
		{Rule: 15, Severity: vault.SeverityWarning, Path: "deep/x.txt", Message: "depth"},
	}
	data, err := JSON("julien", fs)
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Vault    string `json:"vault"`
		Findings []struct {
			Rule     int    `json:"rule"`
			Severity string `json:"severity"`
			Path     string `json:"path"`
			Message  string `json:"message"`
		} `json:"findings"`
		Counts struct {
			Errors   int `json:"error"`
			Warnings int `json:"warning"`
		} `json:"counts"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Vault != "julien" {
		t.Errorf("vault = %q", got.Vault)
	}
	if len(got.Findings) != 2 || got.Findings[0].Rule != 5 || got.Findings[0].Path != "julien/e.md" {
		t.Errorf("findings = %+v", got.Findings)
	}
	if got.Counts.Errors != 1 || got.Counts.Warnings != 1 {
		t.Errorf("counts = %+v", got.Counts)
	}
}

func TestJSONEmpty(t *testing.T) {
	data, err := JSON("v", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"findings":[]`) {
		t.Errorf("empty findings must be []: %s", data)
	}
}

func TestWriteFile(t *testing.T) {
	root := t.TempDir()
	p, err := WriteFile(root, "console text\n")
	if err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(rel, filepath.Join("_system", "status", "lint", "sblint-")) || !strings.HasSuffix(rel, ".txt") {
		t.Errorf("report path = %q", rel)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "console text\n" {
		t.Errorf("report content = %q", data)
	}
	// name must embed a parsable timestamp
	base := filepath.Base(p)
	ts := strings.TrimSuffix(strings.TrimPrefix(base, "sblint-"), ".txt")
	if _, err := time.Parse("20060102-150405", ts); err != nil {
		t.Errorf("report timestamp %q unparseable", ts)
	}
}
