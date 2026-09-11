package rules

import (
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule20: every entry's frontmatter source resolves to an existing
// vault-relative file (ADR 0004 rule 20, error). Empty values and the literal
// `manual` (case-insensitive, /sb-add entries per ADR 0002 addendum) are
// skipped. Resolution: the path as-is relative to the vault root, an optional
// leading "./", and `<path>.md` when the path carries no extension.
func rule20(c *Context) {
	for _, ei := range c.Entries {
		for _, raw := range stringList(ei.File.Frontmatter, "source") {
			src := strings.TrimSpace(raw)
			if src == "" || strings.EqualFold(src, "manual") {
				continue
			}
			if !sourceExists(c.Vault, src) {
				c.Add(20, vault.SeverityError, ei.File.Path, "source file does not exist: "+src)
			}
		}
	}
}

// sourceExists resolves a frontmatter source path against the scanned vault.
func sourceExists(v *vault.Vault, raw string) bool {
	p := strings.TrimPrefix(strings.TrimSpace(raw), "./")
	if p == "" {
		return false
	}
	if _, ok := v.ByPath[p]; ok {
		return true
	}
	if !strings.Contains(lastSegment(p), ".") {
		if _, ok := v.ByPath[p+".md"]; ok {
			return true
		}
	}
	return false
}
