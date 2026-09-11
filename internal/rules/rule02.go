package rules

import (
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule02: entry filenames (and daily-log files) match {YYYY-MM-DD}-{slug}.md.
// Folder notes are exempt.
func rule02(c *Context) {
	for _, ei := range c.Entries {
		if isFolderNote(ei.File) {
			continue
		}
		if !datePrefixRe.MatchString(ei.File.Name) {
			c.Add(2, vault.SeverityError, ei.File.Path, "entry filename must match {YYYY-MM-DD}-{slug}.md")
		}
	}
	for _, f := range c.Vault.Files {
		if !f.Markdown || !strings.HasPrefix(f.Path, "daily-logs/") {
			continue
		}
		if isFolderNote(f) {
			continue
		}
		if !datePrefixRe.MatchString(f.Name) {
			c.Add(2, vault.SeverityError, f.Path, "daily-log filename must match {YYYY-MM-DD}-{slug}.md")
		}
	}
}
