// Package report renders findings as console text, JSON, and timestamped
// report files under <vault>/_system/status/lint/.
package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// DisplayRooted prefixes a relative path with the vault root as given on the
// command line ("sblint julien" -> "julien/This is a test.md"). A root of
// "." keeps paths relative.
func DisplayRooted(rootArg, rel string) string {
	r := strings.TrimSuffix(filepath.ToSlash(rootArg), "/")
	if r == "." || r == "" {
		return rel
	}
	return r + "/" + rel
}

// Console renders the console format: one line per finding
// ([error] R1 path: message), then a summary line.
func Console(rootArg string, findings []vault.Finding) string {
	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "[%s] R%d %s: %s\n", f.Severity, f.Rule, DisplayRooted(rootArg, f.Path), f.Message)
	}
	errors, warnings := Counts(findings)
	fmt.Fprintf(&b, "%d error%s, %d warning%s\n", errors, plural(errors), warnings, plural(warnings))
	return b.String()
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Counts splits findings by severity.
func Counts(findings []vault.Finding) (errors, warnings int) {
	for _, f := range findings {
		switch f.Severity {
		case vault.SeverityError:
			errors++
		case vault.SeverityWarning:
			warnings++
		}
	}
	return errors, warnings
}

type jsonFinding struct {
	Rule     int    `json:"rule"`
	Severity string `json:"severity"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

type jsonCounts struct {
	Errors   int `json:"error"`
	Warnings int `json:"warning"`
}

type jsonReport struct {
	Vault    string        `json:"vault"`
	Findings []jsonFinding `json:"findings"`
	Counts   jsonCounts    `json:"counts"`
}

// JSON renders the machine-readable report.
func JSON(rootArg string, findings []vault.Finding) ([]byte, error) {
	r := jsonReport{Vault: rootArg}
	r.Counts.Errors, r.Counts.Warnings = Counts(findings)
	for _, f := range findings {
		r.Findings = append(r.Findings, jsonFinding{
			Rule:     f.Rule,
			Severity: string(f.Severity),
			Path:     DisplayRooted(rootArg, f.Path),
			Message:  f.Message,
		})
	}
	if r.Findings == nil {
		r.Findings = []jsonFinding{}
	}
	return json.Marshal(r)
}

// WriteFile writes console-format text to
// <vault>/_system/status/lint/sblint-YYYYmmdd-HHMMSS.txt and returns the
// created path.
func WriteFile(root, text string) (string, error) {
	dir := filepath.Join(root, "_system", "status", "lint")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := "sblint-" + time.Now().Format("20060102-150405") + ".txt"
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		return "", err
	}
	return p, nil
}
