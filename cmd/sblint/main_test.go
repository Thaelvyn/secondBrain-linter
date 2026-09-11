package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runMain(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, &out, &errb)
	return code, out.String(), errb.String()
}

func cleanFixture(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("../../testdata/clean-fixture")
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

func TestVersion(t *testing.T) {
	code, out, _ := runMain(t, "--version")
	if code != 0 || out != "sblint v0.2.0\n" {
		t.Fatalf("version: code=%d out=%q", code, out)
	}
}

func TestNoArgsUsage(t *testing.T) {
	code, _, errb := runMain(t)
	if code != 2 || !strings.Contains(errb, "Usage: sblint") {
		t.Fatalf("no args: code=%d err=%q", code, errb)
	}
}

func TestUnknownFlag(t *testing.T) {
	code, _, _ := runMain(t, "--nope")
	if code != 2 {
		t.Fatalf("unknown flag: code=%d", code)
	}
}

func TestMissingVaultExit2(t *testing.T) {
	code, _, errb := runMain(t, filepath.Join(t.TempDir(), "missing"))
	if code != 2 || errb == "" {
		t.Fatalf("missing vault: code=%d err=%q", code, errb)
	}
}

func TestCleanExit0(t *testing.T) {
	code, out, _ := runMain(t, cleanFixture(t))
	if code != 0 {
		t.Fatalf("clean fixture: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "0 errors, 0 warnings") {
		t.Errorf("summary missing: %q", out)
	}
}

func TestViolationsExit1(t *testing.T) {
	code, out, _ := runMain(t, "../../testdata/violations-fixture")
	if code != 1 {
		t.Fatalf("violations fixture: code=%d", code)
	}
	if !strings.Contains(out, "[error] R1 ../../testdata/violations-fixture/personal/Hello World.md") {
		t.Errorf("expected rooted console line, got:\n%s", out)
	}
	if !strings.Contains(out, "26 errors, 9 warnings") {
		t.Errorf("expected summary, got:\n%s", out)
	}
}

func TestJSONOutput(t *testing.T) {
	code, out, _ := runMain(t, "--json", "../../testdata/violations-fixture")
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	var got struct {
		Vault    string `json:"vault"`
		Findings []struct {
			Rule     int    `json:"rule"`
			Severity string `json:"severity"`
		} `json:"findings"`
		Counts map[string]int `json:"counts"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if got.Vault != "../../testdata/violations-fixture" {
		t.Errorf("vault = %q", got.Vault)
	}
	if len(got.Findings) != 35 {
		t.Errorf("findings = %d", len(got.Findings))
	}
	if got.Counts["error"] != 26 || got.Counts["warning"] != 9 {
		t.Errorf("counts = %+v", got.Counts)
	}
}

func TestReportWritesFile(t *testing.T) {
	src := t.TempDir()
	copyTree(t, "../../testdata/clean-fixture", src)
	code, out, errb := runMain(t, "--report", src)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errb)
	}
	if !strings.Contains(errb, "report written:") {
		t.Errorf("expected report notice on stderr, got %q", errb)
	}
	files, err := filepath.Glob(filepath.Join(src, "_system", "status", "lint", "sblint-*.txt"))
	if err != nil || len(files) != 1 {
		t.Fatalf("report files: %v %v", files, err)
	}
	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "0 errors, 0 warnings") {
		t.Errorf("report content: %q", data)
	}
	_ = out
}

func TestReportAndJSONTogether(t *testing.T) {
	src := t.TempDir()
	copyTree(t, "../../testdata/violations-fixture", src)
	code, out, _ := runMain(t, "--json", "--report", src)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("stdout must be pure JSON, got %q", out)
	}
	g, err := filepath.Glob(filepath.Join(src, "_system", "status", "lint", "sblint-*.txt"))
	if err != nil || len(g) != 1 {
		t.Fatalf("report files: %v %v", g, err)
	}
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		d := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(d, 0o755)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(d, data, 0o644)
	})
	if err != nil {
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

func TestFixIdempotent(t *testing.T) {
	src := t.TempDir()
	copyTree(t, "../../testdata/violations-fixture", src)

	code1, out1, err1 := runMain(t, "--fix", src)
	if code1 != 1 {
		t.Fatalf("first fix must leave residual violations, code=%d\n%s", code1, out1)
	}
	if !strings.Contains(err1, "fixed ") {
		t.Errorf("expected fix summary on stderr, got %q", err1)
	}

	code2, _, err2 := runMain(t, "--fix", src)
	if code2 != 1 {
		t.Fatalf("residual violations persist, code=%d", code2)
	}
	if !strings.Contains(err2, "fixed 0 file(s)") {
		t.Errorf("second fix must be a no-op, stderr=%q", err2)
	}

	// The R19 drift is gone after the fix pass.
	code3, out3, _ := runMain(t, src)
	if code3 != 1 {
		t.Fatalf("residual violations persist, code=%d", code3)
	}
	if strings.Contains(out3, "R19") {
		t.Errorf("rule 19 findings must be fixed, got:\n%s", out3)
	}
}

func TestRecountAndIdempotency(t *testing.T) {
	src := t.TempDir()
	copyTree(t, "../../testdata/violations-fixture", src)

	code, out, errb := runMain(t, "--recount", src)
	if code != 1 {
		t.Fatalf("residual violations persist, code=%d", code)
	}
	if !strings.Contains(errb, "recounted 3 file(s)") {
		t.Errorf("expected recount summary, stderr=%q", errb)
	}
	if strings.Contains(out, "R19") {
		t.Errorf("recount must clear R19 findings, got:\n%s", out)
	}

	code2, _, errb2 := runMain(t, "--recount", src)
	if code2 != 1 {
		t.Fatalf("code=%d", code2)
	}
	if !strings.Contains(errb2, "recounted 0 file(s)") {
		t.Errorf("second recount must be a no-op, stderr=%q", errb2)
	}
}

func TestChangedGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	src := t.TempDir()
	copyTree(t, "../../testdata/clean-fixture", src)
	runGit(t, src, "init", "-q")
	runGit(t, src, "config", "user.email", "t@t")
	runGit(t, src, "config", "user.name", "t")
	runGit(t, src, "add", "-A")
	runGit(t, src, "commit", "-qm", "init")

	// Nothing changed: clean output, exit 0.
	code, out, _ := runMain(t, "--changed", src)
	if code != 0 || !strings.Contains(out, "0 errors, 0 warnings") {
		t.Fatalf("unchanged repo: code=%d out=%q", code, out)
	}

	// Break only one entry's source: only its R20 finding is reported.
	p := filepath.Join(src, "personal", "people", "jane-doe", "meeting", "2026", "2026-01-01-kickoff.md")
	broken := strings.Replace(readFile(t, p), "source: ./daily-logs/2026/09/2026-09-10-1432.md", "source: daily-logs/2026/09/2026-09-10-nope.md", 1)
	os.WriteFile(p, []byte(broken), 0o644)

	code, out, _ = runMain(t, "--changed", src)
	if code != 1 {
		t.Fatalf("code=%d", code)
	}
	if !strings.Contains(out, "R20") || !strings.Contains(out, "1 error, 0 warnings") {
		t.Errorf("--changed must report only the broken entry, got:\n%s", out)
	}

	// Commit the break and a new untracked file: both are reported.
	runGit(t, src, "add", "-A")
	runGit(t, src, "commit", "-qm", "break")
	os.WriteFile(filepath.Join(src, "stray.md"), []byte("stray\n"), 0o644)
	code, out, _ = runMain(t, "--changed", src)
	if code != 1 || !strings.Contains(out, "R17") {
		t.Fatalf("untracked file must be reported: code=%d out=%q", code, out)
	}
}

func TestChangedMtimeFallback(t *testing.T) {
	src := t.TempDir() // not a git repo -> mtime fallback
	copyTree(t, "../../testdata/clean-fixture", src)

	// Record the entry with an old mtime so the later modification is
	// guaranteed detectable even on coarse-granularity filesystems.
	p := filepath.Join(src, "personal", "people", "jane-doe", "meeting", "2026", "2026-01-01-kickoff.md")
	past := time.Now().Add(-2 * time.Second)
	if err := os.Chtimes(p, past, past); err != nil {
		t.Fatal(err)
	}

	code, _, _ := runMain(t, "--changed", src)
	if code != 0 {
		t.Fatalf("first run over a clean vault: code=%d", code)
	}

	broken := strings.Replace(readFile(t, p), "source: ./daily-logs/2026/09/2026-09-10-1432.md", "source: daily-logs/2026/09/2026-09-10-nope.md", 1)
	if err := os.WriteFile(p, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	code, out, _ := runMain(t, "--changed", src)
	if code != 1 || !strings.Contains(out, "R20") {
		t.Fatalf("modified file must be reported: code=%d out=%q", code, out)
	}

	code, out, _ = runMain(t, "--changed", src)
	if code != 0 || !strings.Contains(out, "0 errors, 0 warnings") {
		t.Fatalf("no further changes: code=%d out=%q", code, out)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestFixJSON(t *testing.T) {
	src := t.TempDir()
	copyTree(t, "../../testdata/violations-fixture", src)
	code, out, errb := runMain(t, "--fix", "--json", src)
	if code != 1 {
		t.Fatalf("residual violations persist, code=%d", code)
	}
	if !json.Valid([]byte(out)) {
		t.Fatalf("stdout must be pure JSON, got %q", out)
	}
	if !strings.Contains(errb, "fixed ") {
		t.Errorf("expected fix summary on stderr, got %q", errb)
	}
	if strings.Contains(out, `"rule":19`) {
		t.Errorf("R19 findings must be fixed, JSON has:\n%s", out)
	}
}

func TestChangedStateSaveFailure(t *testing.T) {
	src := t.TempDir()
	// _system as a regular file makes the mtime state dir uncreatable.
	os.WriteFile(filepath.Join(src, "_system"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(src, "stray.md"), []byte("x\n"), 0o644)
	code, _, errb := runMain(t, "--changed", src)
	if code != 1 {
		t.Fatalf("stray root file must be a finding, code=%d", code)
	}
	if !strings.Contains(errb, "cannot save --changed state") {
		t.Errorf("expected state-save notice on stderr, got %q", errb)
	}
}

func TestTaxonomyMissing(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "Hello World.md"), []byte("x\n"), 0o644)
	code, out, _ := runMain(t, src)
	if code != 1 {
		t.Fatalf("missing taxonomy must surface as a rule 4 finding, code=%d", code)
	}
	if !strings.Contains(out, "cannot read _system/taxonomy.yaml") {
		t.Errorf("expected taxonomy error finding, got:\n%s", out)
	}
}

func TestFixWriteError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks do not apply to root")
	}
	src := t.TempDir()
	copyTree(t, "../../testdata/violations-fixture", src)
	note := filepath.Join(src, "personal", "projects", "projects.md")
	if err := os.Chmod(note, 0o444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(note, 0o644)
	code, _, errb := runMain(t, "--fix", src)
	if code != 2 || !strings.Contains(errb, "sblint: fix:") {
		t.Fatalf("unwritable note must abort the fix: code=%d err=%q", code, errb)
	}
}

func TestReportWriteError(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "_system"), []byte("x"), 0o644)
	code, _, errb := runMain(t, "--report", src)
	if code != 2 || !strings.Contains(errb, "cannot write report") {
		t.Fatalf("code=%d err=%q", code, errb)
	}
}

func TestFixChangedGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	src := t.TempDir()
	copyTree(t, "../../testdata/clean-fixture", src)
	runGit(t, src, "init", "-q")
	runGit(t, src, "config", "user.email", "t@t")
	runGit(t, src, "config", "user.name", "t")

	// Commit a stale children_counts, then touch one of its entries.
	note := filepath.Join(src, "personal", "people", "jane-doe", "jane-doe.md")
	broken := strings.Replace(readFile(t, note), "children_counts: {meeting: 2}", "children_counts: {}", 1)
	os.WriteFile(note, []byte(broken), 0o644)
	runGit(t, src, "add", "-A")
	runGit(t, src, "commit", "-qm", "drift")

	entry := filepath.Join(src, "personal", "people", "jane-doe", "meeting", "2026", "2026-01-01-kickoff.md")
	os.WriteFile(entry, []byte(readFile(t, entry)+"\n"), 0o644)

	code, out, errb := runMain(t, "--changed", "--fix", src)
	if code != 0 {
		t.Fatalf("fix must clear the changed-subtree R19 finding, code=%d", code)
	}
	if !strings.Contains(errb, "fixed 1 file(s):") || !strings.Contains(errb, "personal/people/jane-doe/jane-doe.md") {
		t.Errorf("expected jane-doe.md to be fixed, stderr=%q", errb)
	}
	if strings.Contains(out, "R19") {
		t.Errorf("R19 finding for the changed subtree must be fixed, got:\n%s", out)
	}

	code, _, errb = runMain(t, "--changed", "--fix", src)
	if code != 0 || !strings.Contains(errb, "fixed 0 file(s)") {
		t.Fatalf("second changed+fix must be a no-op: code=%d stderr=%q", code, errb)
	}
}
