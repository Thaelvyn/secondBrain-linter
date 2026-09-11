package rules

import (
	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule17: files outside known top-level scopes. Allowed at the vault root:
// scope dirs from the taxonomy (personal, work, goals, reference), the
// structural folders (_system, daily-logs, calendar, inbox, attachments),
// and the files todos.md / README.md. Dotfiles/dirs are ignored at scan
// time. Unknown root *folders* are reported by rule 4 instead, so this rule
// fires on stray root files only (no double-reporting).
func rule17(c *Context) {
	for _, f := range c.Vault.Files {
		if topLevelDir(f.Path) != "" {
			continue
		}
		if !knownRootFile[f.Name] {
			c.Add(17, vault.SeverityError, f.Path, "file outside known top-level scopes; allowed root files: todos.md, README.md")
		}
	}
}
