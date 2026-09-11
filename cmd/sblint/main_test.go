package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	if code != 0 || !strings.HasPrefix(out, "sblint ") {
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
	if !strings.Contains(out, "23 errors, 2 warnings") {
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
	if len(got.Findings) != 25 {
		t.Errorf("findings = %d", len(got.Findings))
	}
	if got.Counts["error"] != 23 || got.Counts["warning"] != 2 {
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
