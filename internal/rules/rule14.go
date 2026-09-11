package rules

import (
	"fmt"

	"github.com/Thaelvyn/secondBrain-linter/internal/vault"
)

// rule14: orphan entity folder - an entity-type folder note with no entries
// beneath it. Stubs are allowed but flagged as warnings.
func rule14(c *Context) {
	for _, dir := range sortedKeys(c.FolderNotes) {
		f := c.FolderNotes[dir]
		if t, ok := fmString(f.Frontmatter, "type"); ok && t == "entity" {
			if c.EntriesBeneath[dir] == 0 {
				c.Add(14, vault.SeverityWarning, dir, fmt.Sprintf("orphan entity folder %q: entity folder note present but no entries beneath it (stub allowed, flagged)", dir))
			}
		}
	}
}
