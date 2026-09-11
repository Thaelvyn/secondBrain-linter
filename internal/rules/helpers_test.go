package rules

import (
	"testing"

	"github.com/Thaelvyn/secondBrain-linter/internal/taxonomy"
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// lintVault scans files into a temp vault, runs the full rule set and returns
// the sorted findings.
func lintVault(t *testing.T, files map[string]string, tax *taxonomy.Taxonomy) []vault.Finding {
	t.Helper()
	v, err := scanTestVault(t, files)
	if err != nil {
		t.Fatal(err)
	}
	c := New(v, tax)
	Run(c)
	return sortedFindings(c.Findings)
}

// byRule keeps only findings for one rule.
func byRule(fs []vault.Finding, rule int) []vault.Finding {
	var out []vault.Finding
	for _, f := range fs {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

// scanTestVaultDir is scanTestVault without the Scan: it returns the temp root.
func scanTestVaultDir(t *testing.T, files map[string]string) (string, error) {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		p := dir + "/" + rel
		if err := mkdirAll(p); err != nil {
			return "", err
		}
		if err := writeFile(p, content); err != nil {
			return "", err
		}
	}
	return dir, nil
}
