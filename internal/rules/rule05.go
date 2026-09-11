package rules

import (
	"strings"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule05: entry required frontmatter fields are present (ADR 0002 list).
func rule05(c *Context) {
	for _, ei := range c.Entries {
		var missing []string
		for _, k := range requiredEntryFields {
			if _, ok := ei.File.Frontmatter[k]; !ok {
				missing = append(missing, k)
			}
		}
		if len(missing) > 0 {
			c.Add(5, vault.SeverityError, ei.File.Path, "entry missing required frontmatter field(s): "+strings.Join(missing, ", "))
		}
	}
}
