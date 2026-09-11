package rules

import (
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule01: folder and file slugs are lowercase, kebab-case, ASCII, no spaces.
// Exempt: dotfiles/dot-dirs (skipped at scan), README.md anywhere, and
// underscore-prefixed vault-internal names (_system, _shared) which are
// ADR-sanctioned structural conventions.
func rule01(c *Context) {
	for _, f := range c.Vault.Files {
		if f.Name == "README.md" || isDotName(f.Name) || strings.HasPrefix(f.Name, "_") {
			continue
		}
		if !validSlug(f.Name) {
			c.Add(1, vault.SeverityError, f.Path, "filename slug must be lowercase kebab-case ASCII without spaces")
		}
	}
	for _, d := range c.Vault.Dirs {
		base := lastSegment(d)
		if isDotName(base) || strings.HasPrefix(base, "_") {
			continue
		}
		if !validSlug(base) {
			c.Add(1, vault.SeverityError, d, "folder slug must be lowercase kebab-case ASCII without spaces")
		}
	}
}

func isDotName(s string) bool { return len(s) > 0 && s[0] == '.' }
